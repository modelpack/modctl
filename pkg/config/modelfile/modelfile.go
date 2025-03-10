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

package modelfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultModelfileName is the default name of the modelfile.
const DefaultModelfileName = "Modelfile"

type GenerateConfig struct {
	Workspace                   string
	Name                        string
	Version                     string
	Output                      string
	IgnoreUnrecognizedFileTypes bool
	Overwrite                   bool
	Arch                        string
	Family                      string
	Format                      string
	ParamSize                   string
	Precision                   string
	Quantization                string
}

func NewGenerateConfig() *GenerateConfig {
	return &GenerateConfig{
		Workspace:                   ".",
		Name:                        "",
		Version:                     "",
		Output:                      "",
		IgnoreUnrecognizedFileTypes: false,
		Overwrite:                   false,
		Arch:                        "",
		Family:                      "",
		Format:                      "",
		ParamSize:                   "",
		Precision:                   "",
		Quantization:                "",
	}
}

func (g *GenerateConfig) Convert(workspace string) error {
	modelfilePath := filepath.Join(g.Output, DefaultModelfileName)
	absModelfilePath, err := filepath.Abs(modelfilePath)
	if err != nil {
		return err
	}
	g.Output = absModelfilePath

	if !strings.HasSuffix(workspace, "/") {
		workspace += "/"
	}

	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return err
	}
	g.Workspace = absWorkspace
	return nil
}

func (g *GenerateConfig) Validate() error {
	if len(g.Output) == 0 {
		return fmt.Errorf("output path is required")
	}

	// Check if the output path exists modelfile, if so, check if we can overwrite it.
	// If the output path does not exist, we can create the modelfile.
	if _, err := os.Stat(g.Output); err == nil {
		if !g.Overwrite {
			return fmt.Errorf("Modelfile already exists at %s - use --overwrite to overwrite", g.Output)
		}
	}

	return nil
}
