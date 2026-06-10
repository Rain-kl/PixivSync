// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package upload

import (
	"context"
	"fmt"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/storage"
	"github.com/Rain-kl/Wavelet/internal/task"
	"gorm.io/gorm"
)

// CleanupUnusedUploadsHandler 清理未使用上传文件的异步任务处理器
type CleanupUnusedUploadsHandler struct{}

// Execute 执行清理未使用上传文件的业务逻辑
func (h *CleanupUnusedUploadsHandler) Execute(ctx context.Context, _ []byte) (*task.TaskResult, error) {
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
			return nil, fmt.Errorf(ErrQueryUnusedUploadsFailed, err)
		}

		// 没有更多数据，退出循环
		if len(unusedUploads) == 0 {
			break
		}

		task.AppendLog(ctx, "本批次找到 %d 个需要清理的上传文件", len(unusedUploads))

		// 处理每个未使用的上传文件
		for _, upload := range unusedUploads {
			totalProcessed++

			if err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
				// 更新上传记录状态
				if err := tx.Model(&model.Upload{}).
					Where("id = ? AND status = ?", upload.ID, model.UploadStatusPending).
					Update("status", model.UploadStatusDeleted).Error; err != nil {
					return err
				}

				// Delete from S3
				if err := storage.DeleteObject(ctx, upload.FilePath); err != nil {
					return err
				}

				return nil
			}); err != nil {
				task.AppendLog(ctx, "清理上传文件失败 [ID:%d]: %v", upload.ID, err)
				lastID = upload.ID
				continue
			}

			totalDeleted++
			lastID = upload.ID
		}
	}

	msg := fmt.Sprintf("共处理 %d 个文件，成功删除 %d 个", totalProcessed, totalDeleted)
	task.AppendLog(ctx, "%s", msg)
	return &task.TaskResult{Message: msg}, nil
}
