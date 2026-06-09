/*
Copyright 2025 linux.do
Modified by Arctel.net, 2026

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"log"

	"github.com/Rain-kl/Wavelet/internal/db/migrator"
	"github.com/spf13/cobra"
)

// Version is the application version string. It is set at link time via:
//
//	-ldflags="-X github.com/Rain-kl/Wavelet/internal/cmd.Version=<version>"
//
// When not set (e.g. local `go run`), the value defaults to "dev".
var Version = "dev"

var rootCmd = &cobra.Command{
	Use: "wavelet",
	PreRun: func(_ *cobra.Command, _ []string) {
		migrator.Migrate()
	},
	Run: func(_ *cobra.Command, args []string) {
		// 无参数时默认以融合模式启动所有服务
		if len(args) == 0 {
			allCmd.Run(allCmd, args)
			return
		}
		appMode := args[0]
		switch appMode {
		case "api":
			apiCmd.Run(apiCmd, args)
		case "scheduler":
			schedulerCmd.Run(schedulerCmd, args)
		case "worker":
			workerCmd.Run(workerCmd, args)
		case "all":
			allCmd.Run(allCmd, args)
		default:
			log.Fatal("[CMD] unknown app mode\n")
		}
	},
}

func init() {
	rootCmd.Version = Version
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

// Execute 执行根命令
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("[CMD] execute failed; %s\n", err)
	}
}
