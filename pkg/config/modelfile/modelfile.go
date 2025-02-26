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

import "fmt"

type GenerateConfig struct {
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

func (g *GenerateConfig) Validate() error {
	if len(g.Output) == 0 {
		return fmt.Errorf("output path is required")
	}

	return nil
}
