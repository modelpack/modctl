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

package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/modelpack/modctl/pkg/config"
)

// TestIntegration_CLI_ConcurrencyZero tests that a Pull config with Concurrency=0
// fails validation with a concurrency-related error message.
func TestIntegration_CLI_ConcurrencyZero(t *testing.T) {
	cfg := config.NewPull()
	cfg.Concurrency = 0
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "concurrency")
}

// TestIntegration_CLI_ConcurrencyNegative tests that a Pull config with a negative
// Concurrency fails validation with a concurrency-related error message.
func TestIntegration_CLI_ConcurrencyNegative(t *testing.T) {
	cfg := config.NewPull()
	cfg.Concurrency = -1
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "concurrency")
}

// TestIntegration_CLI_ExtractFromRemoteNoDir tests that enabling ExtractFromRemote
// without specifying an ExtractDir fails validation.
func TestIntegration_CLI_ExtractFromRemoteNoDir(t *testing.T) {
	cfg := config.NewPull()
	cfg.ExtractFromRemote = true
	cfg.ExtractDir = ""
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "extract dir")
}
