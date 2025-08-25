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

var pullConfig = config.NewPull()

// pullCmd represents the modctl command for pull.
var pullCmd = &cobra.Command{
	Use:                "pull [flags] <target>",
	Short:              "Pull a model artifact from the remote registry.",
	Args:               cobra.ExactArgs(1),
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := pullConfig.Validate(); err != nil {
			return err
		}

		return runPull(context.Background(), args[0])
	},
}

// init initializes pull command.
func init() {
	flags := pullCmd.Flags()
	flags.IntVar(&pullConfig.Concurrency, "concurrency", pullConfig.Concurrency, "specify the number of concurrent pull operations")
	flags.BoolVar(&pullConfig.PlainHTTP, "plain-http", false, "use plain HTTP instead of HTTPS")
	flags.BoolVar(&pullConfig.Insecure, "insecure", false, "use insecure connection for the pull operation and skip TLS verification")
	flags.StringVar(&pullConfig.Proxy, "proxy", "", "use proxy for the pull operation")
	flags.StringVar(&pullConfig.ExtractDir, "extract-dir", "", "specify the extract dir for extracting the model artifact")
	flags.BoolVar(&pullConfig.ExtractFromRemote, "extract-from-remote", false, "turning on this flag will pull and extract the data from remote registry and no longer store model artifact locally, so user must specify extract-dir as the output directory")
	flags.StringVar(&pullConfig.DragonflyEndpoint, "dragonfly-endpoint", "", "specify the dragonfly endpoint for the pull operation, which will download and hardlink the blob by dragonfly GRPC service, this mode requires extract-from-remote must be true")

	if err := viper.BindPFlags(flags); err != nil {
		panic(fmt.Errorf("bind cache pull flags to viper: %w", err))
	}
}

// runPull runs the pull modctl.
func runPull(ctx context.Context, target string) error {
	b, err := backend.New(rootConfig.StoargeDir)
	if err != nil {
		return err
	}

	if target == "" {
		return fmt.Errorf("target is required")
	}

	if err := b.Pull(ctx, target, pullConfig); err != nil {
		return err
	}

	fmt.Printf("Successfully pulled model artifact: %s\n", target)
	return nil
}
