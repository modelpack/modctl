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

package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/CloudNativeAI/modctl/pkg/modelfile"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var genConfig = modelfile.NewModelfileGenConfig()

// modelfileGenCmd represents the modctl tools command for generate modelfile.
var modelfileGenCmd = &cobra.Command{
	Use:                "genmodelfile [flags] <path>",
	Short:              "A command line tool for generating modelfile.",
	Args:               cobra.ExactArgs(1),
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGenModelfile(context.Background(), args[0])
	},
}

// init initializes build command.
func init() {
	flags := modelfileGenCmd.Flags()
	flags.StringVarP(&genConfig.Name, "name", "n", "", "Model name (string), such as llama3-8b-instruct, gpt2-xl, qwen2-vl-72b-instruct, etc.")
	flags.StringVarP(&genConfig.Version, "version", "v", "", "Model version (string), such as v1, v2, etc.")
	flags.StringVarP(&genConfig.OutputPath, "output", "o", "./", "Output path (string), such as /path/to/output.")
	flags.BoolVar(&genConfig.IgnoreUnrecognized, "ignore-unrecognized", false, "Ignore the unrecognized file types in directory.")
	flags.BoolVar(&genConfig.Overwrite, "overwrite", false, "Overwrite the existing modelfile.")
	flags.StringVar(&genConfig.Arch, "arch", "", "Model architecture (string), such as transformer, cnn, rnn, etc.")
	flags.StringVar(&genConfig.Family, "family", "", "Model family (string), such as llama3, gpt2, qwen2, etc.")
	flags.StringVar(&genConfig.Format, "format", "", "Model format (string), such as safetensors, pytorch, onnx, etc.")
	flags.StringVar(&genConfig.Paramsize, "paramsize", "", "Number of parameters in the model (string), such as 7B, 13B, 72B, etc.")
	flags.StringVar(&genConfig.Precision, "precision", "", "Model precision (string), such as bf16, fp16, int8, etc.")
	flags.StringVar(&genConfig.Quantization, "quantization", "", "Model quantization (string), such as awq, gptq, etc.")

	if err := viper.BindPFlags(flags); err != nil {
		panic(fmt.Errorf("bind cache list flags to viper: %w", err))
	}
}

func runGenModelfile(ctx context.Context, modelPath string) error {

	if !strings.HasSuffix(modelPath, "/") {
		modelPath += "/"
	}

	return modelfile.RunGenModelfile(ctx, modelPath, genConfig)
}
