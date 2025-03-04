/*
 *     Copyright 2024 The CNAI Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package backend

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/CloudNativeAI/modctl/pkg/storage"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"golang.org/x/sync/errgroup"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"
)

// Pull pulls an artifact from a registry.
func (b *backend) Pull(ctx context.Context, target string, opts ...Option) error {
	// apply options.
	options := &Options{}
	for _, opt := range opts {
		opt(options)
	}

	// parse the repository and tag from the target.
	ref, err := ParseReference(target)
	if err != nil {
		return fmt.Errorf("failed to parse the target: %w", err)
	}

	repo, tag := ref.Repository(), ref.Tag()

	// create the src storage from the remote repository.
	src, err := remote.NewRepository(target)
	if err != nil {
		return fmt.Errorf("failed to create remote repository: %w", err)
	}

	// gets the credentials store.
	credStore, err := credentials.NewStoreFromDocker(credentials.StoreOptions{AllowPlaintextPut: true})
	if err != nil {
		return fmt.Errorf("failed to create credential store: %w", err)
	}

	// create the http client.
	httpClient := &http.Client{}
	if options.proxy != "" {
		proxyURL, err := url.Parse(options.proxy)
		if err != nil {
			return fmt.Errorf("failed to parse the proxy URL: %w", err)
		}

		httpClient.Transport = retry.NewTransport(&http.Transport{
			Proxy: http.ProxyURL(proxyURL),
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: options.insecure,
			},
		})
	}

	src.Client = &auth.Client{
		Cache:      auth.NewCache(),
		Credential: credentials.Credential(credStore),
		Client:     httpClient,
	}

	if options.plainHTTP {
		src.PlainHTTP = true
	}

	manifestDesc, manifestReader, err := src.Manifests().FetchReference(ctx, tag)
	if err != nil {
		return fmt.Errorf("failed to fetch the manifest: %w", err)
	}

	defer manifestReader.Close()

	var manifest ocispec.Manifest
	if err := json.NewDecoder(manifestReader).Decode(&manifest); err != nil {
		return fmt.Errorf("failed to decode the manifest: %w", err)
	}

	// create the progress bar to track the progress of push.
	pb := NewProgressBar()
	defer pb.Wait()

	// copy the image to the destination, there are three steps:
	// 1. copy the layers.
	// 2. copy the config.
	// 3. copy the manifest.
	// note: the order is important, manifest should be pushed at last.

	// copy the layers.
	dst := b.store
	g := &errgroup.Group{}
	g.SetLimit(options.concurrency)
	for _, layer := range manifest.Layers {
		g.Go(func() error { return pullIfNotExist(ctx, pb, promptCopyingBlob, src, dst, layer, repo, tag) })
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed to pull blob to local: %w", err)
	}

	// copy the config.
	if err := pullIfNotExist(ctx, pb, promptCopyingConfig, src, dst, manifest.Config, repo, tag); err != nil {
		return fmt.Errorf("failed to pull config to local: %w", err)
	}

	// copy the manifest.
	if err := pullIfNotExist(ctx, pb, promptCopyingManifest, src, dst, manifestDesc, repo, tag); err != nil {
		return fmt.Errorf("failed to pull manifest to local: %w", err)
	}

	// export the target model artifact to the output directory if needed.
	if options.output != "" {
		if err := exportModelArtifact(ctx, dst, manifest, repo, options.output); err != nil {
			return fmt.Errorf("failed to export the artifact to the output directory: %w", err)
		}
	}

	return nil
}

// pullIfNotExist copies the content from the src storage to the dst storage if the content does not exist.
func pullIfNotExist(ctx context.Context, pb *ProgressBar, prompt string, src *remote.Repository, dst storage.Storage, desc ocispec.Descriptor, repo, tag string) error {
	// check whether the content exists in the destination storage.
	exist, _ := dst.StatBlob(ctx, repo, desc.Digest.String())
	if exist {
		pb.PrintMessage(prompt, desc, "skipped: already exists")
		return nil
	}

	// fetch the content from the source storage.
	content, err := src.Fetch(ctx, desc)
	if err != nil {
		return fmt.Errorf("failed to fetch the content from source: %w", err)
	}

	defer content.Close()
	// push the content to the destination, and wrap the content reader for progress bar,
	// manifest should use dst.Manifests().Push, others should use dst.Blobs().Push.
	if desc.MediaType == ocispec.MediaTypeImageManifest {
		body, err := io.ReadAll(pb.Add(prompt, desc, content))
		if err != nil {
			pb.Abort(desc)
			return fmt.Errorf("failed to read the manifest: %w", err)
		}

		if _, err := dst.PushManifest(ctx, repo, tag, body); err != nil {
			return err
		}
	} else {
		if _, _, err := dst.PushBlob(ctx, repo, pb.Add(prompt, desc, content)); err != nil {
			pb.Abort(desc)
			return err
		}
	}

	return nil
}
