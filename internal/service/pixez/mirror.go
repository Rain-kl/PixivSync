// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package pixez

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/db/idgen"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/repository"
	"github.com/Rain-kl/Wavelet/internal/storage"
	"github.com/Rain-kl/Wavelet/pkg/logger"
	"github.com/Rain-kl/Wavelet/internal/task"
	"gorm.io/gorm"
)

const (
	pixezMirrorUploadType = "pixez_mirror"
	localUploadDirPerm    = 0755
	localUploadFilePerm   = 0644
)

// MirrorStatus is the client-facing PixEz mirror status DTO.
type MirrorStatus struct {
	TaskID          string `json:"task_id"`
	IllustID        int64  `json:"illust_id,omitempty"`
	NovelID         int64  `json:"novel_id,omitempty"`
	Status          string `json:"status"`
	Mirrored        bool   `json:"mirrored"`
	TotalCount      int    `json:"total_count"`
	SuccessCount    int    `json:"success_count"`
	FailedCount     int    `json:"failed_count"`
	RequestURLsJSON string `json:"request_urls_json,omitempty"`
	RetryURLsJSON   string `json:"retry_urls_json,omitempty"`
	ErrorMessage    string `json:"error_message,omitempty"`
}

// EnsureMirrorIllustQueued creates or updates the read-model row for an illust mirror task.
func EnsureMirrorIllustQueued(ctx context.Context, illustID int64, taskID string) (model.PixezMirrorIllust, error) {
	now := time.Now()
	record := model.PixezMirrorIllust{
		IllustID:        illustID,
		TaskID:          taskID,
		Status:          model.PixezMirrorStatusQueued,
		ImageFilesJSON:  "[]",
		RequestURLsJSON: "[]",
		RetryURLsJSON:   "[]",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.PixezMirrorIllust
		err := tx.Where("illust_id = ?", illustID).First(&existing).Error
		if err == nil {
			updates := map[string]any{
				keyTaskID:       taskID,
				keyStatus:       model.PixezMirrorStatusQueued,
				keyErrorMessage: "",
				keyUpdatedAt:    now,
			}
			if existing.ImageFilesJSON == "" {
				updates["image_files_json"] = "[]"
			}
			if existing.RequestURLsJSON == "" {
				updates["request_urls_json"] = "[]"
			}
			if existing.RetryURLsJSON == "" {
				updates["retry_urls_json"] = "[]"
			}
			return tx.Model(&model.PixezMirrorIllust{}).Where("illust_id = ?", illustID).Updates(updates).Error
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return tx.Create(&record).Error
	})
	if err != nil {
		return model.PixezMirrorIllust{}, err
	}

	return GetMirrorIllust(ctx, illustID)
}

// EnsureMirrorNovelQueued creates or updates the read-model row for a novel mirror task.
func EnsureMirrorNovelQueued(ctx context.Context, novelID int64, taskID string) (model.PixezMirrorNovel, error) {
	now := time.Now()
	record := model.PixezMirrorNovel{
		NovelID:         novelID,
		TaskID:          taskID,
		Status:          model.PixezMirrorStatusQueued,
		RequestURLsJSON: "[]",
		RetryURLsJSON:   "[]",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.PixezMirrorNovel
		err := tx.Where("novel_id = ?", novelID).First(&existing).Error
		if err == nil {
			updates := map[string]any{
				keyTaskID:       taskID,
				keyStatus:       model.PixezMirrorStatusQueued,
				keyErrorMessage: "",
				keyUpdatedAt:    now,
			}
			if existing.RequestURLsJSON == "" {
				updates["request_urls_json"] = "[]"
			}
			if existing.RetryURLsJSON == "" {
				updates["retry_urls_json"] = "[]"
			}
			return tx.Model(&model.PixezMirrorNovel{}).Where("novel_id = ?", novelID).Updates(updates).Error
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return tx.Create(&record).Error
	})
	if err != nil {
		return model.PixezMirrorNovel{}, err
	}

	return GetMirrorNovel(ctx, novelID)
}

// GetMirrorIllust returns the read-model row for one illustration.
func GetMirrorIllust(ctx context.Context, illustID int64) (model.PixezMirrorIllust, error) {
	var record model.PixezMirrorIllust
	err := db.DB(ctx).Where("illust_id = ?", illustID).First(&record).Error
	return record, err
}

