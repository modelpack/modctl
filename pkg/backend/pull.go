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
	"encoding/json"
	"fmt"
	"io"

	retry "github.com/avast/retry-go/v4"
	sha256 "github.com/minio/sha256-simd"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	internalpb "github.com/CloudNativeAI/modctl/internal/pb"
	"github.com/CloudNativeAI/modctl/pkg/backend/remote"
	"github.com/CloudNativeAI/modctl/pkg/config"
	"github.com/CloudNativeAI/modctl/pkg/storage"
)

// Pull pulls an artifact from a registry.
func (b *backend) Pull(ctx context.Context, target string, cfg *config.Pull) error {
	logrus.Infof("pull: starting pull operation for target %s [config: %+v]", target, cfg)

	// pullByDragonfly is called if a Dragonfly endpoint is specified in the configuration.
	if cfg.DragonflyEndpoint != "" {
		logrus.Infof("pull: using dragonfly for target %s", target)
		return b.pullByDragonfly(ctx, target, cfg)
	}

	// parse the repository and tag from the target.
	ref, err := ParseReference(target)
	if err != nil {
		return fmt.Errorf("failed to parse the target: %w", err)
	}

	repo, tag := ref.Repository(), ref.Tag()
	src, err := remote.New(repo, remote.WithPlainHTTP(cfg.PlainHTTP), remote.WithInsecure(cfg.Insecure), remote.WithProxy(cfg.Proxy))
	if err != nil {
		return fmt.Errorf("failed to create the remote client: %w", err)
	}

	manifestDesc, manifestReader, err := src.Manifests().FetchReference(ctx, tag)
	if err != nil {
		// fallback to fetch the manifest without proxy.
		if cfg.Proxy != "" {
			fmt.Printf("Failed to fetch the manifest with proxy, fallback to fetch without proxy, error: %v\n", err)
			cfg.Proxy = ""
			src, err = remote.New(repo, remote.WithPlainHTTP(cfg.PlainHTTP), remote.WithInsecure(cfg.Insecure), remote.WithProxy(cfg.Proxy))
			if err != nil {
				return fmt.Errorf("failed to create the remote client: %w", err)
			}
			manifestDesc, manifestReader, err = src.Manifests().FetchReference(ctx, tag)
			if err != nil {
				return fmt.Errorf("failed to fetch the manifest: %w", err)
			}
		} else {
			return fmt.Errorf("failed to fetch the manifest: %w", err)
		}
	}

	defer manifestReader.Close()

	var manifest ocispec.Manifest
	if err := json.NewDecoder(manifestReader).Decode(&manifest); err != nil {
		return fmt.Errorf("failed to decode the manifest: %w", err)
	}

	logrus.Debugf("pull: loaded manifest for target %s [manifest: %+v]", target, manifest)

	// TODO: need refactor as currently use a global flag to control the progress bar render.
	if cfg.DisableProgress {
		internalpb.SetDisableProgress(true)
	}

	// create the progress bar to track the progress of push.
	pb := internalpb.NewProgressBar(cfg.ProgressWriter)
	pb.Start()
	defer pb.Stop()

	// copy the image to the destination, there are three steps:
	// 1. copy the layers.
	// 2. copy the config.
	// 3. copy the manifest.
	// note: the order is important, manifest should be pushed at last.

	// copy the layers.
	dst := b.store
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(cfg.Concurrency)

	var fn func(desc ocispec.Descriptor) error
	if cfg.ExtractFromRemote {
		fn = func(desc ocispec.Descriptor) error {
			return pullAndExtractFromRemote(gctx, pb, internalpb.NormalizePrompt("Pulling blob"), src, cfg.ExtractDir, desc)
		}
	} else {
		fn = func(desc ocispec.Descriptor) error {
			return pullIfNotExist(gctx, pb, internalpb.NormalizePrompt("Pulling blob"), src, dst, desc, repo, tag)
		}
	}

	logrus.Infof("pull: processing layers for target %s [count: %d]", target, len(manifest.Layers))
	for _, layer := range manifest.Layers {
		g.Go(func() error {
			select {
			case <-gctx.Done():
				return gctx.Err()
			default:
			}

			return retry.Do(func() error {
				logrus.Debugf("pull: processing layer %s", layer.Digest)
				// call the before hook.
				cfg.Hooks.BeforePullLayer(layer, manifest)
				err := fn(layer)
				// call the after hook.
				cfg.Hooks.AfterPullLayer(layer, err)
				if err != nil {
					err = fmt.Errorf("pull: failed to process layer %s: %w", layer.Digest, err)
					logrus.Error(err)
				}

				return err
			}, append(defaultRetryOpts, retry.Context(gctx))...)
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed to pull blob to local: %w", err)
	}

	logrus.Infof("pull: successfully processed layers [count: %d]", len(manifest.Layers))

	// return earlier if extract from remote is enabled as config and manifest
	// are not needed for this operation.
	if cfg.ExtractFromRemote {
		return nil
	}

	// copy the config.
	if err := retry.Do(func() error {
		return pullIfNotExist(ctx, pb, internalpb.NormalizePrompt("Pulling config"), src, dst, manifest.Config, repo, tag)
	}, append(defaultRetryOpts, retry.Context(ctx))...); err != nil {
		return fmt.Errorf("failed to pull config to local: %w", err)
	}

	// copy the manifest.
	if err := retry.Do(func() error {
		return pullIfNotExist(ctx, pb, internalpb.NormalizePrompt("Pulling manifest"), src, dst, manifestDesc, repo, tag)
	}, append(defaultRetryOpts, retry.Context(ctx))...); err != nil {
		return fmt.Errorf("failed to pull manifest to local: %w", err)
	}

	// export the target model artifact to the output directory if needed.
	if cfg.ExtractDir != "" {
		// set the concurrency to 1 because the pull already has concurrency control.
		extractCfg := &config.Extract{Concurrency: 1, Output: cfg.ExtractDir}
		if err := exportModelArtifact(ctx, dst, manifest, repo, extractCfg); err != nil {
			return fmt.Errorf("failed to export the artifact to the output directory: %w", err)
		}
		logrus.Infof("pull: successfully pulled and extracted artifact %s", target)
	}

	logrus.Infof("pull: successfully pulled artifact %s", target)
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
	hash := sha256.New()
	reader = io.TeeReader(reader, hash)

	// push the content to the destination, and wrap the content reader for progress bar,
	// manifest should use dst.Manifests().Push, others should use dst.Blobs().Push.
	if desc.MediaType == ocispec.MediaTypeImageManifest {
		// check whether the content exists in the destination storage.
		exist, err := dst.StatManifest(ctx, repo, desc.Digest.String())
		if err != nil {
			err = fmt.Errorf("failed to check manifest %s, err: %w", desc.Digest.String(), err)
			pb.Abort(desc.Digest.String(), err)
			return err
		}

		if exist {
			pb.Complete(desc.Digest.String(), fmt.Sprintf("%s %s", internalpb.NormalizePrompt("Skipped blob"), desc.Digest.String()))
			return nil
		}

		body, err := io.ReadAll(reader)
		if err != nil {
			err = fmt.Errorf("failed to read manifest %s, err: %w", desc.Digest.String(), err)
			pb.Abort(desc.Digest.String(), err)
			return err
		}

		if _, err := dst.PushManifest(ctx, repo, tag, body); err != nil {
			err = fmt.Errorf("failed to store manifest %s, err: %w", desc.Digest.String(), err)
			pb.Abort(desc.Digest.String(), err)
			return err
		}
	} else {
		exist, err := dst.StatBlob(ctx, repo, desc.Digest.String())
		if err != nil {
			err = fmt.Errorf("failed to check blob %s, err: %w", desc.Digest.String(), err)
			pb.Abort(desc.Digest.String(), err)
			return err
		}

		if exist {
			pb.Complete(desc.Digest.String(), fmt.Sprintf("%s %s", internalpb.NormalizePrompt("Skipped blob"), desc.Digest.String()))
			return nil
		}

		if _, _, err := dst.PushBlob(ctx, repo, reader, desc); err != nil {
			err = fmt.Errorf("failed to store blob %s, err: %w", desc.Digest.String(), err)
			pb.Abort(desc.Digest.String(), err)
			return err
		}
	}

	// validate the digest of the blob.
	if err := validateDigest(desc.Digest.String(), hash.Sum(nil)); err != nil {
		err = fmt.Errorf("failed to validate the digest of the blob %s, err: %w", desc.Digest.String(), err)
		pb.Abort(desc.Digest.String(), err)
		return err
	}

	return nil
}

// pullAndExtractFromRemote pulls the layer and extract it to the target output path directly,
// and will not store the layer to the local storage.
func pullAndExtractFromRemote(ctx context.Context, pb *internalpb.ProgressBar, prompt string, src *remote.Repository, outputDir string, desc ocispec.Descriptor) error {
	// fetch the content from the source storage.
	content, err := src.Fetch(ctx, desc)
	if err != nil {
		return fmt.Errorf("failed to fetch the content from source: %w", err)
	}
	defer content.Close()

	reader := pb.Add(prompt, desc.Digest.String(), desc.Size, content)
	hash := sha256.New()
	reader = io.TeeReader(reader, hash)

	if err := extractLayer(desc, outputDir, reader); err != nil {
		err = fmt.Errorf("failed to extract the blob %s to output directory: %w", desc.Digest.String(), err)
		pb.Abort(desc.Digest.String(), err)
		return err
	}

	// validate the digest of the blob.
	if err := validateDigest(desc.Digest.String(), hash.Sum(nil)); err != nil {
		err = fmt.Errorf("failed to validate the digest of the blob %s, err: %w", desc.Digest.String(), err)
		pb.Abort(desc.Digest.String(), err)
		return err
	}

	return nil
}

// validateDigest validates the hash digest whether matches the expected digest.
func validateDigest(digest string, hash []byte) error {
	if digest == "" {
		return fmt.Errorf("digest is empty")
	}

	if len(hash) != sha256.Size {
		return fmt.Errorf("invalid hash length")
	}

	if digest != fmt.Sprintf("sha256:%x", hash) {
		return fmt.Errorf("actual digest %s does not match the expected digest %s", fmt.Sprintf("sha256:%x", hash), digest)
	}

	return nil
}
