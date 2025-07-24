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

var extractConfig = config.NewExtract()

// extractCmd represents the modctl command for extract.
var extractCmd = &cobra.Command{
	Use:                "extract <target> --output <output>",
	Short:              "A command line tool for modctl extract",
	Args:               cobra.ExactArgs(1),
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := extractConfig.Validate(); err != nil {
			return err
		}

		return runExtract(context.Background(), args[0])
	},
}

// init initializes extract command.
func init() {
	flags := extractCmd.Flags()
	flags.StringVar(&extractConfig.Output, "output", "", "specify the output for extracting the model artifact")
	flags.IntVar(&extractConfig.Concurrency, "concurrency", extractConfig.Concurrency, "specify the concurrency for extracting the model artifact")

	if err := viper.BindPFlags(flags); err != nil {
		panic(fmt.Errorf("bind cache extract flags to viper: %w", err))
	}
}

// runExtract runs the extract modctl.
func runExtract(ctx context.Context, target string) error {
	b, err := backend.New(rootConfig.StoargeDir)
	if err != nil {
		return err
	}

	if target == "" {
		return fmt.Errorf("target is required")
	}

	if err := b.Extract(ctx, target, extractConfig); err != nil {
		return err
	}

	fmt.Printf("Successfully extracted model artifact %s to %s\n", target, extractConfig.Output)
	return nil
}
