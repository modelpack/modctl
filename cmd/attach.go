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
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/CloudNativeAI/modctl/pkg/backend"
	"github.com/CloudNativeAI/modctl/pkg/config"
)

var attachConfig = config.NewAttach()

// attachCmd represents the modctl command for attach.
var attachCmd = &cobra.Command{
	Use:                "attach [flags] <file>",
	Short:              "A command line tool for modctl attach",
	Args:               cobra.ExactArgs(1),
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := attachConfig.Validate(); err != nil {
			return err
		}

		return runAttach(context.Background(), args[0])
	},
}

// init initializes build command.
func init() {
	flags := attachCmd.Flags()
	flags.StringVarP(&attachConfig.Source, "source", "s", "", "source model artifact name")
	flags.StringVarP(&attachConfig.Target, "target", "t", "", "target model artifact name")
	flags.BoolVarP(&attachConfig.OutputRemote, "output-remote", "", false, "turning on this flag will output model artifact to remote registry directly")
	flags.BoolVarP(&attachConfig.PlainHTTP, "plain-http", "", false, "turning on this flag will use plain HTTP instead of HTTPS")
	flags.BoolVarP(&attachConfig.Insecure, "insecure", "", false, "turning on this flag will disable TLS verification")
	flags.BoolVarP(&attachConfig.Force, "force", "f", false, "turning on this flag will force the attach, which will overwrite the layer if it already exists with same filepath")
	flags.BoolVar(&attachConfig.Nydusify, "nydusify", false, "[EXPERIMENTAL] nydusify the model artifact")
	flags.MarkHidden("nydusify")
	flags.BoolVar(&attachConfig.Raw, "raw", false, "turning on this flag will attach model artifact layer in raw format")
	flags.BoolVar(&attachConfig.Config, "config", false, "turning on this flag will overwrite model artifact config layer")

	if err := viper.BindPFlags(flags); err != nil {
		panic(fmt.Errorf("bind cache list flags to viper: %w", err))
	}
}

// runAttach runs the attach modctl.
func runAttach(ctx context.Context, filepath string) error {
	b, err := backend.New(rootConfig.StoargeDir)
	if err != nil {
		return err
	}

	if err := b.Attach(ctx, filepath, attachConfig); err != nil {
		return err
	}

	fmt.Printf("Successfully attached model artifact: %s\n", attachConfig.Target)

	return nil
}
