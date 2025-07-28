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
	"os"

	configmodelfile "github.com/CloudNativeAI/modctl/pkg/config/modelfile"
	"github.com/CloudNativeAI/modctl/pkg/modelfile"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var generateConfig = configmodelfile.NewGenerateConfig()

// generateCmd represents the modelfile tools command for generating modelfile.
var generateCmd = &cobra.Command{
	Use:                "generate [flags] <path>",
	Short:              "A command line tool for generating modelfile in the workspace, the workspace must be a directory including model files and model configuration files",
	Args:               cobra.ExactArgs(1),
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := generateConfig.Convert(args[0]); err != nil {
			return err
		}

		if err := generateConfig.Validate(); err != nil {
			return err
		}

		return runGenerate(context.Background())
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
	flags.StringVarP(&generateConfig.Output, "output", "O", ".", "specify the output path of modelfile, must be a directory")
	flags.BoolVar(&generateConfig.IgnoreUnrecognizedFileTypes, "ignore-unrecognized-file-types", false, "ignore the unrecognized file types in the workspace")
	flags.BoolVar(&generateConfig.Overwrite, "overwrite", false, "overwrite the existing modelfile")

	// Mark the ignore-unrecognized-file-types flag as deprecated and hidden
	flags.MarkDeprecated("ignore-unrecognized-file-types", "this flag will be removed in the next release")
	flags.MarkHidden("ignore-unrecognized-file-types")

	if err := viper.BindPFlags(flags); err != nil {
		panic(fmt.Errorf("bind cache list flags to viper: %w", err))
	}
}

// runGenerate runs the generate modelfile.
func runGenerate(_ context.Context) error {
	fmt.Printf("Generating modelfile for %s\n", generateConfig.Workspace)
	modelfile, err := modelfile.NewModelfileByWorkspace(generateConfig.Workspace, generateConfig)
	if err != nil {
		return fmt.Errorf("failed to generate modelfile: %w", err)
	}

	content := modelfile.Content()
	if err := os.WriteFile(generateConfig.Output, content, 0644); err != nil {
		return fmt.Errorf("failed to write modelfile: %w", err)
	}

	fmt.Printf("Successfully generated modelfile:\n%s\n", string(content))
	return nil
}
