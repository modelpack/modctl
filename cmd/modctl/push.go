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

	"github.com/CloudNativeAI/modctl/pkg/backend"
	"github.com/CloudNativeAI/modctl/pkg/config"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var pushConfig = config.NewPull()

// pushCmd represents the modctl command for push.
var pushCmd = &cobra.Command{
	Use:                "push [flags] <target>",
	Short:              "A command line tool for modctl push",
	Args:               cobra.ExactArgs(1),
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPush(context.Background(), args[0])
	},
}

// init initializes push command.
func init() {
	flags := pushCmd.Flags()
	flags.BoolVarP(&pushConfig.PlainHTTP, "plain-http", "p", false, "use plain HTTP instead of HTTPS")

	if err := viper.BindPFlags(flags); err != nil {
		panic(fmt.Errorf("bind cache push flags to viper: %w", err))
	}
}

// runPush runs the push modctl.
func runPush(ctx context.Context, target string) error {
	b, err := backend.New()
	if err != nil {
		return err
	}

	opts := []backend.Option{}
	if pushConfig.PlainHTTP {
		opts = append(opts, backend.WithPlainHTTP())
	}

	if err := b.Push(ctx, target, opts...); err != nil {
		return err
	}

	fmt.Printf("Successfully pushed model artifact: %s\n", target)
	return nil
}
