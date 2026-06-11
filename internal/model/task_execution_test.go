// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"context"
	"testing"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTaskExecutionTestEnvironment(t *testing.T) func() {
	sqliteDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	err = sqliteDB.AutoMigrate(&TaskExecution{})
	require.NoError(t, err)

	db.SetDB(sqliteDB)

	return func() {
		db.SetDB(nil)
	}
}

func TestCreateTaskExecution(t *testing.T) {
	cleanup := setupTaskExecutionTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	execution := &TaskExecution{
		TaskID:      "manual_cleanup_123",
		TaskType:    "upload:cleanup_unused",
		TaskName:    "清理未使用上传",
		Status:      TaskExecutionStatusPending,
		Retryable:   true,
		MaxRetry:    3,
		RetryCount:  0,
		Payload:     `{"test": true}`,
		TriggeredBy: "manual",
	}

	err := CreateTaskExecution(ctx, execution)
	require.NoError(t, err)
	assert.NotZero(t, execution.ID, "ID should be generated")
	assert.NotZero(t, execution.CreatedAt, "CreatedAt should be set")
	assert.NotZero(t, execution.UpdatedAt, "UpdatedAt should be set")
}

func TestGetTaskExecutionByTaskID(t *testing.T) {
	cleanup := setupTaskExecutionTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	// 创建记录
	execution := &TaskExecution{
		TaskID:      "test_task_id_001",
		TaskType:    "upload:cleanup_unused",
		TaskName:    "清理未使用上传",
		Status:      TaskExecutionStatusPending,
		Retryable:   true,
		MaxRetry:    3,
		TriggeredBy: "manual",
	}
	err := CreateTaskExecution(ctx, execution)
	require.NoError(t, err)

	// 按 TaskID 查询
	found, err := GetTaskExecutionByTaskID(ctx, "test_task_id_001")
	require.NoError(t, err)
	assert.Equal(t, execution.ID, found.ID)
	assert.Equal(t, "test_task_id_001", found.TaskID)
	assert.Equal(t, TaskExecutionStatusPending, found.Status)
	assert.True(t, found.Retryable)
	assert.Equal(t, 3, found.MaxRetry)

	// 查询不存在的 TaskID
	_, err = GetTaskExecutionByTaskID(ctx, "nonexistent")
	assert.Error(t, err, "should return error for non-existent taskID")
}

func TestGetTaskExecutionByID(t *testing.T) {
	cleanup := setupTaskExecutionTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	execution := &TaskExecution{
		TaskID:      "test_by_id_001",
		TaskType:    "upload:cleanup_unused",
		TaskName:    "清理未使用上传",
		Status:      TaskExecutionStatusPending,
		TriggeredBy: "system",
	}
	err := CreateTaskExecution(ctx, execution)
	require.NoError(t, err)

	// 按主键查询
	found, err := GetTaskExecutionByID(ctx, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, execution.TaskID, found.TaskID)
}

func TestUpdateTaskExecution(t *testing.T) {
	cleanup := setupTaskExecutionTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	// 创建记录
	execution := &TaskExecution{
		TaskID:      "test_update_001",
		TaskType:    "upload:cleanup_unused",
		TaskName:    "清理未使用上传",
		Status:      TaskExecutionStatusPending,
		TriggeredBy: "manual",
	}
	err := CreateTaskExecution(ctx, execution)
	require.NoError(t, err)

	// 更新状态为 running
	now := time.Now()
	execution.Status = TaskExecutionStatusRunning
	execution.StartedAt = &now
	err = UpdateTaskExecution(ctx, execution)
	require.NoError(t, err)

	// 验证更新
	found, err := GetTaskExecutionByTaskID(ctx, "test_update_001")
	require.NoError(t, err)
	assert.Equal(t, TaskExecutionStatusRunning, found.Status)
	assert.NotNil(t, found.StartedAt)

	// 更新为 succeeded
	finishTime := time.Now()
	execution.Status = TaskExecutionStatusSucceeded
	execution.FinishedAt = &finishTime
	execution.Duration = 1500
	execution.Result = "共清理 50 个文件"
	err = UpdateTaskExecution(ctx, execution)
	require.NoError(t, err)

	found, err = GetTaskExecutionByTaskID(ctx, "test_update_001")
	require.NoError(t, err)
	assert.Equal(t, TaskExecutionStatusSucceeded, found.Status)
	assert.Equal(t, int64(1500), found.Duration)
	assert.Equal(t, "共清理 50 个文件", found.Result)
}

func TestUpdateTaskExecutionFailed(t *testing.T) {
	cleanup := setupTaskExecutionTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	execution := &TaskExecution{
		TaskID:      "test_fail_001",
		TaskType:    "upload:cleanup_unused",
		TaskName:    "清理未使用上传",
		Status:      TaskExecutionStatusPending,
		Retryable:   true,
		MaxRetry:    3,
		TriggeredBy: "manual",
	}
	err := CreateTaskExecution(ctx, execution)
	require.NoError(t, err)

	// 标记为失败
	now := time.Now()
	execution.Status = TaskExecutionStatusFailed
	execution.StartedAt = &now
	execution.FinishedAt = &now
	execution.Duration = 200
	execution.ErrorMessage = "S3 连接超时"
	err = UpdateTaskExecution(ctx, execution)
	require.NoError(t, err)

	found, err := GetTaskExecutionByTaskID(ctx, "test_fail_001")
	require.NoError(t, err)
	assert.Equal(t, TaskExecutionStatusFailed, found.Status)
	assert.Equal(t, "S3 连接超时", found.ErrorMessage)
	assert.Equal(t, int64(200), found.Duration)
}

