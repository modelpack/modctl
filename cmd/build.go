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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/modelpack/modctl/pkg/backend"
	"github.com/modelpack/modctl/pkg/config"
)

var buildConfig = config.NewBuild()

// buildCmd represents the modctl command for build.
var buildCmd = &cobra.Command{
	Use:               "build [flags] <path>",
	Short:             "Build the model artifact with the context by specified path.",
	Args:              cobra.ExactArgs(1),
	DisableAutoGenTag: true,
	SilenceUsage:      true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := buildConfig.Validate(); err != nil {
			return err
		}

		return runBuild(cmd.Context(), args[0])
	},
}

// init initializes build command.
func init() {
	flags := buildCmd.Flags()
	flags.IntVarP(&buildConfig.Concurrency, "concurrency", "c", buildConfig.Concurrency, "specify the number of concurrent build operations")
	flags.StringVarP(&buildConfig.Target, "target", "t", buildConfig.Target, "target model artifact name")
	flags.StringVarP(&buildConfig.Modelfile, "modelfile", "f", buildConfig.Modelfile, "model file path")
	flags.BoolVarP(&buildConfig.OutputRemote, "output-remote", "", false, "turning on this flag will output model artifact to remote registry directly")
	flags.BoolVarP(&buildConfig.PlainHTTP, "plain-http", "", false, "turning on this flag will use plain HTTP instead of HTTPS")
	flags.BoolVarP(&buildConfig.Insecure, "insecure", "", false, "turning on this flag will disable TLS verification")
	flags.BoolVar(&buildConfig.Nydusify, "nydusify", false, "[EXPERIMENTAL] nydusify the model artifact")
	flags.MarkHidden("nydusify")
	flags.StringVar(&buildConfig.SourceURL, "source-url", "", "source URL")
	flags.StringVar(&buildConfig.SourceRevision, "source-revision", "", "source revision")
	flags.BoolVar(&buildConfig.Raw, "raw", true, "turning on this flag will build model artifact layers in raw format")
	flags.BoolVar(&buildConfig.Reasoning, "reasoning", false, "turning on this flag will mark this model as reasoning model in the config")
	flags.BoolVar(&buildConfig.NoCreationTime, "no-creation-time", false, "turning on this flag will not set createdAt in the config, which will be helpful for repeated builds")
	flags.IntVar(&buildConfig.RetryConfig.MaxAttempts, "retry-attempts", 0, "Max total attempts per file (initial + retries; 0 = use default of 6, 1 = no retry)")
	flags.DurationVar(&buildConfig.RetryConfig.PerAttemptTimeout, "per-attempt-timeout", 0, "Timeout for a single transfer attempt (0 = derive from file size; <0 = disabled)")

	if err := viper.BindPFlags(flags); err != nil {
		panic(fmt.Errorf("bind build flags to viper: %w", err))
	}
}

// runBuild runs the build modctl.
func runBuild(ctx context.Context, workDir string) error {
	b, err := backend.New(rootConfig.StorageDir)
	if err != nil {
		return err
	}

	if err := b.Build(ctx, buildConfig.Modelfile, workDir, buildConfig.Target, buildConfig); err != nil {
		return err
	}

	fmt.Printf("Successfully built model artifact: %s\n", buildConfig.Target)

	return nil
}
