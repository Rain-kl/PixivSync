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

// Package cmd 提供 CLI 命令入口
package cmd

import (
	"log"
	"sync"

	"github.com/Rain-kl/Wavelet/internal/router"
	"github.com/Rain-kl/Wavelet/internal/task/scheduler"
	"github.com/Rain-kl/Wavelet/internal/task/worker"
	"github.com/spf13/cobra"
)

var allCmd = &cobra.Command{
	Use:   "all",
	Short: "以融合模式同时启动 API、Worker 和 Scheduler",
	Run: func(_ *cobra.Command, _ []string) {
		log.Println("[All] 融合模式启动")

		var wg sync.WaitGroup

		// 启动 API HTTP 服务
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Println("[All] 启动 API 服务")
			router.Serve()
		}()

		// 启动 Asynq Worker 任务处理服务
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Println("[All] 启动 Worker 服务")
			if err := worker.StartWorker(); err != nil {
				log.Printf("[All] Worker 启动失败: %v\n", err)
			}
		}()

		// 启动 Asynq 定时任务调度器
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Println("[All] 启动 Scheduler 服务")
			if err := scheduler.StartScheduler(); err != nil {
				log.Printf("[All] Scheduler 启动失败: %v\n", err)
			}
		}()

		wg.Wait()
	},
}
