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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	retry "github.com/avast/retry-go/v4"
	godigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	internalpb "github.com/modelpack/modctl/internal/pb"
	"github.com/modelpack/modctl/pkg/backend/remote"
	"github.com/modelpack/modctl/pkg/config"
	"github.com/modelpack/modctl/pkg/storage"
)

// Push pushes the image to the registry.
func (b *backend) Push(ctx context.Context, target string, cfg *config.Push) error {
	logrus.Infof("push: starting push operation for target %s [config: %+v]", target, cfg)
	// parse the repository and tag from the target.
	ref, err := ParseReference(target)
	if err != nil {
		return fmt.Errorf("failed to parse the target: %w", err)
	}

	repo, tag := ref.Repository(), ref.Tag()

	// create the src storage from the image storage path.
	src := b.store
	dst, err := remote.New(repo, remote.WithPlainHTTP(cfg.PlainHTTP), remote.WithInsecure(cfg.Insecure))
	if err != nil {
		return fmt.Errorf("failed to create the destination: %w", err)
	}

	manifestRaw, _, err := src.PullManifest(ctx, repo, tag)
	if err != nil {
		return fmt.Errorf("failed to pull the manifest: %w", err)
	}

	logrus.Debugf("push: loaded manifest for target %s [manifest: %s]", target, string(manifestRaw))

	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
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
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(cfg.Concurrency)

	logrus.Infof("push: processing layers for target %s [count: %d]", target, len(manifest.Layers))
	for _, layer := range manifest.Layers {
		g.Go(func() error {
			select {
			case <-gctx.Done():
				return gctx.Err()
			default:
			}

			return retry.Do(func() error {
				logrus.Debugf("push: processing layer %s", layer.Digest)
				if err := pushIfNotExist(gctx, pb, internalpb.NormalizePrompt("Copying blob"), src, dst, layer, repo, tag); err != nil {
					return err
				}
				logrus.Debugf("push: successfully processed layer %s", layer.Digest)
				return nil
			}, append(defaultRetryOpts, retry.Context(gctx))...)
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed to push blob to remote: %w", err)
	}

	// copy the config.
	if err := retry.Do(func() error {
		return pushIfNotExist(ctx, pb, internalpb.NormalizePrompt("Copying config"), src, dst, manifest.Config, repo, tag)
	}, append(defaultRetryOpts, retry.Context(ctx))...); err != nil {
		return fmt.Errorf("failed to push config to remote: %w", err)
	}

	// copy the manifest.
	if err := retry.Do(func() error {
		return pushIfNotExist(ctx, pb, internalpb.NormalizePrompt("Copying manifest"), src, dst, ocispec.Descriptor{
			MediaType: manifest.MediaType,
			Size:      int64(len(manifestRaw)),
			Digest:    godigest.FromBytes(manifestRaw),
			Data:      manifestRaw,
		}, repo, tag)
	}, append(defaultRetryOpts, retry.Context(ctx))...); err != nil {
		return fmt.Errorf("failed to push manifest to remote: %w", err)
	}

	logrus.Infof("push: successfully pushed artifact %s", target)
	return nil
}

// pushIfNotExist copies the content from the src storage to the dst storage if the content does not exist.
func pushIfNotExist(ctx context.Context, pb *internalpb.ProgressBar, prompt string, src storage.Storage, dst *remote.Repository, desc ocispec.Descriptor, repo, tag string) error {
	// check whether the content exists in the destination storage.
	exist, err := dst.Exists(ctx, desc)
	if err != nil {
		return err
	}

	if exist {
		pb.Add(prompt, desc.Digest.String(), desc.Size, bytes.NewReader([]byte{}))
		// if the descriptor is the manifest, should check the tag existence as well.
		if desc.MediaType == ocispec.MediaTypeImageManifest {
			_, _, err := dst.FetchReference(ctx, tag)
			if err != nil {
				// try to push the tag if error occurred when fetch reference.
				if err := dst.Tag(ctx, desc, tag); err != nil {
					err = fmt.Errorf("failed to push tag %s, err: %w", tag, err)
					pb.Abort(desc.Digest.String(), err)
					return err
				}
			}
		}

		pb.Complete(desc.Digest.String(), fmt.Sprintf("%s %s", internalpb.NormalizePrompt("Skipped blob"), desc.Digest.String()))
		return nil
	}

	// push the content to the destination, and wrap the content reader for progress bar,
	// manifest should use dst.Manifests().Push, others should use dst.Blobs().Push.
	if desc.MediaType == ocispec.MediaTypeImageManifest {
		reader := pb.Add(prompt, desc.Digest.String(), desc.Size, bytes.NewReader(desc.Data))
		if err := dst.Manifests().Push(ctx, desc, reader); err != nil {
			err = fmt.Errorf("failed to push manifest %s, err: %w", desc.Digest.String(), err)
			pb.Abort(desc.Digest.String(), err)
			return err
		}

		// push tag
		if err := dst.Tag(ctx, desc, tag); err != nil {
			err = fmt.Errorf("failed to push tag %s, err: %w", tag, err)
			pb.Abort(desc.Digest.String(), err)
			return err
		}
	} else {
		// fetch the content from the source storage.
		content, err := src.PullBlob(ctx, repo, desc.Digest.String())
		if err != nil {
			return err
		}

		reader := pb.Add(prompt, desc.Digest.String(), desc.Size, content)
		// resolve issue: https://github.com/modelpack/modctl/issues/50
		// wrap the content to the NopCloser, because the implementation of the distribution will
		// always return the error when Close() is called.
		// refer: https://github.com/distribution/distribution/blob/63d3892315c817c931b88779399a8e9142899a8e/registry/storage/filereader.go#L105
		if err := dst.Blobs().Push(ctx, desc, io.NopCloser(reader)); err != nil {
			err = fmt.Errorf("failed to push blob %s, err: %w", desc.Digest.String(), err)
			pb.Abort(desc.Digest.String(), err)
			return err
		}
	}

	return nil
}