// GetMirrorNovel returns the read-model row for one novel.
func GetMirrorNovel(ctx context.Context, novelID int64) (model.PixezMirrorNovel, error) {
	var record model.PixezMirrorNovel
	err := db.DB(ctx).Where("novel_id = ?", novelID).First(&record).Error
	return record, err
}

// MirrorIllustStatus builds the client-facing illustration status.
func MirrorIllustStatus(record model.PixezMirrorIllust) MirrorStatus {
	return MirrorStatus{
		TaskID:          record.TaskID,
		IllustID:        record.IllustID,
		Status:          record.Status,
		Mirrored:        record.SuccessCount > 0,
		TotalCount:      record.TotalCount,
		SuccessCount:    record.SuccessCount,
		FailedCount:     record.FailedCount,
		RequestURLsJSON: record.RequestURLsJSON,
		RetryURLsJSON:   record.RetryURLsJSON,
		ErrorMessage:    record.ErrorMessage,
	}
}

// MirrorNovelStatus builds the client-facing novel status.
func MirrorNovelStatus(record model.PixezMirrorNovel) MirrorStatus {
	return MirrorStatus{
		TaskID:          record.TaskID,
		NovelID:         record.NovelID,
		Status:          record.Status,
		Mirrored:        record.SuccessCount > 0,
		TotalCount:      record.TotalCount,
		SuccessCount:    record.SuccessCount,
		FailedCount:     record.FailedCount,
		RequestURLsJSON: record.RequestURLsJSON,
		RetryURLsJSON:   record.RetryURLsJSON,
		ErrorMessage:    record.ErrorMessage,
	}
}

