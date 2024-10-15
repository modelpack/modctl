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

// rmCmd represents the modctl command for rm.
var rmCmd = &cobra.Command{
	Use:                "rm [flags]",
	Short:              "A command line tool for modctl rm",
	Args:               cobra.NoArgs,
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		logrus.Infof("running rm")
		return runRm(context.Background())
	},
}

// init initializes rm command.
func init() {
	flags := rmCmd.Flags()

	if err := viper.BindPFlags(flags); err != nil {
		panic(fmt.Errorf("bind cache rm flags to viper: %w", err))
	}
}

// runRm runs the rm modctl.
func runRm(ctx context.Context) error {
	// TODO: Add rm modctl logic here.
	return nil
}
