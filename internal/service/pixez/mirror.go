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

	uploadapp "github.com/Rain-kl/Wavelet/internal/apps/upload"
	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/repository"
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
	if existing, err := GetMirrorIllust(ctx, illustID); err == nil && illustComplete(ctx, existing) {
		return existing, nil
	}
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
			if existing.LeaseExpiresAt != nil && existing.LeaseExpiresAt.After(now) {
				return nil
			}
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
			return tx.Model(&model.PixezMirrorIllust{}).Where("illust_id = ? AND (lease_expires_at IS NULL OR lease_expires_at < ?)", illustID, now).Updates(updates).Error
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
	if existing, err := GetMirrorNovel(ctx, novelID); err == nil && novelComplete(existing) {
		return existing, nil
	}
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
			if existing.LeaseExpiresAt != nil && existing.LeaseExpiresAt.After(now) {
				return nil
			}
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
			return tx.Model(&model.PixezMirrorNovel{}).Where("novel_id = ? AND (lease_expires_at IS NULL OR lease_expires_at < ?)", novelID, now).Updates(updates).Error
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

// ProcessMirrorIllust preserves saved pages and only downloads missing content.
func ProcessMirrorIllust(ctx context.Context, client *Client, taskID string, illustID int64) error {
	if client == nil {
		client = DefaultClient
	}
	if err := waitMirrorConcurrencyLimit(ctx, &model.PixezMirrorIllust{}, "illust_id", illustID, model.ConfigKeyPixezMirrorIllustConcurrency); err != nil {
		return err
	}
	return runProtectedMirror(ctx, model.PixezMirrorTargetIllust, illustID, taskID, func(leaseCtx context.Context) error {
		return fillMirrorIllust(leaseCtx, client, illustID)
	})
}

func fillMirrorIllust(ctx context.Context, client *Client, id int64) error {
	row, detail, err := loadMirrorIllustSnapshot(ctx, client, id)
	if err != nil {
		return err
	}
	urls := CollectIllustImageURLs(detail)
	var oldFiles []model.PixezMirrorImageFile
	if row.ImageFilesJSON != "" {
		if err = json.Unmarshal([]byte(row.ImageFilesJSON), &oldFiles); err != nil {
			return fmt.Errorf("decode existing image mappings: %w", err)
		}
	}
	files := make([]model.PixezMirrorImageFile, 0, len(urls))
	failed := make([]string, 0)
	interval, e := repository.GetIntByKey(ctx, model.ConfigKeyPixezMirrorDownloadInterval)
	if e != nil {
		interval = 1
	}
	for page, u := range urls {
		if saved, ok := savedMirrorPage(ctx, oldFiles, page, u); ok {
			files = append(files, saved)
			continue
		}
		if page > 0 && interval > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(interval) * time.Second):
			}
		}
		data, mimeType, downloadErr := client.DownloadFile(ctx, u)
		if downloadErr != nil {
			failed = append(failed, u)
			continue
		}
		f, ingestErr := registerMirrorUpload(ctx, u, page, data, mimeType)
		if ingestErr != nil {
			failed = append(failed, u)
			continue
		}
		files = append(files, f)
		// Persist each successful page so cancellation never loses references.
		merged := mergeMirrorFiles(oldFiles, files)
		if err = updateMirrorIllust(ctx, id, map[string]any{"image_files_json": mustJSON(merged), "success_count": len(merged)}); err != nil {
			return err
		}
	}
	status := model.PixezMirrorStatusSuccess
	message := ""
	if len(failed) > 0 || (detail.Illust.PageCount > 0 && len(files) < detail.Illust.PageCount) {
		status = model.PixezMirrorStatusFailed
		message = "illustration backup is incomplete"
	}
	if err = updateMirrorIllust(ctx, id, map[string]any{"status": status, "image_files_json": mustJSON(mergeMirrorFiles(oldFiles, files)), "request_urls_json": mustJSON(urls), "retry_urls_json": mustJSON(failed), "total_count": max(len(urls), detail.Illust.PageCount), "success_count": len(files), "failed_count": max(len(failed), detail.Illust.PageCount-len(files)), "error_message": message, "updated_at": time.Now()}); err != nil {
		return err
	}
	if status == model.PixezMirrorStatusFailed {
		updateBookmarkMirrorStatus(ctx, model.PixezMirrorTargetIllust, id, model.PixezBookmarkMirrorFailed)
		return errors.New(message)
	}
	updateBookmarkMirrorStatus(ctx, model.PixezMirrorTargetIllust, id, model.PixezBookmarkMirrorDone)
	task.AppendLog(ctx, "插画备份完成 id=%d pages=%d", id, len(files))
	return nil
}

func mergeMirrorFiles(oldFiles, newFiles []model.PixezMirrorImageFile) []model.PixezMirrorImageFile {
	merged := append([]model.PixezMirrorImageFile{}, oldFiles...)
	for _, f := range newFiles {
		found := false
		for i, old := range merged {
			if old.Page == f.Page {
				merged[i] = f
				found = true
				break
			}
		}
		if !found {
			merged = append(merged, f)
		}
	}
	return merged
}

