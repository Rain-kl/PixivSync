// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package upload

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/storage"
	"github.com/Rain-kl/Wavelet/internal/task"
	"gorm.io/gorm"
)

// 异步任务名称与管理类型定义
const (
	// CleanupUnusedUploadsTask 清理未使用上传任务标识
	CleanupUnusedUploadsTask = "upload:cleanup_unused"
	// TaskTypeCleanupUploads 清理未使用上传管理类型
	TaskTypeCleanupUploads = "cleanup_unused_uploads"
	// WarmImageCacheTask 图片压缩缓存预热任务标识
	WarmImageCacheTask = "upload:warm_image_cache"
	// TaskTypeWarmImageCache 图片压缩缓存预热管理类型
	TaskTypeWarmImageCache = "warm_image_cache"
)

var warmImageCacheMu sync.Mutex

// CleanupUnusedUploadsMeta represents the task metadata.
var CleanupUnusedUploadsMeta = task.TaskMeta{
	Type:         TaskTypeCleanupUploads,
	AsynqTask:    CleanupUnusedUploadsTask,
	Name:         "清理未使用上传",
	Description:  "清理超过1小时未使用的上传文件",
	SupportsTime: false,
	MaxRetry:     task.DefaultMaxRetry,
	Queue:        task.QueueDefault,
	Retryable:    true,
}

// WarmImageCacheMeta represents the image cache warmup task metadata.
var WarmImageCacheMeta = task.TaskMeta{
	Type:         TaskTypeWarmImageCache,
	AsynqTask:    WarmImageCacheTask,
	Name:         "预热图片压缩缓存",
	Description:  "串行将文件管理中的图片转换为指定质量的 WebP 并写入永久缓存",
	SupportsTime: false,
	MaxRetry:     task.DefaultMaxRetry,
	Queue:        task.QueueDefault,
	Retryable:    true,
	Params: []task.TaskParam{
		{
			Name:        "quality",
			Label:       "图片质量",
			Type:        "string",
			Required:    true,
			Placeholder: "low / medium / high",
			Description: "WebP 压缩质量，仅支持 low、medium、high",
		},
	},
}

// WarmImageCachePayload is the image cache warmup task payload.
type WarmImageCachePayload struct {
	Quality string `json:"quality"`
}

// CleanupUnusedUploadsHandler 清理未使用上传文件的异步任务处理器
type CleanupUnusedUploadsHandler struct{}

// WarmImageCacheHandler serially warms compressed image cache entries.
type WarmImageCacheHandler struct{}

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

// ValidatePayload validates and normalizes image cache warmup parameters.
func (h *WarmImageCacheHandler) ValidatePayload(payload []byte) ([]byte, error) {
	if len(payload) == 0 {
		return nil, errors.New(errImageCacheWarmupPayloadRequired)
	}

	var req WarmImageCachePayload
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf(errInvalidImageCacheWarmupPayload, err)
	}

	req.Quality = strings.ToLower(strings.TrimSpace(req.Quality))
	if req.Quality != imageQualityLow &&
		req.Quality != imageQualityMedium &&
		req.Quality != imageQualityHigh {
		return nil, errors.New(errInvalidImageCacheWarmupQuality)
	}

	return json.Marshal(req)
}

// Execute serially converts all managed images to WebP cache entries.
func (h *WarmImageCacheHandler) Execute(ctx context.Context, payload []byte) (*task.TaskResult, error) {
	normalizedPayload, err := h.ValidatePayload(payload)
	if err != nil {
		task.AppendLog(ctx, "图片缓存预热参数无效: %v", err)
		return nil, err
	}

	var req WarmImageCachePayload
	if err := json.Unmarshal(normalizedPayload, &req); err != nil {
		return nil, fmt.Errorf(errParseImageCacheWarmupPayload, err)
	}

	task.AppendLog(ctx, "等待获取图片缓存预热执行锁，质量: %s", req.Quality)
	warmImageCacheMu.Lock()
	defer warmImageCacheMu.Unlock()

	const (
		batchSize      = 50
		maxFailureLogs = 5
	)
	var lastID uint64
	var totalProcessed int
	var totalCached int
	var totalGenerated int
	var totalFailed int

	task.AppendLog(ctx, "开始串行预热图片压缩缓存，质量: %s，每批: %d", req.Quality, batchSize)

	for {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("image cache warmup canceled: %w", err)
		}

		var uploads []model.Upload
		if err := db.DB(ctx).
			Where("id > ? AND status != ? AND (LOWER(mime_type) LIKE ? OR LOWER(extension) IN ?)",
				lastID,
				model.UploadStatusDeleted,
				"image/%",
				[]string{"jpg", "jpeg", "png", "webp", "gif"},
			).
			Order("id ASC").
			Limit(batchSize).
			Find(&uploads).Error; err != nil {
			task.AppendLog(ctx, "查询图片上传记录失败: %v", err)
			return nil, fmt.Errorf(errQueryImagesForCacheWarmup, err)
		}

		if len(uploads) == 0 {
			break
		}

		batchGenerated := 0
		batchCached := 0
		batchFailed := 0
		for i := range uploads {
			if err := ctx.Err(); err != nil {
				return nil, fmt.Errorf("image cache warmup canceled: %w", err)
			}

			upload := &uploads[i]
			totalProcessed++
			lastID = upload.ID

			_, cacheHit, err := ensureCompressedImageCache(ctx, upload, req.Quality)
			if err != nil {
				totalFailed++
				batchFailed++
				if totalFailed <= maxFailureLogs {
					task.AppendLog(ctx, "图片处理失败 [ID:%d]: %v", upload.ID, err)
				}
				continue
			}
			if cacheHit {
				totalCached++
				batchCached++
				continue
			}
			totalGenerated++
			batchGenerated++
		}

		task.AppendLog(
			ctx,
			"批次完成，末尾 ID: %d，生成: %d，命中: %d，失败: %d",
			lastID,
			batchGenerated,
			batchCached,
			batchFailed,
		)
	}

	msg := fmt.Sprintf(
		"图片缓存预热完成，共处理 %d 张，生成 %d 张，命中 %d 张，失败 %d 张",
		totalProcessed,
		totalGenerated,
		totalCached,
		totalFailed,
	)
	task.AppendLog(ctx, "%s", msg)
	return &task.TaskResult{Message: msg}, nil
}