// ProcessMirrorIllust executes one illustration mirror task.
func ProcessMirrorIllust(ctx context.Context, client *Client, taskID string, illustID int64) error {
	if client == nil {
		client = DefaultClient
	}

	// 限制并发执行插画下载任务数
	if err := waitMirrorConcurrencyLimit(ctx, &model.PixezMirrorIllust{}, "illust_id", illustID, model.ConfigKeyPixezMirrorIllustConcurrency); err != nil {
		return err
	}

	if err := updateMirrorIllust(ctx, illustID, map[string]any{
		keyTaskID:       taskID,
		keyStatus:       model.PixezMirrorStatusProcessing,
		keyErrorMessage: "",
		keyUpdatedAt:    time.Now(),
	}); err != nil {
		return fmt.Errorf("mark illust mirror processing: %w", err)
	}

	user, err := latestMirrorUser(ctx)
	if err != nil {
		task.AppendLog(ctx, "获取可用的 Pixiv Token 失败: %v", err)
		markIllustFailed(ctx, illustID, taskID, err)
		return err
	}

	task.AppendLog(ctx, "正在向 Pixiv 请求插画详情 [illust_id: %d]...", illustID)
	detailBytes, detail, err := client.GetIllustDetail(ctx, user, illustID)
	if err != nil {
		wrapped := fmt.Errorf("fetch Pixiv illust detail illust_id=%d: %w", illustID, err)
		task.AppendLog(ctx, "获取插画详情失败: %v", err)
		markIllustFailed(ctx, illustID, taskID, wrapped)
		return wrapped
	}

	imageURLs := CollectIllustImageURLs(detail)
	task.AppendLog(ctx, "成功获取插画详情: 「%s」 (画师: %s, 包含 %d 张图片)，开始下载...", detail.Illust.Title, detail.Illust.User.Name, len(imageURLs))
	requestURLsJSON := mustJSON(imageURLs)
	files := make([]model.PixezMirrorImageFile, 0, len(imageURLs))
	failedURLs := make([]string, 0)

	// 获取多图下载间隔
	downloadInterval, err := repository.GetIntByKey(ctx, model.ConfigKeyPixezMirrorDownloadInterval)
	if err != nil {
		downloadInterval = 1 // 默认 1 秒
	}

	for idx, imageURL := range imageURLs {
		if idx > 0 && downloadInterval > 0 {
			task.AppendLog(ctx, "等待 %d 秒以满足多图下载间隔限制...", downloadInterval)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(downloadInterval) * time.Second):
			}
		}

		task.AppendLog(ctx, "正在下载第 %d/%d 张图片...", idx+1, len(imageURLs))
		data, mimeType, err := client.DownloadFile(ctx, imageURL)
		if err != nil {
			task.AppendLog(ctx, "第 %d/%d 张图片下载失败: %v", idx+1, len(imageURLs), err)
			failedURLs = append(failedURLs, imageURL)
			continue
		}
		task.AppendLog(ctx, "第 %d/%d 张图片下载成功，大小: %d 字节, 正在保存到存储驱动...", idx+1, len(imageURLs), len(data))
		fileRecord, err := registerMirrorUpload(ctx, imageURL, idx, data, mimeType)
		if err != nil {
			task.AppendLog(ctx, "第 %d/%d 张图片保存失败: %v", idx+1, len(imageURLs), err)
			failedURLs = append(failedURLs, imageURL)
			continue
		}
		task.AppendLog(ctx, "第 %d/%d 张图片保存并登记成功 [存储路径: %s]", idx+1, len(imageURLs), fileRecord.StorageKey)
		files = append(files, fileRecord)
	}

	status := model.PixezMirrorStatusSuccess
	errMessage := ""
	if len(files) == 0 {
		status = model.PixezMirrorStatusFailed
		errMessage = fmt.Sprintf("failed to mirror all %d Pixiv image files", len(imageURLs))
	}

	updates := map[string]any{
		keyTaskID:           taskID,
		keyStatus:           status,
		"detail_json":       string(detailBytes),
		"image_files_json":  mustJSON(files),
		"request_urls_json": requestURLsJSON,
		"retry_urls_json":   mustJSON(failedURLs),
		keyErrorMessage:     errMessage,
		keyTotalCount:       len(imageURLs),
		"success_count":     len(files),
		"failed_count":      len(failedURLs),
		keyUpdatedAt:        time.Now(),
	}
	if err := updateMirrorIllust(ctx, illustID, updates); err != nil {
		return fmt.Errorf("save illust mirror result: %w", err)
	}
	if status == model.PixezMirrorStatusFailed {
		updateBookmarkMirrorStatus(ctx, model.PixezMirrorTargetIllust, illustID, model.PixezBookmarkMirrorFailed)
		return errors.New(errMessage)
	}
	task.AppendLog(ctx, "插画镜像同步成功，共保存 %d/%d 张图片", len(files), len(imageURLs))
	updateBookmarkMirrorStatus(ctx, model.PixezMirrorTargetIllust, illustID, model.PixezBookmarkMirrorDone)
	return nil
}

