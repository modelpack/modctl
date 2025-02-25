/*
 *     Copyright 2025 The CNAI Authors
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
	"fmt"

	"github.com/CloudNativeAI/modctl/pkg/version"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// versionCmd represents the modctl command for version.
var versionCmd = &cobra.Command{
	Use:                "version",
	Short:              "A command line tool for modctl version",
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runVersion()
	},
}

// init initializes version command.
func init() {
	flags := rmCmd.Flags()

	if err := viper.BindPFlags(flags); err != nil {
		panic(fmt.Errorf("bind version flags to viper: %w", err))
	}
}

// runVersion runs the version modctl.
func runVersion() error {
	fmt.Printf("%-12s%s\n", "Version:", version.GitVersion)
	fmt.Printf("%-12s%s\n", "Commit:", version.GitCommit)
	fmt.Printf("%-12s%s\n", "Platform:", version.Platform)
	fmt.Printf("%-12s%s\n", "BuildTime:", version.BuildTime)
	return nil
}
