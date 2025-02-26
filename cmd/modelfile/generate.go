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
	"context"
	"fmt"
	"strings"

	configmodelfile "github.com/CloudNativeAI/modctl/pkg/config/modelfile"
	"github.com/CloudNativeAI/modctl/pkg/modelfile"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var generateConfig = configmodelfile.NewGenerateConfig()

// generateCmd represents the modelfile tools command for generating modelfile.
var generateCmd = &cobra.Command{
	Use:                "generate [flags] <path>",
	Short:              "A command line tool for generating modelfile",
	Args:               cobra.ExactArgs(1),
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGenerate(context.Background(), args[0])
	},
}

// init initializes generate command.
func init() {
	flags := generateCmd.Flags()
	flags.StringVarP(&generateConfig.Name, "name", "n", "", "specify the model name, such as llama3-8b-instruct, gpt2-xl, qwen2-vl-72b-instruct, etc")
	flags.StringVar(&generateConfig.Arch, "arch", "", "specify the model architecture, such as transformer, cnn, rnn, etc")
	flags.StringVar(&generateConfig.Family, "family", "", "specify model family, such as llama3, gpt2, qwen2, etc")
	flags.StringVar(&generateConfig.Format, "format", "", "specify model format, such as safetensors, pytorch, onnx, etc")
	flags.StringVar(&generateConfig.ParamSize, "param-size", "", "specify number of model parameters, such as 8b, 16b, 32b, etc")
	flags.StringVar(&generateConfig.Precision, "precision", "", "specify model precision, such as bf16, fp16, int8, etc")
	flags.StringVar(&generateConfig.Quantization, "quantization", "", "specify model quantization, such as awq, gptq, etc")
	flags.StringVarP(&generateConfig.Output, "output", "O", ".", "specify the output path of modelfile")
	flags.BoolVar(&generateConfig.IgnoreUnrecognizedFileTypes, "ignore-unrecognized-file-types", false, "ignore the unrecognized file types in the workspace")
	flags.BoolVar(&generateConfig.Overwrite, "overwrite", false, "overwrite the existing modelfile")

	if err := viper.BindPFlags(flags); err != nil {
		panic(fmt.Errorf("bind cache list flags to viper: %w", err))
	}
}

func runGenerate(ctx context.Context, modelPath string) error {
	if !strings.HasSuffix(modelPath, "/") {
		modelPath += "/"
	}

	return modelfile.RunGenModelfile(ctx, modelPath, generateConfig)
}
