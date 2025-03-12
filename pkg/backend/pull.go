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

	internalpb "github.com/CloudNativeAI/modctl/internal/pb"
	"github.com/CloudNativeAI/modctl/pkg/archiver"
	"github.com/CloudNativeAI/modctl/pkg/config"
	"github.com/CloudNativeAI/modctl/pkg/storage"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"golang.org/x/sync/errgroup"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"
)

// Pull pulls an artifact from a registry.
func (b *backend) Pull(ctx context.Context, target string, cfg *config.Pull) error {
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
	if cfg.Proxy != "" {
		proxyURL, err := url.Parse(cfg.Proxy)
		if err != nil {
			return fmt.Errorf("failed to parse the proxy URL: %w", err)
		}

		httpClient.Transport = retry.NewTransport(&http.Transport{
			Proxy: http.ProxyURL(proxyURL),
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.Insecure,
			},
		})
	}

	src.Client = &auth.Client{
		Cache:      auth.NewCache(),
		Credential: credentials.Credential(credStore),
		Client:     httpClient,
	}

	if cfg.PlainHTTP {
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
	pb := internalpb.NewProgressBar()
	pb.Start()
	defer pb.Stop()

	// copy the image to the destination, there are three steps:
	// 1. copy the layers.
	// 2. copy the config.
	// 3. copy the manifest.
	// note: the order is important, manifest should be pushed at last.

	// copy the layers.
	dst := b.store
	g := &errgroup.Group{}
	g.SetLimit(cfg.Concurrency)

	var fn func(desc ocispec.Descriptor) error
	if cfg.ExtractFromRemote {
		fn = func(desc ocispec.Descriptor) error {
			return pullAndExtractFromRemote(ctx, pb, internalpb.NormalizePrompt("Pulling blob"), src, cfg.ExtractDir, desc)
		}
	} else {
		fn = func(desc ocispec.Descriptor) error {
			return pullIfNotExist(ctx, pb, internalpb.NormalizePrompt("Pulling blob"), src, dst, desc, repo, tag)
		}
	}

	for _, layer := range manifest.Layers {
		g.Go(func() error { return fn(layer) })
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed to pull blob to local: %w", err)
	}

	// return earlier if extract from remote is enabled as config and manifest
	// are not needed for this operation.
	if cfg.ExtractFromRemote {
		return nil
	}

	// copy the config.
	if err := pullIfNotExist(ctx, pb, internalpb.NormalizePrompt("Pulling config"), src, dst, manifest.Config, repo, tag); err != nil {
		return fmt.Errorf("failed to pull config to local: %w", err)
	}

	// copy the manifest.
	if err := pullIfNotExist(ctx, pb, internalpb.NormalizePrompt("Pulling manifest"), src, dst, manifestDesc, repo, tag); err != nil {
		return fmt.Errorf("failed to pull manifest to local: %w", err)
	}

	// export the target model artifact to the output directory if needed.
	if cfg.ExtractDir != "" {
		if err := exportModelArtifact(ctx, dst, manifest, repo, cfg.ExtractDir); err != nil {
			return fmt.Errorf("failed to export the artifact to the output directory: %w", err)
		}
	}

	return nil
}

// pullIfNotExist copies the content from the src storage to the dst storage if the content does not exist.
func pullIfNotExist(ctx context.Context, pb *internalpb.ProgressBar, prompt string, src *remote.Repository, dst storage.Storage, desc ocispec.Descriptor, repo, tag string) error {
	// fetch the content from the source storage.
	content, err := src.Fetch(ctx, desc)
	if err != nil {
		return err
	}

	defer content.Close()

	reader := pb.Add(prompt, desc.Digest.String(), desc.Size, content)

	// push the content to the destination, and wrap the content reader for progress bar,
	// manifest should use dst.Manifests().Push, others should use dst.Blobs().Push.
	if desc.MediaType == ocispec.MediaTypeImageManifest {
		// check whether the content exists in the destination storage.
		exist, err := dst.StatManifest(ctx, repo, desc.Digest.String())
		if err != nil {
			pb.Complete(desc.Digest.String(), fmt.Sprintf("Failed to check manifest %s, err: %v", desc.Digest.String(), err))
			return err
		}

		if exist {
			pb.Complete(desc.Digest.String(), fmt.Sprintf("%s %s", internalpb.NormalizePrompt("Skipped blob"), desc.Digest.String()))
			return nil
		}

		body, err := io.ReadAll(reader)
		if err != nil {
			pb.Complete(desc.Digest.String(), fmt.Sprintf("Failed to read manifest %s, err: %v", desc.Digest.String(), err))
			return err
		}

		if _, err := dst.PushManifest(ctx, repo, tag, body); err != nil {
			pb.Complete(desc.Digest.String(), fmt.Sprintf("Failed to store manifest %s, err: %v", desc.Digest.String(), err))
			return err
		}
	} else {
		exist, err := dst.StatBlob(ctx, repo, desc.Digest.String())
		if err != nil {
			pb.Complete(desc.Digest.String(), fmt.Sprintf("Failed to check blob %s, err: %v", desc.Digest.String(), err))
			return err
		}

		if exist {
			pb.Complete(desc.Digest.String(), fmt.Sprintf("%s %s", internalpb.NormalizePrompt("Skipped blob"), desc.Digest.String()))
			return nil
		}

		if _, _, err := dst.PushBlob(ctx, repo, reader, desc); err != nil {
			pb.Complete(desc.Digest.String(), fmt.Sprintf("Failed to store blob %s, err: %v", desc.Digest.String(), err))
			return err
		}
	}

	return nil
}

// pullAndExtractFromRemote pulls the layer and extract it to the target output path directly,
// and will not store the layer to the local storage.
func pullAndExtractFromRemote(ctx context.Context, pb *internalpb.ProgressBar, prompt string, src *remote.Repository, output string, desc ocispec.Descriptor) error {
	// fetch the content from the source storage.
	content, err := src.Fetch(ctx, desc)
	if err != nil {
		return fmt.Errorf("failed to fetch the content from source: %w", err)
	}

	defer content.Close()

	reader := pb.Add(prompt, desc.Digest.String(), desc.Size, content)
	if err := archiver.Untar(reader, output); err != nil {
		pb.Complete(desc.Digest.String(), fmt.Sprintf("Failed to pull and extract blob %s from remote, err: %v", desc.Digest.String(), err))
		return fmt.Errorf("failed to untar the blob %s: %w", desc.Digest.String(), err)
	}

	return nil
}