// ProcessMirrorNovel executes one novel mirror task.
func ProcessMirrorNovel(ctx context.Context, client *Client, taskID string, novelID int64) error {
	if client == nil {
		client = DefaultClient
	}

	// 限制并发执行小说下载任务数
	if err := waitMirrorConcurrencyLimit(ctx, &model.PixezMirrorNovel{}, "novel_id", novelID, model.ConfigKeyPixezMirrorNovelConcurrency); err != nil {
		return err
	}

	if err := updateMirrorNovel(ctx, novelID, map[string]any{
		keyTaskID:       taskID,
		keyStatus:       model.PixezMirrorStatusProcessing,
		keyErrorMessage: "",
		keyUpdatedAt:    time.Now(),
	}); err != nil {
		return fmt.Errorf("mark novel mirror processing: %w", err)
	}

	user, err := latestMirrorUser(ctx)
	if err != nil {
		task.AppendLog(ctx, "获取可用的 Pixiv Token 失败: %v", err)
		markNovelFailed(ctx, novelID, taskID, err)
		return err
	}

	task.AppendLog(ctx, "正在向 Pixiv 请求小说详情 [novel_id: %d]...", novelID)
	detailBytes, detail, err := client.GetNovelDetail(ctx, user, novelID)
	if err != nil {
		wrapped := fmt.Errorf("fetch Pixiv novel detail novel_id=%d: %w", novelID, err)
		task.AppendLog(ctx, "获取小说详情失败: %v", err)
		markNovelFailed(ctx, novelID, taskID, wrapped)
		return wrapped
	}
	task.AppendLog(ctx, "成功获取小说详情: 「%s」 (字数: %d)，正在向 Pixiv 请求小说正文...", detail.Novel.Title, detail.Novel.TextLength)

	textBytes, _, err := client.GetNovelText(ctx, user, novelID)
	if err != nil {
		wrapped := fmt.Errorf("fetch Pixiv novel text novel_id=%d: %w", novelID, err)
		task.AppendLog(ctx, "获取小说正文失败: %v", err)
		markNovelFailed(ctx, novelID, taskID, wrapped)
		return wrapped
	}
	task.AppendLog(ctx, "成功获取小说正文，正在保存至数据库中...")

	updates := map[string]any{
		keyTaskID:     taskID,
		keyStatus:     model.PixezMirrorStatusSuccess,
		"detail_json": string(detailBytes),
		"text_json":   string(textBytes),
		"request_urls_json": mustJSON([]string{
			fmt.Sprintf("https://%s/v2/novel/detail?novel_id=%d", pixivAPIHost, novelID),
			fmt.Sprintf("https://%s/webview/v2/novel?id=%d", pixivAPIHost, novelID),
		}),
		"retry_urls_json": "[]",
		keyErrorMessage:   "",
		keyTotalCount:     1,
		"success_count":   1,
		"failed_count":    0,
		keyUpdatedAt:      time.Now(),
	}
	if err := updateMirrorNovel(ctx, novelID, updates); err != nil {
		return fmt.Errorf("save novel mirror result: %w", err)
	}
	task.AppendLog(ctx, "小说镜像及正文已成功保存")
	updateBookmarkMirrorStatus(ctx, model.PixezMirrorTargetNovel, novelID, model.PixezBookmarkMirrorDone)
	return nil
}

func latestMirrorUser(ctx context.Context) (model.PixezPixivUser, error) {
	var user model.PixezPixivUser
	if err := db.DB(ctx).Order("updated_at desc").First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return user, errors.New("no Pixiv user token available; sync an account before mirroring")
		}
		return user, fmt.Errorf("query Pixiv user token failed: %w", err)
	}
	return user, nil
}

func updateMirrorIllust(ctx context.Context, illustID int64, updates map[string]any) error {
	return db.DB(ctx).Model(&model.PixezMirrorIllust{}).Where("illust_id = ?", illustID).Updates(updates).Error
}

func updateMirrorNovel(ctx context.Context, novelID int64, updates map[string]any) error {
	return db.DB(ctx).Model(&model.PixezMirrorNovel{}).Where("novel_id = ?", novelID).Updates(updates).Error
}

func markIllustFailed(ctx context.Context, illustID int64, taskID string, err error) {
	_ = updateMirrorIllust(ctx, illustID, map[string]any{
		keyTaskID:       taskID,
		keyStatus:       model.PixezMirrorStatusFailed,
		keyErrorMessage: err.Error(),
		keyUpdatedAt:    time.Now(),
	})
	updateBookmarkMirrorStatus(ctx, model.PixezMirrorTargetIllust, illustID, model.PixezBookmarkMirrorFailed)
}

func markNovelFailed(ctx context.Context, novelID int64, taskID string, err error) {
	_ = updateMirrorNovel(ctx, novelID, map[string]any{
		keyTaskID:       taskID,
		keyStatus:       model.PixezMirrorStatusFailed,
		keyErrorMessage: err.Error(),
		keyUpdatedAt:    time.Now(),
	})
	updateBookmarkMirrorStatus(ctx, model.PixezMirrorTargetNovel, novelID, model.PixezBookmarkMirrorFailed)
}

func updateBookmarkMirrorStatus(ctx context.Context, targetType string, targetID int64, status int) {
	now := time.Now()
	switch targetType {
	case model.PixezMirrorTargetIllust:
		_ = db.DB(ctx).Model(&model.PixezBookmarkIllust{}).
			Where("illust_id = ?", targetID).
			Updates(map[string]any{"mirror_status": status, keyUpdatedAt: now}).Error
	case model.PixezMirrorTargetNovel:
		_ = db.DB(ctx).Model(&model.PixezBookmarkNovel{}).
			Where("novel_id = ?", targetID).
			Updates(map[string]any{"mirror_status": status, keyUpdatedAt: now}).Error
	}
}

