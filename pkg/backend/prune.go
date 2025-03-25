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
)

// Prune prunes the unused blobs and clean up the storage.
func (b *backend) Prune(ctx context.Context, dryRun, removeUntagged bool) error {
	if err := b.store.PerformGC(ctx, dryRun, removeUntagged); err != nil {
		return fmt.Errorf("faile to perform gc: %w", err)
	}

	if err := b.store.PerformPurgeUploads(ctx, dryRun); err != nil {
		return fmt.Errorf("failed to perform purge uploads: %w", err)
	}

	return nil
}
