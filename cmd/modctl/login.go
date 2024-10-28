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
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/CloudNativeAI/modctl/pkg/config"
	"github.com/CloudNativeAI/modctl/pkg/oci"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var loginConfig = config.NewLogin()

type loginOptions struct {
	username      string
	password      string
	passwordStdin bool
}

// loginCmd represents the modctl command for login.
var loginCmd = &cobra.Command{
	Use:                "login [flags] <registry>",
	Short:              "A command line tool for modctl login",
	Args:               cobra.ExactArgs(1),
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := loginConfig.Validate(); err != nil {
			return err
		}

		return runLogin(context.Background(), args[0])
	},
}

// init initializes login command.
func init() {
	flags := loginCmd.Flags()
	flags.StringVarP(&loginConfig.Username, "username", "u", "", "Username for login")
	flags.StringVarP(&loginConfig.Password, "password", "p", "", "Password for login")
	flags.BoolVar(&loginConfig.PasswordStdin, "password-stdin", false, "Take the password from stdin")

	if err := viper.BindPFlags(flags); err != nil {
		panic(fmt.Errorf("bind cache login flags to viper: %w", err))
	}
}

// runLogin runs the login modctl.
func runLogin(ctx context.Context, registry string) error {
	// read password from stdin if password-stdin is set
	if loginConfig.PasswordStdin {
		fmt.Print("Enter password: ")
		reader := bufio.NewReader(os.Stdin)
		password, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		loginConfig.Password = strings.TrimSpace(password)
	}

	if err := oci.Login(ctx, registry, loginConfig.Username, loginConfig.Password); err != nil {
		return err
	}

	fmt.Println("Login Succeeded.")
	return nil
}
