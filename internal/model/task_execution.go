// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"context"
	"fmt"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/db/idgen"
	"gorm.io/gorm"
)

// TaskExecutionStatus 任务执行状态
type TaskExecutionStatus string

// 任务执行状态
const (
	TaskExecutionStatusPending   TaskExecutionStatus = "pending"
	TaskExecutionStatusRunning   TaskExecutionStatus = "running"
	TaskExecutionStatusSucceeded TaskExecutionStatus = "succeeded"
	TaskExecutionStatusFailed    TaskExecutionStatus = "failed"
)

// TaskExecution 任务执行记录
type TaskExecution struct {
	ID           uint64              `json:"id,string" gorm:"primaryKey"`
	TaskID       string              `json:"task_id" gorm:"size:128;uniqueIndex;not null"`
	TaskType     string              `json:"task_type" gorm:"size:64;index;not null"`
	TaskName     string              `json:"task_name" gorm:"size:128"`
	Status       TaskExecutionStatus `json:"status" gorm:"size:32;index;not null"`
	Retryable    bool                `json:"retryable" gorm:"not null;default:false"`
	MaxRetry     int                 `json:"max_retry" gorm:"not null;default:0"`
	RetryCount   int                 `json:"retry_count" gorm:"not null;default:0"`
	Log          string              `json:"log" gorm:"type:text"`
	ErrorMessage string              `json:"error_message" gorm:"type:text"`
	Result       string              `json:"result" gorm:"type:text"`
	StartedAt    *time.Time          `json:"started_at" gorm:"index"`
	FinishedAt   *time.Time          `json:"finished_at"`
	Duration     int64               `json:"duration" gorm:"comment:耗时毫秒"`
	Payload      string              `json:"payload" gorm:"type:text"`
	TriggeredBy  string              `json:"triggered_by" gorm:"size:32;not null;default:system"`
	CreatedAt    time.Time           `json:"created_at" gorm:"autoCreateTime;index"`
	UpdatedAt    time.Time           `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 表名
func (TaskExecution) TableName() string {
	return "task_executions"
}

// CreateTaskExecution 创建任务执行记录
func CreateTaskExecution(ctx context.Context, execution *TaskExecution) error {
	execution.ID = idgen.NextUint64ID()
	return db.DB(ctx).Create(execution).Error
}

// UpdateTaskExecution 更新任务执行记录
func UpdateTaskExecution(ctx context.Context, execution *TaskExecution) error {
	return db.DB(ctx).Save(execution).Error
}

// GetTaskExecutionByTaskID 根据 TaskID 获取执行记录
func GetTaskExecutionByTaskID(ctx context.Context, taskID string) (*TaskExecution, error) {
	var execution TaskExecution
	if err := db.DB(ctx).Where("task_id = ?", taskID).First(&execution).Error; err != nil {
		return nil, err
	}
	return &execution, nil
}

// GetTaskExecutionByID 根据 ID 获取执行记录
func GetTaskExecutionByID(ctx context.Context, id uint64) (*TaskExecution, error) {
	var execution TaskExecution
	if err := db.DB(ctx).Where("id = ?", id).First(&execution).Error; err != nil {
		return nil, err
	}
	return &execution, nil
}

// AppendTaskExecutionLog 追加日志到执行记录
func AppendTaskExecutionLog(ctx context.Context, taskID string, logLine string) error {
	now := time.Now().Format("15:04:05")
	line := fmt.Sprintf("[%s] %s\n", now, logLine)
	return db.DB(ctx).Model(&TaskExecution{}).
		Where("task_id = ?", taskID).
		Update("log", gorm.Expr("COALESCE(log, '') || ?", line)).Error
}

// ListTaskExecutionsRequest 查询任务执行记录列表请求
type ListTaskExecutionsRequest struct {
	Status   string `form:"status"`
	TaskType string `form:"task_type"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

// ListTaskExecutions 分页查询任务执行记录
func ListTaskExecutions(ctx context.Context, req ListTaskExecutionsRequest) ([]TaskExecution, int64, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	query := db.DB(ctx).Model(&TaskExecution{})

	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.TaskType != "" {
		query = query.Where("task_type = ?", req.TaskType)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var executions []TaskExecution
	offset := (req.Page - 1) * req.PageSize
	if err := query.Order("id DESC").Offset(offset).Limit(req.PageSize).Find(&executions).Error; err != nil {
		return nil, 0, err
	}

	return executions, total, nil
}
