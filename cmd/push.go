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

var pushConfig = config.NewPush()

// pushCmd represents the modctl command for push.
var pushCmd = &cobra.Command{
	Use:                "push [flags] <target>",
	Short:              "A command line tool for modctl push",
	Args:               cobra.ExactArgs(1),
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := pushConfig.Validate(); err != nil {
			return err
		}

		return runPush(context.Background(), args[0])
	},
}

// init initializes push command.
func init() {
	flags := pushCmd.Flags()
	flags.IntVar(&pushConfig.Concurrency, "concurrency", pushConfig.Concurrency, "specify the number of concurrent push operations")
	flags.BoolVar(&pushConfig.PlainHTTP, "plain-http", false, "use plain HTTP instead of HTTPS")
	flags.BoolVar(&pushConfig.Nydusify, "nydusify", false, "[EXPERIMENTAL] nydusify the model artifact")
	flags.MarkHidden("nydusify")

	if err := viper.BindPFlags(flags); err != nil {
		panic(fmt.Errorf("bind cache push flags to viper: %w", err))
	}
}

// runPush runs the push modctl.
func runPush(ctx context.Context, target string) error {
	b, err := backend.New(rootConfig.StoargeDir)
	if err != nil {
		return err
	}

	if err := b.Push(ctx, target, pushConfig); err != nil {
		return err
	}

	fmt.Printf("Successfully pushed model artifact: %s\n", target)

	// nydusify the model artifact if needed.
	if pushConfig.Nydusify {
		nydusName, err := b.Nydusify(ctx, target)
		if err != nil {
			return fmt.Errorf("failed to nydusify %s: %w", target, err)
		}

		fmt.Printf("Successfully nydusify model artifact: %s\n", nydusName)
	}

	return nil
}
