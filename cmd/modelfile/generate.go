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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	configmodelfile "github.com/modelpack/modctl/pkg/config/modelfile"
	"github.com/modelpack/modctl/pkg/modelfile"
	"github.com/modelpack/modctl/pkg/modelprovider"
)

var generateConfig = configmodelfile.NewGenerateConfig()

// generateCmd represents the modelfile tools command for generating modelfile.
var generateCmd = &cobra.Command{
	Use:   "generate [flags] [<path>]",
	Short: "Generate a modelfile from a local workspace or remote model provider",
	Long: `Generate a modelfile from either a local directory containing model files or by downloading a model from a supported provider.

The workspace must be a directory including model files and model configuration files.
Alternatively, use --model-url to download a model from a supported provider (e.g., HuggingFace, ModelScope).

For short-form URLs (owner/repo), you must explicitly specify the provider using --provider flag.
Full URLs with domain names will auto-detect the provider.`,
	Example: `  # Generate from local directory
  modctl modelfile generate ./my-model-dir

  # Generate from Hugging Face using full URL (auto-detects provider)
  modctl modelfile generate --model-url https://huggingface.co/meta-llama/Llama-2-7b-hf

  # Generate from Hugging Face using short form (requires --provider)
  modctl modelfile generate --model-url meta-llama/Llama-2-7b-hf --provider huggingface

  # Generate from ModelScope using full URL (auto-detects provider)
  modctl modelfile generate --model-url https://modelscope.cn/models/qwen/Qwen-7B

  # Generate from ModelScope using short form (requires --provider)
  modctl modelfile generate --model-url qwen/Qwen-7B --provider modelscope

  # Generate with custom output path
  modctl modelfile generate ./my-model-dir --output ./output/modelfile.yaml

  # Generate with metadata overrides
  modctl modelfile generate ./my-model-dir --name my-custom-model --family llama3`,
	Args:               cobra.MaximumNArgs(1),
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		// If model-url is provided, path is optional
		workspace := "."
		if len(args) > 0 {
			workspace = args[0]
		}

		// Validate that either path or model-url is provided
		if generateConfig.ModelURL != "" && len(args) > 0 {
			return fmt.Errorf("the <path> argument and the --model-url flag are mutually exclusive")
		}
		if generateConfig.ModelURL == "" && len(args) == 0 {
			return fmt.Errorf("either a <path> argument or the --model-url flag must be provided")
		}

		if err := generateConfig.Convert(workspace); err != nil {
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
	flags.StringVarP(&generateConfig.Output, "output", "O", ".", "specify the output path of modelfilem, must be a directory")
	flags.BoolVar(&generateConfig.IgnoreUnrecognizedFileTypes, "ignore-unrecognized-file-types", false, "ignore the unrecognized file types in the workspace")
	flags.BoolVar(&generateConfig.Overwrite, "overwrite", false, "overwrite the existing modelfile")
	flags.StringVar(&generateConfig.ModelURL, "model-url", "", "download model from a supported provider (full URL or short-form with --provider)")
	flags.StringVarP(&generateConfig.Provider, "provider", "p", "", "explicitly specify the provider for short-form URLs (huggingface, modelscope)")

	// Mark the ignore-unrecognized-file-types flag as deprecated and hidden
	flags.MarkDeprecated("ignore-unrecognized-file-types", "this flag will be removed in the next release")
	flags.MarkHidden("ignore-unrecognized-file-types")

	if err := viper.BindPFlags(flags); err != nil {
		panic(fmt.Errorf("bind cache list flags to viper: %w", err))
	}
}

// runGenerate runs the generate modelfile.
func runGenerate(ctx context.Context) error {
	// If model URL is provided, download the model first
	if generateConfig.ModelURL != "" {
		fmt.Printf("Model URL provided: %s\n", generateConfig.ModelURL)

		// Get the appropriate provider for this URL
		registry := modelprovider.GetRegistry()
		provider, err := registry.SelectProvider(generateConfig.ModelURL, generateConfig.Provider)
		if err != nil {
			return fmt.Errorf("failed to select provider: %w", err)
		}

		fmt.Printf("Using provider: %s\n", provider.Name())

		// Check if user is authenticated with the provider
		if err := provider.CheckAuth(); err != nil {
			return fmt.Errorf("%s authentication check failed: %w", provider.Name(), err)
		}

		// Create a temporary directory for downloading the model
		// Clean up the temporary directory after the function returns
		tmpDir, err := os.MkdirTemp("", "modctl-model-downloads-*")
		if err != nil {
			return fmt.Errorf("failed to create temporary directory: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		// Download the model
		downloadPath, err := provider.DownloadModel(ctx, generateConfig.ModelURL, tmpDir)
		if err != nil {
			return fmt.Errorf("failed to download model from %s: %w", provider.Name(), err)
		}

		// Update workspace to the downloaded model path
		generateConfig.Workspace = downloadPath
		fmt.Printf("Using downloaded model at: %s\n", downloadPath)
	}

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
