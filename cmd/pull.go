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

var pullConfig = config.NewPull()

// pullCmd represents the modctl command for pull.
var pullCmd = &cobra.Command{
	Use:                "pull [flags] <target>",
	Short:              "A command line tool for modctl pull",
	Args:               cobra.ExactArgs(1),
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPull(context.Background(), args[0])
	},
}

// init initializes pull command.
func init() {
	flags := pullCmd.Flags()
	flags.BoolVar(&pullConfig.PlainHTTP, "plain-http", false, "use plain HTTP instead of HTTPS")
	flags.BoolVar(&pullConfig.Insecure, "insecure", false, "use insecure connection for the pull operation and skip TLS verification")
	flags.StringVar(&pullConfig.Proxy, "proxy", "", "use proxy for the pull operation")
	flags.StringVar(&pullConfig.ExtractDir, "extract-dir", "", "specify the extract dir for extracting the model artifact")

	if err := viper.BindPFlags(flags); err != nil {
		panic(fmt.Errorf("bind cache pull flags to viper: %w", err))
	}
}

// runPull runs the pull modctl.
func runPull(ctx context.Context, target string) error {
	b, err := backend.New()
	if err != nil {
		return err
	}

	if target == "" {
		return fmt.Errorf("target is required")
	}

	opts := []backend.Option{backend.WithInsecure(pullConfig.Insecure)}
	if pullConfig.PlainHTTP {
		opts = append(opts, backend.WithPlainHTTP())
	}

	if pullConfig.Proxy != "" {
		opts = append(opts, backend.WithProxy(pullConfig.Proxy))
	}

	if pullConfig.ExtractDir != "" {
		opts = append(opts, backend.WithOutput(pullConfig.ExtractDir))
	}

	if err := b.Pull(ctx, target, opts...); err != nil {
		return err
	}

	fmt.Printf("Successfully pulled model artifact: %s\n", target)
	return nil
}
