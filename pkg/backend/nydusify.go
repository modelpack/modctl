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
	"os/exec"

	"github.com/sirupsen/logrus"

	"github.com/CloudNativeAI/modctl/pkg/config"
)

const (
	// nydusImageTagSuffix is the suffix for the nydus image tag.
	nydusImageTagSuffix = "_nydus_v2"
)

// Nydusify is a function that converts a given model artifact to a nydus image.
func (b *backend) Nydusify(ctx context.Context, source string, cfg *config.Root) (string, error) {
	logrus.Infof("Nydusifying the model artifact %s", source)

	target := source + nydusImageTagSuffix
	cmd := exec.CommandContext(
		ctx,
		"nydusify",
		"convert",
		"--source-backend-type",
		"model-artifact",
		"--compressor",
		"lz4_block",
		"--fs-version",
		"5",
		"--source",
		source,
		"--target",
		target,
		"--log-level",
		cfg.LogLevel,
		"--log-file",
		cfg.LogFile,
	)
	if err := cmd.Run(); err != nil {
		logrus.Errorf("failed to nydusify the model artifact %s: %v", source, err)
		return "", err
	}

	logrus.Infof("Nydusifyed the model artifact %s successfully", source)
	return target, nil
}
