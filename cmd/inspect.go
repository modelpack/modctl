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
	"encoding/json"
	"fmt"

	"github.com/CloudNativeAI/modctl/pkg/backend"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// inspectCmd represents the modctl command for inspect.
var inspectCmd = &cobra.Command{
	Use:                "inspect [flags] <target>",
	Short:              "A command line tool for modctl inspect",
	Args:               cobra.ExactArgs(1),
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInspect(context.Background(), args[0])
	},
}

// init initializes inspect command.
func init() {
	flags := rmCmd.Flags()

	if err := viper.BindPFlags(flags); err != nil {
		panic(fmt.Errorf("bind cache inspect flags to viper: %w", err))
	}
}

// runInspect runs the inspect modctl.
func runInspect(ctx context.Context, target string) error {
	b, err := backend.New(rootConfig.StoargeDir)
	if err != nil {
		return err
	}

	if target == "" {
		return fmt.Errorf("target is required")
	}

	inspected, err := b.Inspect(ctx, target)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(inspected, "", "	")
	if err != nil {
		return err
	}

	fmt.Println(string(data))
	return nil
}
