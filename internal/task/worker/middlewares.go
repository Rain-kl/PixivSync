/*
Copyright 2025-2026 linux.do
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

// Package worker 提供 Asynq 任务处理服务器与中间件
package worker

import (
	"context"

	"github.com/hibiken/asynq"
)

// taskLoggingMiddleware 任务处理中间件
// 注意：OTel Span 创建、日志记录、TaskExecution 状态管理
// 已由 task.ProcessTask 统一处理，此中间件保留用于未来扩展（如限流、监控等）
func taskLoggingMiddleware(h asynq.Handler) asynq.Handler {
	return asynq.HandlerFunc(func(ctx context.Context, t *asynq.Task) error {
		return h.ProcessTask(ctx, t)
	})
}