func registerMirrorUpload(ctx context.Context, pixivURL string, pageIndex int, data []byte, mimeType string) (model.PixezMirrorImageFile, error) {
	size := int64(len(data))
	hashBytes := sha256.Sum256(data)
	hash := hex.EncodeToString(hashBytes[:])
	fileName := fileNameFromURL(pixivURL)
	if fileName == "" {
		return model.PixezMirrorImageFile{}, fmt.Errorf("invalid Pixiv image URL filename: %s", pixivURL)
	}
	if mimeType == "" {
		mimeType = http.DetectContentType(data[:min(len(data), detectContentBytes)])
	}

	var existing model.Upload
	if err := db.DB(ctx).
		Where("hash = ? AND file_size = ? AND status IN (?, ?)", hash, size, model.UploadStatusPending, model.UploadStatusUsed).
		First(&existing).Error; err == nil {
		return imageFileRecord(pixivURL, pageIndex, existing), nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.PixezMirrorImageFile{}, err
	}

	ext := strings.TrimPrefix(strings.ToLower(path.Ext(fileName)), ".")
	if ext == "" {
		ext = extensionFromMime(mimeType)
	}
	id := idgen.NextUint64ID()
	subPath := fmt.Sprintf("uploads/%s/%d.%s", time.Now().Format("2006/01/02"), id, ext)
	_, backend, err := storage.Active(ctx)
	if err != nil {
		return model.PixezMirrorImageFile{}, fmt.Errorf("active storage configuration: %w", err)
	}

	result, err := backend.Put(ctx, subPath, bytes.NewReader(data), size, mimeType)
	if err != nil {
		return model.PixezMirrorImageFile{}, fmt.Errorf("store mirrored Pixiv image: %w", err)
	}

	upload := model.Upload{
		ID:            id,
		UserID:        firstUploadOwnerID(ctx),
		FileName:      fileName,
		FilePath:      result.Key,
		FileSize:      size,
		MimeType:      mimeType,
		Extension:     ext,
		Hash:          hash,
		Type:          pixezMirrorUploadType,
		Status:        model.UploadStatusUsed,
		AccessMode:    1,
		Metadata: model.UploadMetadata{
			OriginalMime: mimeType,
			Bucket:       result.Bucket,
			Extra: map[string]any{
				"pixez_source_url": pixivURL,
				"pixez_page":       pageIndex,
			},
		},
	}
	if err := db.DB(ctx).Create(&upload).Error; err != nil {
		if deleteErr := backend.Delete(ctx, result.Key); deleteErr != nil {
			logger.WarnF(ctx, "failed to delete mirrored file on database error: %v", deleteErr)
		}
		return model.PixezMirrorImageFile{}, fmt.Errorf("create upload record: %w", err)
	}
	return imageFileRecord(pixivURL, pageIndex, upload), nil
}

func firstUploadOwnerID(ctx context.Context) uint64 {
	var user model.User
	if err := db.DB(ctx).Order("id asc").First(&user).Error; err != nil {
		return 0
	}
	return user.ID
}

func imageFileRecord(pixivURL string, pageIndex int, upload model.Upload) model.PixezMirrorImageFile {
	return model.PixezMirrorImageFile{
		PixivURL:   pixivURL,
		Page:       pageIndex,
		UploadID:   upload.ID,
		FileName:   upload.FileName,
		Hash:       upload.Hash,
		Mime:       upload.MimeType,
		Size:       upload.FileSize,
		StorageKey: upload.FilePath,
	}
}

func extensionFromMime(mimeType string) string {
	extensions, err := mime.ExtensionsByType(mimeType)
	if err == nil && len(extensions) > 0 {
		return strings.TrimPrefix(extensions[0], ".")
	}
	return "bin"
}

func fileNameFromURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return path.Base(parsed.Path)
}

