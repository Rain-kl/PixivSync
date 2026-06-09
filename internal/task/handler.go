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

package task

import "context"

// TaskResult 任务执行结果
//nolint:revive // TaskResult 保留完整名称以避免与通用 Result 混淆
type TaskResult struct {
	Message string // 结果摘要，如 "共清理 120 个文件，耗时 3.2s"
	Detail  string // 可选的详细结果 JSON
}

// PayloadValidator 可选接口，带参数的任务 Handler 应实现此接口。
// 框架在 Admin 下发时自动调用，完成参数校验和标准化（如 Trim 空白）。
// 无参数的任务无需实现，框架会直接透传 payload。
type PayloadValidator interface {
	ValidatePayload(payload []byte) ([]byte, error)
}

// TaskHandler 异步任务处理器接口
// 所有异步任务必须实现此接口，框架将自动管理任务执行记录的创建、状态流转和日志写入。
//
// 开发者只需实现 Execute 方法编写业务逻辑，在方法内通过 task.AppendLog(ctx, ...) 追加执行日志。
// 任务的创建、状态更新、错误记录、重试计数全部由框架透明处理。
//
//nolint:revive // TaskHandler 保留完整名称以避免与通用 Handler 混淆
type TaskHandler interface {
	// Execute 执行任务业务逻辑
	//   - ctx: 已注入 Trace Span 和 taskID 的上下文
	//   - payload: 调度时传入的原始参数（可为 nil）
	//   - 返回 TaskResult 描述执行结果，或 error 表示执行失败
	Execute(ctx context.Context, payload []byte) (*TaskResult, error)
}
