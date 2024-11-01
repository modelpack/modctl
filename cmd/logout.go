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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// logoutCmd represents the modctl command for logout.
var logoutCmd = &cobra.Command{
	Use:                "logout [flags]",
	Short:              "A command line tool for modctl logout",
	Args:               cobra.ExactArgs(1),
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLogout(context.Background(), args[0])
	},
}

// init initializes logout command.
func init() {
	flags := logoutCmd.Flags()

	if err := viper.BindPFlags(flags); err != nil {
		panic(fmt.Errorf("bind cache logout flags to viper: %w", err))
	}
}

// runLogout runs the logout modctl.
func runLogout(ctx context.Context, registry string) error {
	b, err := backend.New()
	if err != nil {
		return err
	}

	if err := b.Logout(ctx, registry); err != nil {
		return err
	}

	fmt.Println("Logout Succeeded.")
	return nil
}
