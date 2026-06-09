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

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db/idgen"
	"github.com/Rain-kl/Wavelet/internal/logger"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/otel_trace"
	"github.com/hibiken/asynq"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// handlerRegistry 已注册的任务处理器
var handlerRegistry = make(map[string]TaskHandler)

// RegisterHandler 注册任务处理器
// 传入任务类型标识（对应 constants.go 中的 AsynqTask 常量）和 TaskHandler 实现
func RegisterHandler(asynqTaskType string, handler TaskHandler) {
	handlerRegistry[asynqTaskType] = handler
}

// getHandler 获取已注册的处理器
func getHandler(asynqTaskType string) (TaskHandler, bool) {
	h, ok := handlerRegistry[asynqTaskType]
	return h, ok
}

// ValidateAndNormalizePayload 校验并标准化任务参数。
// 如果 Handler 实现了 PayloadValidator，调用其 ValidatePayload 方法；
// 否则直接返回原始 payload。
func ValidateAndNormalizePayload(asynqTaskType string, payload []byte) ([]byte, error) {
	handler, ok := getHandler(asynqTaskType)
	if !ok {
		return payload, nil
	}
	if validator, ok := handler.(PayloadValidator); ok {
		return validator.ValidatePayload(payload)
	}
	return payload, nil
}

// contextKey 用于 context 存取 taskID
type contextKey string

const taskIDKey contextKey = "task_execution_task_id"

// withTaskID 将 taskID 注入 context
func withTaskID(ctx context.Context, taskID string) context.Context {
	return context.WithValue(ctx, taskIDKey, taskID)
}

// GetTaskID 从 context 中获取 taskID
func GetTaskID(ctx context.Context) string {
	if v, ok := ctx.Value(taskIDKey).(string); ok {
		return v
	}
	return ""
}

// AppendLog 追加日志到任务执行记录
// 在 TaskHandler.Execute 中调用，日志会自动追加到 TaskExecution.Log 字段
func AppendLog(ctx context.Context, format string, args ...interface{}) {
	taskID := GetTaskID(ctx)
	if taskID == "" {
		// 上下文中没有 taskID，降级到普通日志
		logger.InfoF(ctx, format, args...)
		return
	}

	logLine := fmt.Sprintf(format, args...)
	if err := model.AppendTaskExecutionLog(ctx, taskID, logLine); err != nil {
		logger.ErrorF(ctx, "[TaskExecutor] 追加任务日志失败 taskID=%s: %v", taskID, err)
	}
}

// DispatchTask 下发任务（创建 TaskExecution 记录 → 入队 Asynq）
func DispatchTask(ctx context.Context, taskType string, payload []byte, triggeredBy string) (string, error) {
	meta := GetTaskMeta(taskType)
	if meta == nil {
		return "", fmt.Errorf(errUnknownTaskType, taskType)
	}

	// 生成唯一的 TaskID
	taskID := generateTaskID(taskType, triggeredBy)

	// 创建任务执行记录
	execution := &model.TaskExecution{
		TaskID:      taskID,
		TaskType:    meta.AsynqTask,
		TaskName:    meta.Name,
		Status:      model.TaskExecutionStatusPending,
		Retryable:   meta.Retryable,
		MaxRetry:    meta.MaxRetry,
		RetryCount:  0,
		Payload:     string(payload),
		TriggeredBy: triggeredBy,
	}

	if err := model.CreateTaskExecution(ctx, execution); err != nil {
		return "", fmt.Errorf(errCreateTaskExecutionFailed, err)
	}

	// 入队 Asynq
	taskInfo := asynq.NewTask(meta.AsynqTask, payload)
	if _, err := AsynqClient.Enqueue(
		taskInfo,
		asynq.TaskID(taskID),
		asynq.MaxRetry(meta.MaxRetry),
		asynq.Queue(meta.Queue),
	); err != nil {
		// 入队失败，更新执行记录状态
		execution.Status = model.TaskExecutionStatusFailed
		execution.ErrorMessage = fmt.Sprintf("入队失败: %v", err)
		now := time.Now()
		execution.StartedAt = &now
		execution.FinishedAt = &now
		_ = model.UpdateTaskExecution(ctx, execution)
		return "", fmt.Errorf(errTaskEnqueueFailed, err)
	}

	return taskID, nil
}