// ProcessMirrorNovel fills missing snapshots without replacing archived text.
func ProcessMirrorNovel(ctx context.Context, client *Client, taskID string, novelID int64) error {
	if client == nil {
		client = DefaultClient
	}
	if err := waitMirrorConcurrencyLimit(ctx, &model.PixezMirrorNovel{}, "novel_id", novelID, model.ConfigKeyPixezMirrorNovelConcurrency); err != nil {
		return err
	}
	return runProtectedMirror(ctx, model.PixezMirrorTargetNovel, novelID, taskID, func(leaseCtx context.Context) error { return fillMirrorNovel(leaseCtx, client, novelID) })
}

func fillMirrorNovel(ctx context.Context, client *Client, id int64) error {
	row, err := GetMirrorNovel(ctx, id)
	if err != nil {
		return err
	}
	var detail NovelDetail
	validDetail := json.Unmarshal([]byte(row.DetailJSON), &detail) == nil && detail.Novel.ID == id
	var content NovelWebContent
	validText := json.Unmarshal([]byte(row.TextJSON), &content) == nil && content.Text != ""
	var user model.PixezPixivUser
	if !validDetail || !validText {
		user, err = latestMirrorUser(ctx)
		if err != nil {
			return err
		}
	}
	if !validDetail {
		raw, d, e := client.GetNovelDetail(ctx, user, id)
		if e != nil {
			return e
		}
		if d.Novel.ID != id || strings.Contains(string(raw), limitUnknownNovel) {
			return errors.New("pixiv novel unavailable")
		}
		detail = d
		if err = updateMirrorNovel(ctx, id, map[string]any{"detail_json": string(raw)}); err != nil {
			return err
		}
	}
	if err = IndexNovel(ctx, detail.Novel, true, ""); err != nil {
		return err
	}
	if !validText {
		raw, text, e := client.GetNovelText(ctx, user, id)
		if e != nil {
			return e
		}
		if text.Text == "" {
			return errors.New("pixiv novel returned empty text")
		}
		if err = updateMirrorNovel(ctx, id, map[string]any{"text_json": string(raw)}); err != nil {
			return err
		}
	}
	if err = updateMirrorNovel(ctx, id, map[string]any{"status": model.PixezMirrorStatusSuccess, "total_count": 1, "success_count": 1, "failed_count": 0, "retry_urls_json": "[]", "error_message": "", "updated_at": time.Now()}); err != nil {
		return err
	}
	updateBookmarkMirrorStatus(ctx, model.PixezMirrorTargetNovel, id, model.PixezBookmarkMirrorDone)
	task.AppendLog(ctx, "小说备份完成 id=%d", id)
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
	return protectedMirrorUpdate(ctx, model.PixezMirrorTargetIllust, illustID, updates)
}

func updateMirrorNovel(ctx context.Context, novelID int64, updates map[string]any) error {
	return protectedMirrorUpdate(ctx, model.PixezMirrorTargetNovel, novelID, updates)
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

	ext := strings.TrimPrefix(strings.ToLower(path.Ext(fileName)), ".")
	if ext == "" {
		ext = extensionFromMime(mimeType)
	}

	accessMode := 1
	ingestResult, err := uploadapp.Ingest(ctx, uploadapp.IngestRequest{
		UserID:     firstUploadOwnerID(ctx),
		Reader:     bytes.NewReader(data),
		Size:       size,
		FileName:   fileName,
		MimeType:   mimeType,
		Extension:  ext,
		Hash:       hash,
		Type:       pixezMirrorUploadType,
		AccessMode: &accessMode,
		Status:     model.UploadStatusUsed,
		Metadata: model.UploadMetadata{
			OriginalMime: mimeType,
			Extra: map[string]any{
				"pixez_source_url": pixivURL,
				"pixez_page":       pageIndex,
			},
		},
		Policy:             uploadapp.PolicyResolveExisting,
		SkipExtensionCheck: true,
	})
	if err != nil {
		return model.PixezMirrorImageFile{}, fmt.Errorf("ingest mirrored Pixiv image: %w", err)
	}
	return imageFileRecord(pixivURL, pageIndex, ingestResult.Upload), nil
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

func loadMirrorIllustSnapshot(ctx context.Context, client *Client, id int64) (model.PixezMirrorIllust, IllustDetail, error) {
	row, err := GetMirrorIllust(ctx, id)
	if err != nil {
		return row, IllustDetail{}, err
	}
	var detail IllustDetail
	valid := json.Unmarshal([]byte(row.DetailJSON), &detail) == nil && detail.Illust.ID == id && len(CollectIllustImageURLs(detail)) > 0 && !IsLimitUnknownIllust(detail.Illust)
	if !valid {
		user, e := latestMirrorUser(ctx)
		if e != nil {
			return row, detail, e
		}
		raw, fetched, e := client.GetIllustDetail(ctx, user, id)
		if e != nil {
			return row, detail, e
		}
		if fetched.Illust.ID != id || IsLimitUnknownIllust(fetched.Illust) || len(CollectIllustImageURLs(fetched)) == 0 {
			return row, detail, errors.New("pixiv illustration unavailable")
		}
		detail = fetched
		if e = updateMirrorIllust(ctx, id, map[string]any{"detail_json": string(raw)}); e != nil {
			return row, detail, e
		}
	}
	err = IndexIllust(ctx, detail.Illust, true, "")
	return row, detail, err
}
func savedMirrorPage(ctx context.Context, files []model.PixezMirrorImageFile, page int, u string) (model.PixezMirrorImageFile, bool) {
	for _, f := range files {
		if f.Page == page && f.PixivURL == u && validMirrorFile(ctx, f) {
			return f, true
		}
	}
	return model.PixezMirrorImageFile{}, false
}
