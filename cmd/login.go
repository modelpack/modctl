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
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/CloudNativeAI/modctl/pkg/backend"
	"github.com/CloudNativeAI/modctl/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var loginConfig = config.NewLogin()

// loginCmd represents the modctl command for login.
var loginCmd = &cobra.Command{
	Use:   "login [flags] <registry>",
	Short: "A command line tool for modctl login",
	Example: `
# login to docker hub:
modctl login -u foo registry-1.docker.io

# login to registry served over http:
modctl login -u foo --plain-http registry-insecure.io
`,
	Args:               cobra.ExactArgs(1),
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := loginConfig.Validate(); err != nil {
			return err
		}

		return runLogin(cmd.Context(), args[0])
	},
}

// init initializes login command.
func init() {
	flags := loginCmd.Flags()
	flags.StringVarP(&loginConfig.Username, "username", "u", "", "Username for login")
	flags.StringVarP(&loginConfig.Password, "password", "p", "", "Password for login")
	flags.BoolVar(&loginConfig.PasswordStdin, "password-stdin", true, "Take the password from stdin by default")
	flags.BoolVar(&loginConfig.PlainHTTP, "plain-http", false, "Allow http connections to registry")

	if err := viper.BindPFlags(flags); err != nil {
		panic(fmt.Errorf("bind cache login flags to viper: %w", err))
	}
}

// runLogin runs the login modctl.
func runLogin(ctx context.Context, registry string) error {
	b, err := backend.New(rootConfig.StoargeDir)
	if err != nil {
		return err
	}

	// read password from stdin if password-stdin is set
	if loginConfig.PasswordStdin && loginConfig.Password == "" {
		fmt.Print("Enter password: ")
		password, err := terminal.ReadPassword(syscall.Stdin)
		if err != nil {
			return err
		}

		loginConfig.Password = strings.TrimSpace(string(password))
	}

	fmt.Println("\nLogging In...")

	opts := []backend.Option{
		backend.WithPlainHTTP(loginConfig.PlainHTTP),
	}

	if err := b.Login(ctx, registry, loginConfig.Username, loginConfig.Password, opts...); err != nil {
		return err
	}

	fmt.Println("Login Succeeded.")
	return nil
}
