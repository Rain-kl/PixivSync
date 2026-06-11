// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

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
