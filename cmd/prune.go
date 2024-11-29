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

	"github.com/CloudNativeAI/modctl/pkg/backend"
	"github.com/CloudNativeAI/modctl/pkg/config"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var pruneConfig = config.NewPrune()

// pruneCmd represents the modctl command for prune.
var pruneCmd = &cobra.Command{
	Use:                "prune [flags]",
	Short:              "A command line tool for modctl prune",
	Args:               cobra.NoArgs,
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPrune(context.Background())
	},
}

// init initializes prune command.
func init() {
	flags := rmCmd.Flags()
	flags.BoolVar(&pruneConfig.DryRun, "dry-run", false, "do not remove any blobs, just print what would be removed")
	flags.BoolVar(&pruneConfig.RemoveUntagged, "remove-untagged", true, "remove untagged manifests")

	if err := viper.BindPFlags(flags); err != nil {
		panic(fmt.Errorf("bind cache rm flags to viper: %w", err))
	}
}

// runPrune runs the prune modctl.
func runPrune(ctx context.Context) error {
	b, err := backend.New()
	if err != nil {
		return err
	}

	return b.Prune(ctx, pruneConfig.DryRun, pruneConfig.RemoveUntagged)
}
