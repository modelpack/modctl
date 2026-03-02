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
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/modelpack/modctl/cmd/modelfile"
	internalpb "github.com/modelpack/modctl/internal/pb"
	"github.com/modelpack/modctl/pkg/config"
)

var rootConfig *config.Root
var logFile *os.File

// rootCmd represents the modctl command.
var rootCmd = &cobra.Command{
	Use:                "modctl",
	Short:              "A command line tool for managing artifact bundled based on the ModelPack Specification",
	Args:               cobra.MaximumNArgs(1),
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Start pprof server if enabled.
		if rootConfig.Pprof {
			go func() {
				err := http.ListenAndServe(rootConfig.PprofAddr, nil)
				if err != nil {
					log.Fatal(err)
				}
			}()
		}

		// Ensure log directory exists.
		if err := os.MkdirAll(rootConfig.LogDir, 0755); err != nil {
			return err
		}

		// Ensure log file exists.
		var err error
		logFile, err = os.OpenFile(filepath.Join(rootConfig.LogDir, "modctl.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return err
		}

		logLevel, err := logrus.ParseLevel(rootConfig.LogLevel)
		if err != nil {
			return err
		}

		logrus.SetOutput(logFile)
		logrus.SetLevel(logLevel)
		logrus.SetFormatter(&logrus.TextFormatter{})

		// TODO: need refactor as currently use a global flag to control the progress bar render.
		internalpb.SetDisableProgress(rootConfig.DisableProgress)
		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if logFile != nil {
			return logFile.Close()
		}

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sig
		os.Exit(1)
	}()

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
	flags.StringVar(&rootConfig.StorageDir, "storage-dir", rootConfig.StorageDir, "specify the storage directory for modctl")
	flags.BoolVar(&rootConfig.Pprof, "pprof", rootConfig.Pprof, "enable pprof")
	flags.StringVar(&rootConfig.PprofAddr, "pprof-addr", rootConfig.PprofAddr, "specify the address for pprof")
	flags.BoolVar(&rootConfig.DisableProgress, "no-progress", rootConfig.DisableProgress, "disable progress bar")
	flags.StringVar(&rootConfig.LogDir, "log-dir", rootConfig.LogDir, "specify the log directory for modctl")
	flags.StringVar(&rootConfig.LogLevel, "log-level", rootConfig.LogLevel, "specify the log level for modctl")

	// Bind common flags.
	if err := viper.BindPFlags(flags); err != nil {
		panic(err)
	}

	// Add sub command.
	rootCmd.AddCommand(versionCmd)
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
	rootCmd.AddCommand(tagCmd)
	rootCmd.AddCommand(fetchCmd)
	rootCmd.AddCommand(attachCmd)
	rootCmd.AddCommand(uploadCmd)
	rootCmd.AddCommand(modelfile.RootCmd)
}