func TestUpdateTaskExecutionDoesNotOverwriteLog(t *testing.T) {
	cleanup := setupTaskExecutionTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	execution := &TaskExecution{
		TaskID:      "test_omit_log_001",
		TaskType:    "upload:cleanup_unused",
		TaskName:    "清理未使用上传",
		Status:      TaskExecutionStatusPending,
		TriggeredBy: "manual",
	}
	err := CreateTaskExecution(ctx, execution)
	require.NoError(t, err)

	// In a real execution, logs are appended to the DB asynchronously via AppendTaskExecutionLog
	err = AppendTaskExecutionLog(ctx, "test_omit_log_001", "第一条执行日志")
	require.NoError(t, err)

	// The local struct still has empty Log because it was not reloaded
	assert.Empty(t, execution.Log)

	// Now complete/update the execution (e.g. status, duration)
	execution.Status = TaskExecutionStatusSucceeded
	execution.Duration = 100
	err = UpdateTaskExecution(ctx, execution)
	require.NoError(t, err)

	// Get the updated execution record and check that the Log was NOT overwritten/wiped
	found, err := GetTaskExecutionByTaskID(ctx, "test_omit_log_001")
	require.NoError(t, err)
	assert.Equal(t, TaskExecutionStatusSucceeded, found.Status)
	assert.Contains(t, found.Log, "第一条执行日志")
}

func TestAppendTaskExecutionLog(t *testing.T) {
	cleanup := setupTaskExecutionTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	execution := &TaskExecution{
		TaskID:      "test_log_001",
		TaskType:    "upload:cleanup_unused",
		TaskName:    "清理未使用上传",
		Status:      TaskExecutionStatusPending,
		TriggeredBy: "manual",
	}
	err := CreateTaskExecution(ctx, execution)
	require.NoError(t, err)

	// 追加多条日志
	err = AppendTaskExecutionLog(ctx, "test_log_001", "开始扫描未使用上传文件")
	require.NoError(t, err)

	err = AppendTaskExecutionLog(ctx, "test_log_001", "本批次找到 42 个待清理文件")
	require.NoError(t, err)

	err = AppendTaskExecutionLog(ctx, "test_log_001", "清理完成，共删除 42 个文件")
	require.NoError(t, err)

	// 验证日志内容
	found, err := GetTaskExecutionByTaskID(ctx, "test_log_001")
	require.NoError(t, err)
	assert.Contains(t, found.Log, "开始扫描未使用上传文件")
	assert.Contains(t, found.Log, "本批次找到 42 个待清理文件")
	assert.Contains(t, found.Log, "清理完成，共删除 42 个文件")
}

func TestAppendTaskExecutionLogNonExistent(t *testing.T) {
	cleanup := setupTaskExecutionTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	// 对不存在的 TaskID 追加日志不应报错（COALESCE 处理空值）
	err := AppendTaskExecutionLog(ctx, "nonexistent_task", "测试日志")
	// SQLite 下 COALESCE + || 操作不应报错
	assert.NoError(t, err)
}

func TestListTaskExecutions(t *testing.T) {
	cleanup := setupTaskExecutionTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	// 创建多条记录，包含不同状态和类型
	records := []*TaskExecution{
		{TaskID: "list_001", TaskType: "upload:cleanup_unused", TaskName: "清理上传", Status: TaskExecutionStatusSucceeded, TriggeredBy: "manual"},
		{TaskID: "list_002", TaskType: "upload:cleanup_unused", TaskName: "清理上传", Status: TaskExecutionStatusFailed, TriggeredBy: "system"},
		{TaskID: "list_003", TaskType: "other:task", TaskName: "其他任务", Status: TaskExecutionStatusPending, TriggeredBy: "manual"},
		{TaskID: "list_004", TaskType: "upload:cleanup_unused", TaskName: "清理上传", Status: TaskExecutionStatusRunning, TriggeredBy: "manual"},
		{TaskID: "list_005", TaskType: "other:task", TaskName: "其他任务", Status: TaskExecutionStatusSucceeded, TriggeredBy: "system"},
	}
	for _, r := range records {
		err := CreateTaskExecution(ctx, r)
		require.NoError(t, err)
	}

	// 查询全部（分页）
	items, total, err := ListTaskExecutions(ctx, ListTaskExecutionsRequest{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, items, 5)

	// 按状态筛选：failed
	items, total, err = ListTaskExecutions(ctx, ListTaskExecutionsRequest{Status: "failed", Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, items, 1)
	assert.Equal(t, "list_002", items[0].TaskID)

	// 按类型筛选
	_, total, err = ListTaskExecutions(ctx, ListTaskExecutionsRequest{TaskType: "other:task", Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)

	// 分页测试
	items, total, err = ListTaskExecutions(ctx, ListTaskExecutionsRequest{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, items, 2)

	items2, total2, err := ListTaskExecutions(ctx, ListTaskExecutionsRequest{Page: 2, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(5), total2)
	assert.Len(t, items2, 2)

	// 确保分页数据不重复
	assert.NotEqual(t, items[0].ID, items2[0].ID)

	// 状态 + 类型组合筛选
	items, total, err = ListTaskExecutions(ctx, ListTaskExecutionsRequest{Status: "succeeded", TaskType: "upload:cleanup_unused", Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, "list_001", items[0].TaskID)
}

func TestListTaskExecutionsDefaultPaging(t *testing.T) {
	cleanup := setupTaskExecutionTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	// 不传分页参数，应使用默认值 page=1, pageSize=20
	items, total, err := ListTaskExecutions(ctx, ListTaskExecutionsRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Len(t, items, 0)
}

func TestTaskExecutionTableName(t *testing.T) {
	execution := TaskExecution{}
	assert.Equal(t, "w_task_executions", execution.TableName())
}
