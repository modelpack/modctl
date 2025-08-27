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

	"github.com/modelpack/modctl/pkg/backend"
	"github.com/modelpack/modctl/pkg/config"
)

var fetchConfig = config.NewFetch()

// fetchCmd represents the modctl command for fetch.
var fetchCmd = &cobra.Command{
	Use:                "fetch [flags] <target>",
	Short:              "Fetch can retrieve files from the remote model repository, enabling selective download of partial model files by filtering based on file path patterns.",
	Args:               cobra.ExactArgs(1),
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := fetchConfig.Validate(); err != nil {
			return err
		}

		return runFetch(context.Background(), args[0])
	},
}

// init initializes fetch command.
func init() {
	flags := fetchCmd.Flags()
	flags.IntVar(&fetchConfig.Concurrency, "concurrency", fetchConfig.Concurrency, "specify the number of concurrent fetch operations")
	flags.BoolVar(&fetchConfig.PlainHTTP, "plain-http", false, "use plain HTTP instead of HTTPS")
	flags.BoolVar(&fetchConfig.Insecure, "insecure", false, "use insecure connection for the fetch operation and skip TLS verification")
	flags.StringVar(&fetchConfig.Proxy, "proxy", "", "use proxy for the fetch operation")
	flags.StringVar(&fetchConfig.Output, "output", "", "specify the directory for fetching the model artifact")
	flags.StringSliceVar(&fetchConfig.Patterns, "patterns", []string{}, "specify the patterns for fetching the model artifact")

	if err := viper.BindPFlags(flags); err != nil {
		panic(fmt.Errorf("bind cache pull flags to viper: %w", err))
	}
}

// runFetch runs the fetch modctl.
func runFetch(ctx context.Context, target string) error {
	b, err := backend.New(rootConfig.StoargeDir)
	if err != nil {
		return err
	}

	if target == "" {
		return fmt.Errorf("target is required")
	}

	if err := b.Fetch(ctx, target, fetchConfig); err != nil {
		return err
	}

	fmt.Printf("Successfully fetched model artifact: %s\n", target)
	return nil
}