// FindMirroredImageUpload resolves a /mirror/pximg path to an Upload record.
func FindMirroredImageUpload(ctx context.Context, pximgPath string) (model.Upload, error) {
	cleanPath := strings.TrimPrefix(pximgPath, "/")
	if cleanPath == "" || strings.Contains(cleanPath, "..") {
		return model.Upload{}, fmt.Errorf("invalid image path")
	}

	requestedName := path.Base(cleanPath)
	illustID, ok := leadingNumericID(requestedName)
	if !ok {
		return model.Upload{}, fmt.Errorf("cannot determine illust ID from filename")
	}

	var record model.PixezMirrorIllust
	if err := db.DB(ctx).Where("illust_id = ?", illustID).First(&record).Error; err != nil {
		return model.Upload{}, err
	}

	originalName := originalImageFilename(requestedName)
	stripExt := func(filename string) string {
		return strings.TrimSuffix(filename, path.Ext(filename))
	}
	originalBase := stripExt(originalName)
	requestedBase := stripExt(requestedName)

	var files []model.PixezMirrorImageFile
	if err := json.Unmarshal([]byte(record.ImageFilesJSON), &files); err != nil {
		return model.Upload{}, fmt.Errorf("parse image_files_json: %w", err)
	}
	for _, file := range files {
		fileBase := stripExt(file.FileName)
		pixivBase := stripExt(path.Base(file.PixivURL))
		if file.FileName == requestedName || file.FileName == originalName || path.Base(file.PixivURL) == originalName ||
			fileBase == requestedBase || fileBase == originalBase || pixivBase == originalBase {
			var upload model.Upload
			if err := db.DB(ctx).
				Where("id = ? AND status IN (?, ?)", file.UploadID, model.UploadStatusPending, model.UploadStatusUsed).
				First(&upload).Error; err != nil {
				return model.Upload{}, err
			}
			return upload, nil
		}
	}

	return model.Upload{}, gorm.ErrRecordNotFound
}

func leadingNumericID(filename string) (int64, bool) {
	var b strings.Builder
	for _, r := range filename {
		if r < '0' || r > '9' {
			break
		}
		b.WriteRune(r)
	}
	if b.Len() == 0 {
		return 0, false
	}
	id, err := strconv.ParseInt(b.String(), 10, 64)
	return id, err == nil && id > 0
}

func originalImageFilename(filename string) string {
	ext := path.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	if idx := strings.Index(base, "_master"); idx != -1 {
		return base[:idx] + ext
	}
	if idx := strings.Index(base, "_square"); idx != -1 {
		return base[:idx] + ext
	}
	return filename
}

// RewritePximgURLs rewrites Pixiv image hosts to the current mirror prefix.
func RewritePximgURLs(raw string, prefix string) string {
	escapedPrefix := strings.ReplaceAll(prefix, "/", "\\/")
	dataStr := strings.ReplaceAll(raw, "https://i.pximg.net", prefix)
	dataStr = strings.ReplaceAll(dataStr, "https://s.pximg.net", prefix)
	dataStr = strings.ReplaceAll(dataStr, "https:\\/\\/i.pximg.net", escapedPrefix)
	dataStr = strings.ReplaceAll(dataStr, "https:\\/\\/s.pximg.net", escapedPrefix)
	return dataStr
}

func mustJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func waitMirrorConcurrencyLimit(ctx context.Context, modelObj any, idColumn string, targetID int64, configKey string) error {
	maxConcurrency, err := repository.GetIntByKey(ctx, configKey)
	if err != nil {
		maxConcurrency = 5 // 默认限制为 5
	}

	const checkInterval = 2 * time.Second
	loggedWait := false
	for {
		var activeCount int64
		// 防御性设计：只统计 15 分钟内更新过的活跃任务，防止因进程异常崩溃导致 processing 状态长期泄漏、引发死锁
		err := db.DB(ctx).Model(modelObj).
			Where("status = ? AND "+idColumn+" <> ? AND updated_at > ?", model.PixezMirrorStatusProcessing, targetID, time.Now().Add(-15*time.Minute)).
			Count(&activeCount).Error
		if err != nil {
			return fmt.Errorf("check active mirror count for %s: %w", idColumn, err)
		}
		if int(activeCount) < maxConcurrency {
			if loggedWait {
				task.AppendLog(ctx, "并发通道已释放，开始执行镜像任务")
			}
			return nil
		}
		if !loggedWait {
			task.AppendLog(ctx, "当前活跃并发任务数 (%d) 已达限制 (%d)，任务进入等待通道...", activeCount, maxConcurrency)
			loggedWait = true
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(checkInterval):
		}
	}
}
