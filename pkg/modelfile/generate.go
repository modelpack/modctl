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

package modelfile

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type ModelfileGenConfig struct {
	Name               string
	Version            string
	OutputPath         string
	IgnoreUnrecognized bool
	Overwrite          bool
	Arch               string
	Family             string
	Format             string
	Paramsize          string
	Precision          string
	Quantization       string
}

func NewModelfileGenConfig() *ModelfileGenConfig {
	return &ModelfileGenConfig{
		Name:               "",
		Version:            "",
		OutputPath:         "",
		IgnoreUnrecognized: false,
		Overwrite:          false,
		Arch:               "",
		Family:             "",
		Format:             "",
		Paramsize:          "",
		Precision:          "",
		Quantization:       "",
	}
}

func (c *ModelfileGenConfig) Validate() error {
	// if len(c.Name) == 0 {
	// 	return fmt.Errorf("model name is required")
	// }

	if len(c.OutputPath) == 0 {
		return fmt.Errorf("output path is required")
	}

	return nil
}

func RunGenModelfile(ctx context.Context, modelPath string, genConfig *ModelfileGenConfig) error {
	if err := genConfig.Validate(); err != nil {
		return fmt.Errorf("failed to validate modelfile gen config: %w", err)
	}

	// Convert modelPath to absolute path
	realModelPath, err := filepath.Abs(modelPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for model: %w", err)
	}

	// check if file exists
	genPath := filepath.Join(genConfig.OutputPath, "Modelfile")
	genPath, err = filepath.Abs(genPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for modelfile: %w", err)
	}
	if _, err := os.Stat(genPath); err == nil {
		if !genConfig.Overwrite {
			return fmt.Errorf("Modelfile already exists at %s - use --overwrite to overwrite", genPath)
		}
	}

	fmt.Printf("Generating modelfile for %s\n", realModelPath)

	modelfile, err := AutoModelfile(realModelPath, genConfig)
	if err != nil {
		return fmt.Errorf("failed to generate modelfile: %w", err)
	}

	// save the modelfile to the output path
	if err := modelfile.SaveToFile(genPath); err != nil {
		return fmt.Errorf("failed to save modelfile: %w", err)
	}

	// read modelfile from disk and print it
	content, err := os.ReadFile(genPath)
	if err != nil {
		return fmt.Errorf("failed to read modelfile: %w", err)
	}
	fmt.Printf("Successfully generated modelfile:\n%s\n", string(content))

	return nil
}
