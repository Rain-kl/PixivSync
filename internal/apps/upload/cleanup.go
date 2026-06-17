// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

// Package upload implements upload tasks and file cleanup services.
package upload

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/storage"
	"github.com/Rain-kl/Wavelet/internal/task"
	"github.com/Rain-kl/Wavelet/pkg/logger"
	"gorm.io/gorm"
)

// 异步任务名称与管理类型定义
const (
	// SystemCleanupTask 系统定期垃圾清理任务标识
	SystemCleanupTask = "system:cleanup"
	// TaskTypeSystemCleanup 系统定期垃圾清理管理类型
	TaskTypeSystemCleanup = "system_cleanup"

	// 错误描述常量
	errStorageReadOnly          = "存储迁移维护中，当前仅允许读取文件"
	errQueryUnusedUploadsFailed = "查询未使用的上传文件失败: %w"
)

// SystemCleanupMeta represents the task metadata.
var SystemCleanupMeta = task.TaskMeta{
	Type:         TaskTypeSystemCleanup,
	AsynqTask:    SystemCleanupTask,
	Name:         "系统垃圾清理",
	Description:  "定期清理未使用上传文件、历史推送记录和过期任务执行日志",
	SupportsTime: false,
	MaxRetry:     task.DefaultMaxRetry,
	Queue:        task.QueueDefault,
	Retryable:    true,
}

// SystemCleanupHandler 系统定期垃圾清理异步任务处理器
type SystemCleanupHandler struct{}

// Execute 执行系统清理（包含文件清理、历史推送日志和任务执行日志清理）
func (h *SystemCleanupHandler) Execute(ctx context.Context, _ []byte) (*task.TaskResult, error) {
	if storageReadOnly(ctx) {
		return nil, errors.New(errStorageReadOnly)
	}
	const batchSize = 100 // 每批处理100个文件
	var lastID uint64
	var totalProcessed int
	var totalDeleted int

	// 计算1小时前的时间
	oneHourAgo := time.Now().Add(-1 * time.Hour)

	task.AppendLog(ctx, "开始扫描未使用上传文件，阈值: %s", oneHourAgo.Format(time.RFC3339))

	for {
		// 使用游标分页查询未使用且超过1小时的上传记录
		var unusedUploads []model.Upload
		if err := db.DB(ctx).
			Where("id > ? AND status = ? AND created_at < ?", lastID, model.UploadStatusPending, oneHourAgo).
			Order("id ASC").
			Limit(batchSize).
			Find(&unusedUploads).Error; err != nil {
			task.AppendLog(ctx, "查询未使用的上传文件失败: %v", err)
			return nil, fmt.Errorf(errQueryUnusedUploadsFailed, err)
		}

		// 没有更多数据，退出循环
		if len(unusedUploads) == 0 {
			break
		}

		task.AppendLog(ctx, "本批次找到 %d 个需要清理的上传文件", len(unusedUploads))

		// 处理每个未使用的上传文件
		for _, u := range unusedUploads {
			totalProcessed++

			if err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
				// 更新上传记录状态
				if err := tx.Model(&model.Upload{}).
					Where("id = ? AND status = ?", u.ID, model.UploadStatusPending).
					Update("status", model.UploadStatusDeleted).Error; err != nil {
					return err
				}

				driver := storage.Driver(u.StorageDriver)
				if driver == "" {
					driver = storage.DriverLocal
				}
				backend, err := storage.ForDriver(ctx, driver)
				if err != nil {
					return err
				}
				if err := backend.Delete(ctx, u.FilePath); err != nil {
					return err
				}

				return nil
			}); err != nil {
				task.AppendLog(ctx, "清理上传文件失败 [ID:%d]: %v", u.ID, err)
				lastID = u.ID
				continue
			}

			recordUploadStatsRemove(ctx, &u)
			totalDeleted++
			lastID = u.ID
		}
	}

	// 2. 清理超过7天的历史推送日志
	task.AppendLog(ctx, "开始清理历史推送审计日志，只保留最近7天数据...")
	cutoff := time.Now().AddDate(0, 0, -7)
	var pushHistoryCount int64
	if err := db.DB(ctx).Model(&model.PushHistory{}).Where("created_at < ?", cutoff).Count(&pushHistoryCount).Error; err != nil {
		task.AppendLog(ctx, "统计待清理的历史推送记录失败: %v", err)
	} else if pushHistoryCount > 0 {
		if err := db.DB(ctx).Where("created_at < ?", cutoff).Delete(&model.PushHistory{}).Error; err != nil {
			task.AppendLog(ctx, "删除历史推送记录失败: %v", err)
		} else {
			task.AppendLog(ctx, "成功删除 %d 条历史推送记录 (截止时间: %s)", pushHistoryCount, cutoff.Format("2006-01-02 15:04:05"))
		}
	} else {
		task.AppendLog(ctx, "没有需要清理的历史推送记录 (截止时间: %s)", cutoff.Format("2006-01-02 15:04:05"))
	}

	// 3. 清理任务执行日志：高频任务保留3天，低频任务保留30天。
	task.AppendLog(ctx, "开始清理任务执行日志：高频任务保留最近3天，低频任务保留最近30天...")
	taskLogStats, err := model.CleanupTaskExecutionLogs(ctx, time.Now())
	if err != nil {
		task.AppendLog(ctx, "清理任务执行日志失败: %v", err)
		logger.ErrorF(ctx, "清理任务执行日志失败: %v", err)
	} else {
		task.AppendLog(ctx, "成功清理任务执行日志 %d 条（高频 %d 条，低频 %d 条）",
			taskLogStats.HighFrequencyDeleted+taskLogStats.LowFrequencyDeleted,
			taskLogStats.HighFrequencyDeleted,
			taskLogStats.LowFrequencyDeleted,
		)
	}

	msg := fmt.Sprintf("系统清理完成。成功清理未使用的上传文件 %d/%d 个；清理历史推送审计日志 %d 条；清理任务执行日志 %d 条。",
		totalDeleted,
		totalProcessed,
		pushHistoryCount,
		taskLogStats.HighFrequencyDeleted+taskLogStats.LowFrequencyDeleted,
	)
	task.AppendLog(ctx, "%s", msg)
	return &task.TaskResult{Message: msg}, nil
}

func storageReadOnly(ctx context.Context) bool {
	var execution model.TaskExecution
	err := db.DB(ctx).Where("task_type = ?", "storage:migrate").Order("id DESC").First(&execution).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false
		}
		logger.ErrorF(ctx, "读取存储维护状态失败: %v", err)
		return true
	}
	return execution.Status != model.TaskExecutionStatusSucceeded
}
