/*
 *     Copyright 2025 The CNAI Authors
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

	internalpb "github.com/modelpack/modctl/internal/pb"
	"github.com/modelpack/modctl/pkg/backend/build"
	"github.com/modelpack/modctl/pkg/backend/processor"
	"github.com/modelpack/modctl/pkg/config"
)

// Upload uploads the file to a model artifact repository in advance, but will not push config and manifest.
func (b *backend) Upload(ctx context.Context, filepath string, cfg *config.Upload) error {
	logrus.Infof("upload: starting upload operation for file %s [repository: %s]", filepath, cfg.Repo)
	proc := b.getProcessor(filepath, cfg.Raw)
	if proc == nil {
		return fmt.Errorf("failed to get processor for file %s", filepath)
	}

	opts := []build.Option{
		build.WithPlainHTTP(cfg.PlainHTTP),
		build.WithInsecure(cfg.Insecure),
	}
	builder, err := build.NewBuilder(build.OutputTypeRemote, b.store, cfg.Repo, "", opts...)
	if err != nil {
		return fmt.Errorf("failed to create builder: %w", err)
	}

	pb := internalpb.NewProgressBar()
	pb.Start()
	defer pb.Stop()

	if _, err = proc.Process(ctx, builder, ".", processor.WithProgressTracker(pb)); err != nil {
		return fmt.Errorf("failed to process layers: %w", err)
	}

	logrus.Infof("upload: successfully uploaded file %s [repository: %s]", filepath, cfg.Repo)
	return nil
}
