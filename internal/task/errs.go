/*
Copyright 2026 Arctel.net

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

const (
	errUnknownTaskType            = "未知的任务类型: %s"
	errCreateTaskExecutionFailed  = "创建任务执行记录失败: %w"
	errTaskEnqueueFailed          = "任务入队失败: %w"
	errTaskExecutionNotFound      = "任务执行记录不存在: %w"
	errRetryOnlyFailedTask        = "只有失败的任务才能重试，当前状态: %s"
	errTaskNotRetryable           = "该任务不支持重试"
	errTaskMaxRetryExceeded       = "已达到最大重试次数 %d"
	errCreateRetryExecutionFailed = "创建重试任务执行记录失败: %w"
	errRetryTaskEnqueueFailed     = "重试任务入队失败: %w"
	errUnregisteredTaskHandler    = "未注册的任务处理器: %s"
)
