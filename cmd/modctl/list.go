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

// listCmd represents the modctl command for list.
var listCmd = &cobra.Command{
	Use:                "list [flags]",
	Short:              "A command line tool for modctl list",
	Args:               cobra.NoArgs,
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		logrus.Infof("running list")
		return runList(context.Background())
	},
}

// init initializes list command.
func init() {
	flags := listCmd.Flags()

	if err := viper.BindPFlags(flags); err != nil {
		panic(fmt.Errorf("bind cache list flags to viper: %w", err))
	}
}

// runList runs the list modctl.
func runList(ctx context.Context) error {
	// TODO: Add list modctl logic here.
	return nil
}
