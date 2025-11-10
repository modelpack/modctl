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
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/modelpack/modctl/pkg/backend"
	"github.com/modelpack/modctl/pkg/config"
)

var uploadConfig = config.NewUpload()

// uploadCmd represents the modctl command for upload.
var uploadCmd = &cobra.Command{
	Use:                "upload [flags] <file>",
	Short:              "Upload a file to the remote end in advance to save time in the later build, applicable to the scenario of uploading while downloading, this function needs to be used together with build.",
	Args:               cobra.ExactArgs(1),
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := uploadConfig.Validate(); err != nil {
			return err
		}

		return runUpload(context.Background(), args[0])
	},
}

// init initializes upload command.
func init() {
	flags := uploadCmd.Flags()
	flags.StringVarP(&uploadConfig.Repo, "repo", "", "", "target model artifact repository name")
	flags.BoolVarP(&uploadConfig.PlainHTTP, "plain-http", "", false, "turning on this flag will use plain HTTP instead of HTTPS")
	flags.BoolVarP(&uploadConfig.Insecure, "insecure", "", false, "turning on this flag will disable TLS verification")
	flags.BoolVar(&uploadConfig.Raw, "raw", true, "turning on this flag will upload model artifact layer in raw format")

	if err := viper.BindPFlags(flags); err != nil {
		panic(fmt.Errorf("bind cache list flags to viper: %w", err))
	}
}

// runUpload runs the upload modctl.
func runUpload(ctx context.Context, filepath string) error {
	b, err := backend.New(rootConfig.StoargeDir)
	if err != nil {
		return err
	}

	if err := b.Upload(ctx, filepath, uploadConfig); err != nil {
		return err
	}

	fmt.Printf("Successfully uploaded %s to model artifact repository: %s\n", filepath, uploadConfig.Repo)
	return nil
}
