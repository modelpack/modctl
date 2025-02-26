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
	"os"

	"github.com/CloudNativeAI/modctl/pkg/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootConfig *config.Root

// rootCmd represents the modctl command.
var rootCmd = &cobra.Command{
	Use:                "modctl",
	Short:              "A command line tool for managing artifact bundled based on the Model Format Specification",
	Args:               cobra.ExactArgs(1),
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		logrus.Debug("modctl is running")

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	var err error
	rootConfig, err = config.NewRoot()
	if err != nil {
		panic(err)
	}

	// Bind more cache specific persistent flags.
	flags := rootCmd.PersistentFlags()
	flags.StringVar(&rootConfig.StoargeDir, "storage-dir", rootConfig.StoargeDir, "specify the storage directory for modctl")

	// Bind common flags.
	if err := viper.BindPFlags(flags); err != nil {
		panic(err)
	}

	// Add sub command.
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(rmCmd)
	rootCmd.AddCommand(pruneCmd)
	rootCmd.AddCommand(inspectCmd)
	rootCmd.AddCommand(extractCmd)
	rootCmd.AddCommand(modelfileGenCmd)
}
