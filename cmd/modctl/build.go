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

package modctl

import (
	"context"
	"fmt"

	"github.com/CloudNativeAI/modctl/pkg/oci"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var buildOpts = buildOptions{}

type buildOptions struct {
	target    string
	modelfile string
}

// buildCmd represents the modctl command for build.
var buildCmd = &cobra.Command{
	Use:                "build [flags] <path>",
	Short:              "A command line tool for modctl build",
	Args:               cobra.ExactArgs(1),
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBuild(context.Background(), args[0])
	},
}

// init initializes build command.
func init() {
	flags := buildCmd.Flags()
	flags.StringVarP(&buildOpts.target, "target", "t", "", "target model artifact name")
	flags.StringVarP(&buildOpts.modelfile, "modelfile", "f", "", "model file path")

	if err := viper.BindPFlags(flags); err != nil {
		panic(fmt.Errorf("bind cache list flags to viper: %w", err))
	}
}

// runBuild runs the build modctl.
func runBuild(ctx context.Context, workDir string) error {
	if len(buildOpts.target) == 0 {
		return fmt.Errorf("target model artifact name is required")
	}

	if len(buildOpts.modelfile) == 0 {
		buildOpts.modelfile = "Modelfile"
	}

	if err := oci.Build(ctx, buildOpts.modelfile, workDir, buildOpts.target); err != nil {
		return err
	}

	fmt.Printf("Successfully built model artifact: %s\n", buildOpts.target)
	return nil
}
