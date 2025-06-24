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
	"fmt"

	"github.com/sirupsen/logrus"
)

// Remove removes the target from the storage, notice that remove only removes the manifest,
// the blobs may still be used by other manifests, so should use prune to remove the unused blobs.
func (b *backend) Remove(ctx context.Context, target string) (string, error) {
	logrus.Infof("removing target %s", target)
	ref, err := ParseReference(target)
	if err != nil {
		return "", fmt.Errorf("failed to parse target: %w", err)
	}

	// if the reference is a tag, it will only untagged this manifest,
	// but if provided a digest, it will remove the manifest and all tags referencing it.
	repo, reference := ref.Repository(), ref.Tag()
	if ref.Digest() != "" {
		reference = ref.Digest()
	}

	if reference == "" {
		return "", fmt.Errorf("invalid reference, tag or digest must be provided")
	}

	if err := b.store.DeleteManifest(ctx, repo, reference); err != nil {
		return "", fmt.Errorf("failed to delete manifest %s: %w", reference, err)
	}

	logrus.Infof("manifest %s removed", reference)

	return reference, nil
}
