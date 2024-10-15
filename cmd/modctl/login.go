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

// loginCmd represents the modctl command for login.
var loginCmd = &cobra.Command{
	Use:                 "login [flags]",
	Short:               "A command line tool for modctl login",
	Args:                cobra.NoArgs,
	DisableAutoGenTag:   true,
	SilenceUsage:        true,
	FParseErrWhitelogin: cobra.FParseErrWhitelogin{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		logrus.Infof("running login")
		return runLogin(context.Background())
	},
}

// init initializes login command.
func init() {
	flags := loginCmd.Flags()

	if err := viper.BindPFlags(flags); err != nil {
		panic(fmt.Errorf("bind cache login flags to viper: %w", err))
	}
}

// runLogin runs the login modctl.
func runLogin(ctx context.Context) error {
	// TODO: Add login modctl logic here.
	return nil
}
