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

package modelfile

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// RootCmd represents the modelfile tools command for modelfile operation.
var RootCmd = &cobra.Command{
	Use:                "modelfile",
	Short:              "A command line tool for modelfile operation",
	Args:               cobra.ExactArgs(1),
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

// init initializes modelfile command.
func init() {
	flags := RootCmd.Flags()

	if err := viper.BindPFlags(flags); err != nil {
		panic(err)
	}

	// Add sub command.
	RootCmd.AddCommand(generateCmd)
}