// RetryTask 重试失败的任务
func RetryTask(ctx context.Context, id uint64) (string, error) {
	execution, err := model.GetTaskExecutionByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf(errTaskExecutionNotFound, err)
	}

	if execution.Status != model.TaskExecutionStatusFailed {
		return "", fmt.Errorf(errRetryOnlyFailedTask, execution.Status)
	}

	if !execution.Retryable {
		return "", errors.New(errTaskNotRetryable)
	}

	if execution.RetryCount >= execution.MaxRetry {
		return "", fmt.Errorf(errTaskMaxRetryExceeded, execution.MaxRetry)
	}

	// 生成新的 TaskID
	newTaskID := generateRetryTaskID(execution.TaskID, execution.RetryCount+1)

	// 创建新的执行记录
	newExecution := &model.TaskExecution{
		TaskID:      newTaskID,
		TaskType:    execution.TaskType,
		TaskName:    execution.TaskName,
		Status:      model.TaskExecutionStatusPending,
		Retryable:   execution.Retryable,
		MaxRetry:    execution.MaxRetry,
		RetryCount:  execution.RetryCount + 1,
		Payload:     execution.Payload,
		TriggeredBy: "retry",
	}

	if err := model.CreateTaskExecution(ctx, newExecution); err != nil {
		return "", fmt.Errorf(errCreateRetryExecutionFailed, err)
	}

	// 入队 Asynq
	taskInfo := asynq.NewTask(execution.TaskType, []byte(execution.Payload))
	if _, err := AsynqClient.Enqueue(
		taskInfo,
		asynq.TaskID(newTaskID),
		asynq.MaxRetry(execution.MaxRetry),
		asynq.Queue(PrefixedQueue(QueueDefault)),
	); err != nil {
		newExecution.Status = model.TaskExecutionStatusFailed
		newExecution.ErrorMessage = fmt.Sprintf("重试入队失败: %v", err)
		now := time.Now()
		newExecution.StartedAt = &now
		newExecution.FinishedAt = &now
		_ = model.UpdateTaskExecution(ctx, newExecution)
		return "", fmt.Errorf(errRetryTaskEnqueueFailed, err)
	}

	return newTaskID, nil
}

// ProcessTask Asynq 实际调用的统一处理函数
// Worker 注册时统一使用此函数，内部自动分发到对应的 TaskHandler
func ProcessTask(ctx context.Context, t *asynq.Task) error {
	// 初始化 Trace
	ctx, span := otel_trace.Start(ctx, "TaskProcess_"+t.Type(), trace.WithSpanKind(trace.SpanKindConsumer))
	defer span.End()

	// 添加任务信息到 Span
	span.SetAttributes(
		attribute.String("task.type", t.Type()),
		attribute.Int("task.payload_size", len(t.Payload())),
		attribute.String("task.id", t.ResultWriter().TaskID()),
	)

	taskID := t.ResultWriter().TaskID()

	// 注入 taskID 到 context
	ctx = withTaskID(ctx, taskID)

	// 查找处理器
	handler, ok := getHandler(t.Type())
	if !ok {
		err := fmt.Errorf(errUnregisteredTaskHandler, t.Type())
		logger.ErrorF(ctx, "[TaskExecutor] %v", err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	// 从数据库加载执行记录
	execution, err := model.GetTaskExecutionByTaskID(ctx, taskID)
	if err != nil {
		logger.ErrorF(ctx, "[TaskExecutor] 查询执行记录失败 taskID=%s: %v", taskID, err)
		// 执行记录不存在，仍然执行任务但不记录状态
		_, execErr := handler.Execute(ctx, t.Payload())
		if execErr != nil {
			span.SetStatus(codes.Error, execErr.Error())
			return execErr
		}
		return nil
	}

	// 更新状态为 running
	now := time.Now()
	execution.Status = model.TaskExecutionStatusRunning
	execution.StartedAt = &now
	if err := model.UpdateTaskExecution(ctx, execution); err != nil {
		logger.ErrorF(ctx, "[TaskExecutor] 更新执行状态失败 taskID=%s: %v", taskID, err)
	}

	// 开始计时
	start := time.Now()

	// 执行业务逻辑
	result, execErr := handler.Execute(ctx, t.Payload())

	// 计算耗时
	duration := time.Since(start)
	finishTime := time.Now()
	execution.Duration = duration.Milliseconds()
	execution.FinishedAt = &finishTime

	if execErr != nil {
		// 执行失败
		execution.Status = model.TaskExecutionStatusFailed
		execution.ErrorMessage = execErr.Error()

		logger.ErrorF(ctx,
			"[TaskExecutor] 任务处理失败 Type: %s TaskID: %s Duration: %d ms Error: %v",
			t.Type(), taskID, duration.Milliseconds(), execErr,
		)

		span.SetStatus(codes.Error, execErr.Error())
		span.RecordError(execErr)
	} else {
		// 执行成功
		execution.Status = model.TaskExecutionStatusSucceeded
		if result != nil {
			execution.Result = result.Message
			if result.Detail != "" {
				execution.Result = fmt.Sprintf("%s\n%s", result.Message, result.Detail)
			}
		}

		logger.InfoF(ctx,
			"[TaskExecutor] 任务处理完成 Type: %s TaskID: %s Duration: %d ms",
			t.Type(), taskID, duration.Milliseconds(),
		)
	}

	// 更新执行记录
	if err := model.UpdateTaskExecution(ctx, execution); err != nil {
		logger.ErrorF(ctx, "[TaskExecutor] 更新执行记录失败 taskID=%s: %v", taskID, err)
	}

	return execErr
}

// generateTaskID 生成任务 ID
func generateTaskID(taskType string, triggeredBy string) string {
	uniqueID := idgen.NextUint64ID()
	return fmt.Sprintf("%s_%s_%d", triggeredBy, taskType, uniqueID)
}

// generateRetryTaskID 生成重试任务 ID
func generateRetryTaskID(originalTaskID string, retryCount int) string {
	return fmt.Sprintf("retry_%d_%s", retryCount, originalTaskID)
}
