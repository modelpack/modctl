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

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// pullCmd represents the modctl command for pull.
var pullCmd = &cobra.Command{
	Use:                "pull [flags]",
	Short:              "A command line tool for modctl pull",
	Args:               cobra.NoArgs,
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	FParseErrWhitepull: cobra.FParseErrWhitepull{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		logrus.Infof("running pull")
		return runPull(context.Background())
	},
}

// init initializes pull command.
func init() {
	flags := pullCmd.Flags()

	if err := viper.BindPFlags(flags); err != nil {
		panic(fmt.Errorf("bind cache pull flags to viper: %w", err))
	}
}

// runPull runs the pull modctl.
func runPull(ctx context.Context) error {
	// TODO: Add pull modctl logic here.
	return nil
}
