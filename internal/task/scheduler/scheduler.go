// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/Rain-kl/Wavelet/internal/config"
	"github.com/Rain-kl/Wavelet/internal/task"

	"github.com/hibiken/asynq"
)

const (
	cleanupDedupWindow = 23 * time.Hour // 清理任务去重窗口
	cleanupMaxRetry    = 3              // 清理任务最大重试次数
)

var (
	scheduler     *asynq.Scheduler
	schedulerOnce sync.Once
)

func init() {
	// AsynqClient 已在 task 包中初始化
}

// GetAsynqClient 获取全局 AsynqClient
func GetAsynqClient() *asynq.Client {
	return task.AsynqClient
}

// StartScheduler 启动调度器
func StartScheduler() error {
	var err error
	schedulerOnce.Do(func() {
		location, locErr := time.LoadLocation("Asia/Shanghai")
		if locErr != nil {
			err = fmt.Errorf(errLoadLocationFailed, locErr)
			return
		}
		scheduler = asynq.NewScheduler(
			task.RedisOpt,
			&asynq.SchedulerOpts{
				Location: location,
			},
		)

		// 清理未使用的上传文件任务
		if _, err = scheduler.Register(
			config.Config.Scheduler.CleanupUnusedUploadsTaskCron,
			asynq.NewTask(task.CleanupUnusedUploadsTask, nil),
			asynq.Unique(cleanupDedupWindow),
			asynq.MaxRetry(cleanupMaxRetry),
		); err != nil {
			return
		}

		// 启动调度器
		err = scheduler.Run()
	})
	return err
}
