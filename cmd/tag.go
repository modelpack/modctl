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

	"github.com/modelpack/modctl/pkg/backend"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// tagCmd represents the modctl command for tag.
var tagCmd = &cobra.Command{
	Use:                "tag [flags] <source> <target>",
	Short:              "Tag can tag one model artifact to another one without rebuiding.",
	Args:               cobra.ExactArgs(2),
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTag(context.Background(), args[0], args[1])
	},
}

// init initializes tag command.
func init() {
	flags := tagCmd.Flags()

	if err := viper.BindPFlags(flags); err != nil {
		panic(fmt.Errorf("bind cache tag flags to viper: %w", err))
	}
}

// runTag runs the tag modctl.
func runTag(ctx context.Context, source, target string) error {
	b, err := backend.New(rootConfig.StoargeDir)
	if err != nil {
		return err
	}

	if source == "" || target == "" {
		return fmt.Errorf("source and target are required")
	}

	return b.Tag(ctx, source, target)
}
