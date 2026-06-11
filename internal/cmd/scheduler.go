// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package cmd

import (
	"log"

	"github.com/Rain-kl/Wavelet/internal/task/scheduler"

	"github.com/spf13/cobra"
)

var schedulerCmd = &cobra.Command{
	Use:   "scheduler",
	Short: "wavelet Scheduler",
	Run: func(_ *cobra.Command, _ []string) {
		log.Println("[Scheduler] 启动定时任务调度服务")
		if err := scheduler.StartScheduler(); err != nil {
			log.Fatalf("[调度器] 启动失败: %v", err)
		}
	},
}
