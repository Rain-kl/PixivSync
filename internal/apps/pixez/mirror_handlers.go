/*
Copyright 2026 linux.do
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

package pixez

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	pixezsvc "github.com/Rain-kl/Wavelet/internal/service/pixez"
	"github.com/Rain-kl/Wavelet/internal/storage"
	"github.com/Rain-kl/Wavelet/internal/task"
	"github.com/Rain-kl/Wavelet/internal/util"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type batchIllustMirrorRequest struct {
	IllustIDs []int64 `json:"illust_ids"`
}

type batchNovelMirrorRequest struct {
	NovelIDs []int64 `json:"novel_ids"`
}

const maxBatchMirrorIDs = 500

// MirrorIllust dispatches an illustration mirror task.
func MirrorIllust(c *gin.Context) {
	illustID, ok := parsePositiveIDParam(c, "illust_id")
	if !ok {
		return
	}
	record, err := dispatchIllustMirrorIfNeeded(c, illustID)
	if err != nil {
		c.JSON(http.StatusOK, util.Err(errDispatchMirrorTaskFailed))
		return
	}
	c.JSON(http.StatusOK, util.OK(pixezsvc.MirrorIllustStatus(record)))
}

// CheckIllustMirror returns illustration mirror status.
func CheckIllustMirror(c *gin.Context) {
	illustID, ok := parsePositiveIDParam(c, "illust_id")
	if !ok {
		return
	}
	record, err := pixezsvc.GetMirrorIllust(c.Request.Context(), illustID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusOK, util.Err(errQueryMirrorStatusFailed))
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		record = model.PixezMirrorIllust{IllustID: illustID, Status: ""}
	}
	c.JSON(http.StatusOK, util.OK(pixezsvc.MirrorIllustStatus(record)))
}

// BatchCheckIllustMirror returns mirrored illustration IDs.
func BatchCheckIllustMirror(c *gin.Context) {
	var req batchIllustMirrorRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IllustIDs) == 0 {
		c.JSON(http.StatusBadRequest, util.Err(errInvalidRequestBody))
		return
	}
	respondBatchMirror(c, req.IllustIDs, mirroredIllustIDs)
}

// MirrorNovel dispatches a novel mirror task.
func MirrorNovel(c *gin.Context) {
	novelID, ok := parsePositiveIDParam(c, "novel_id")
	if !ok {
		return
	}
	record, err := dispatchNovelMirrorIfNeeded(c, novelID)
	if err != nil {
		c.JSON(http.StatusOK, util.Err(errDispatchMirrorTaskFailed))
		return
	}
	c.JSON(http.StatusOK, util.OK(pixezsvc.MirrorNovelStatus(record)))
}

// CheckNovelMirror returns novel mirror status.
func CheckNovelMirror(c *gin.Context) {
	novelID, ok := parsePositiveIDParam(c, "novel_id")
	if !ok {
		return
	}
	record, err := pixezsvc.GetMirrorNovel(c.Request.Context(), novelID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusOK, util.Err(errQueryMirrorStatusFailed))
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		record = model.PixezMirrorNovel{NovelID: novelID, Status: ""}
	}
	c.JSON(http.StatusOK, util.OK(pixezsvc.MirrorNovelStatus(record)))
}

// BatchCheckNovelMirror returns mirrored novel IDs.
func BatchCheckNovelMirror(c *gin.Context) {
	var req batchNovelMirrorRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.NovelIDs) == 0 {
		c.JSON(http.StatusBadRequest, util.Err(errInvalidRequestBody))
		return
	}
	respondBatchMirror(c, req.NovelIDs, mirroredNovelIDs)
}

func respondBatchMirror(c *gin.Context, requestedIDs []int64, query func(context.Context, []int64) ([]int64, error)) {
	if len(requestedIDs) > maxBatchMirrorIDs {
		c.JSON(http.StatusBadRequest, util.Err(errTooManyMirrorIDs))
		return
	}
	ids, err := query(c.Request.Context(), requestedIDs)
	if err != nil {
		c.JSON(http.StatusOK, util.Err(errQueryMirrorStatusFailed))
		return
	}
	c.JSON(http.StatusOK, util.OK(gin.H{"mirrored_ids": ids}))
}

func mirroredIllustIDs(ctx context.Context, requestedIDs []int64) ([]int64, error) {
	var records []model.PixezMirrorIllust
	if err := db.DB(ctx).
		Where("illust_id IN ? AND success_count > 0", requestedIDs).
		Find(&records).Error; err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.IllustID)
	}
	return ids, nil
}

func mirroredNovelIDs(ctx context.Context, requestedIDs []int64) ([]int64, error) {
	var records []model.PixezMirrorNovel
	if err := db.DB(ctx).
		Where("novel_id IN ? AND success_count > 0", requestedIDs).
		Find(&records).Error; err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.NovelID)
	}
	return ids, nil
}

// GetMirroredIllustDetail returns Pixiv-shape mirrored illustration detail.
func GetMirroredIllustDetail(c *gin.Context) {
	illustID, ok := parsePositiveQueryID(c, "illust_id")
	if !ok {
		return
	}
	record, err := pixezsvc.GetMirrorIllust(c.Request.Context(), illustID)
	if err != nil || record.DetailJSON == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "mirror not found"})
		return
	}
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.String(http.StatusOK, pixezsvc.RewritePximgURLs(record.DetailJSON, mirrorURLPrefix(c)))
}

// ServeMirroredImage streams a mirrored pximg file from Upload storage.
func ServeMirroredImage(c *gin.Context) {
	upload, err := pixezsvc.FindMirroredImageUpload(c.Request.Context(), c.Param("path"))
	if err == nil {
		c.Header("Content-Type", upload.MimeType)
		c.Header("Content-Length", strconv.FormatInt(upload.FileSize, 10))
		if upload.StorageDriver == "local" || (upload.StorageDriver == "" && !storage.IsEnabled()) {
			c.File(upload.FilePath)
			return
		}
		obj, err := storage.GetObjectViaCache(c.Request.Context(), upload.FilePath)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "mirror file not found"})
			return
		}
		if obj.CachePath != "" {
			c.File(obj.CachePath)
			return
		}
		defer func() { _ = obj.Body.Close() }()
		_, _ = io.Copy(c.Writer, obj.Body)
		return
	}

	// Fallback to proxying from Pixiv on the fly
	cleanPath := strings.TrimPrefix(c.Param("path"), "/")
	if cleanPath == "" || strings.Contains(cleanPath, "..") {
		c.JSON(http.StatusNotFound, gin.H{"error": "mirror file not found"})
		return
	}

	pixivURL := "https://i.pximg.net/" + cleanPath
	data, mimeType, err := pixezsvc.DefaultClient.DownloadFile(c.Request.Context(), pixivURL)
	if err != nil {
		// Try s.pximg.net as fallback
		pixivURL = "https://s.pximg.net/" + cleanPath
		data, mimeType, err = pixezsvc.DefaultClient.DownloadFile(c.Request.Context(), pixivURL)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "mirror file not found"})
			return
		}
	}

	c.Header("Content-Type", mimeType)
	c.Header("Content-Length", strconv.Itoa(len(data)))
	_, _ = c.Writer.Write(data)
}

// GetMirroredNovelDetail returns Pixiv-shape mirrored novel detail.
func GetMirroredNovelDetail(c *gin.Context) {
	novelID, ok := parsePositiveQueryID(c, "novel_id")
	if !ok {
		return
	}
	record, err := pixezsvc.GetMirrorNovel(c.Request.Context(), novelID)
	if err != nil || record.DetailJSON == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "novel mirror not found"})
		return
	}
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.String(http.StatusOK, pixezsvc.RewritePximgURLs(record.DetailJSON, mirrorURLPrefix(c)))
}

// GetMirroredNovelText returns Pixiv webview JSON for mirrored novel text.
func GetMirroredNovelText(c *gin.Context) {
	novelID, ok := parsePositiveQueryID(c, "novel_id")
	if !ok {
		return
	}
	record, err := pixezsvc.GetMirrorNovel(c.Request.Context(), novelID)
	if err != nil || record.TextJSON == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "novel mirror not found"})
		return
	}
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.String(http.StatusOK, pixezsvc.RewritePximgURLs(record.TextJSON, mirrorURLPrefix(c)))
}

// ListMirroredIllusts returns paginated mirrored illustration read-model rows.
func ListMirroredIllusts(c *gin.Context) {
	page, pageSize := parsePage(c)
	var total int64
	query := db.DB(c.Request.Context()).Model(&model.PixezMirrorIllust{})
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(errQueryMirrorStatusFailed))
		return
	}
	var records []model.PixezMirrorIllust
	if err := query.Order("updated_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&records).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(errQueryMirrorStatusFailed))
		return
	}
	items := make([]gin.H, 0, len(records))
	for _, record := range records {
		title, userName := extractIllustDisplay(record.DetailJSON)
		items = append(items, gin.H{
			"task_id":       record.TaskID,
			"illust_id":     record.IllustID,
			"title":         title,
			"user_name":     userName,
			"status":        record.Status,
			"success_count": record.SuccessCount,
			"total_count":   record.TotalCount,
			"has_mirror":    record.SuccessCount > 0,
			"created_at":    record.CreatedAt,
			"updated_at":    record.UpdatedAt,
		})
	}
	c.JSON(http.StatusOK, util.OK(gin.H{"items": items, "total": total, "page": page, "page_size": pageSize}))
}

// ListMirroredNovels returns paginated mirrored novel read-model rows.
func ListMirroredNovels(c *gin.Context) {
	page, pageSize := parsePage(c)
	var total int64
	query := db.DB(c.Request.Context()).Model(&model.PixezMirrorNovel{})
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(errQueryMirrorStatusFailed))
		return
	}
	var records []model.PixezMirrorNovel
	if err := query.Order("updated_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&records).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(errQueryMirrorStatusFailed))
		return
	}
	items := make([]gin.H, 0, len(records))
	for _, record := range records {
		title, userName := extractNovelDisplay(record.DetailJSON)
		items = append(items, gin.H{
			"task_id":    record.TaskID,
			"novel_id":   record.NovelID,
			"title":      title,
			"user_name":  userName,
			"status":     record.Status,
			"has_mirror": record.SuccessCount > 0,
			"created_at": record.CreatedAt,
			"updated_at": record.UpdatedAt,
		})
	}
	c.JSON(http.StatusOK, util.OK(gin.H{"items": items, "total": total, "page": page, "page_size": pageSize}))
}

// DeleteMirroredIllust deletes one mirrored illustration read-model and marks its uploads deleted.
func DeleteMirroredIllust(c *gin.Context) {
	illustID, ok := parsePositiveIDParam(c, "illust_id")
	if !ok {
		return
	}
	deleted, err := deleteMirroredIllust(c.Request.Context(), illustID)
	if err != nil {
		c.JSON(http.StatusOK, util.Err(errDeleteMirrorFailed))
		return
	}
	c.JSON(http.StatusOK, util.OK(gin.H{"deleted": deleted, "illust_id": illustID}))
}

// DeleteMirroredNovel deletes one mirrored novel read-model.
func DeleteMirroredNovel(c *gin.Context) {
	novelID, ok := parsePositiveIDParam(c, "novel_id")
	if !ok {
		return
	}
	if err := db.DB(c.Request.Context()).Where("novel_id = ?", novelID).Delete(&model.PixezMirrorNovel{}).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(errDeleteMirrorFailed))
		return
	}
	c.JSON(http.StatusOK, util.OK(gin.H{"deleted": true, "novel_id": novelID}))
}

// BatchDeleteMirroredItems deletes mirrored items by target type.
func BatchDeleteMirroredItems(c *gin.Context) {
	var req struct {
		TargetType string  `json:"target_type"`
		IDs        []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, util.Err(errInvalidRequestBody))
		return
	}
	deleted := 0
	switch req.TargetType {
	case model.PixezMirrorTargetIllust:
		for _, id := range req.IDs {
			ok, err := deleteMirroredIllust(c.Request.Context(), id)
			if err != nil {
				c.JSON(http.StatusOK, util.Err(errDeleteMirrorFailed))
				return
			}
			if ok {
				deleted++
			}
		}
	case model.PixezMirrorTargetNovel:
		result := db.DB(c.Request.Context()).Where("novel_id IN ?", req.IDs).Delete(&model.PixezMirrorNovel{})
		if result.Error != nil {
			c.JSON(http.StatusOK, util.Err(errDeleteMirrorFailed))
			return
		}
		deleted = int(result.RowsAffected)
	default:
		c.JSON(http.StatusBadRequest, util.Err(errInvalidRequestBody))
		return
	}
	c.JSON(http.StatusOK, util.OK(gin.H{"deleted_count": deleted}))
}

//nolint:dupl // Illust and novel mirror dispatch flows keep parallel structures for distinct model types
func dispatchIllustMirrorIfNeeded(c *gin.Context, illustID int64) (model.PixezMirrorIllust, error) {
	record, err := pixezsvc.GetMirrorIllust(c.Request.Context(), illustID)
	if err == nil && record.Status != model.PixezMirrorStatusFailed {
		return record, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return record, err
	}
	payload, _ := json.Marshal(mirrorIllustPayload{IllustID: illustID})
	taskID, err := task.DispatchTask(c.Request.Context(), task.TaskTypePixezMirrorIllust, payload, "api")
	if err != nil {
		return record, err
	}
	return pixezsvc.EnsureMirrorIllustQueued(c.Request.Context(), illustID, taskID)
}

//nolint:dupl // Illust and novel mirror dispatch flows keep parallel structures for distinct model types
func dispatchNovelMirrorIfNeeded(c *gin.Context, novelID int64) (model.PixezMirrorNovel, error) {
	record, err := pixezsvc.GetMirrorNovel(c.Request.Context(), novelID)
	if err == nil && record.Status != model.PixezMirrorStatusFailed {
		return record, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return record, err
	}
	payload, _ := json.Marshal(mirrorNovelPayload{NovelID: novelID})
	taskID, err := task.DispatchTask(c.Request.Context(), task.TaskTypePixezMirrorNovel, payload, "api")
	if err != nil {
		return record, err
	}
	return pixezsvc.EnsureMirrorNovelQueued(c.Request.Context(), novelID, taskID)
}

func parsePositiveIDParam(c *gin.Context, key string) (int64, bool) {
	raw := c.Param(key)
	id, err := strconv.ParseInt(raw, 10, 64)
	if raw == "" || err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, util.Err(fmt.Sprintf("%s is required", key)))
		return 0, false
	}
	return id, true
}

func parsePositiveQueryID(c *gin.Context, key string) (int64, bool) {
	raw := c.Query(key)
	id, err := strconv.ParseInt(raw, 10, 64)
	if raw == "" || err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s is required", key)})
		return 0, false
	}
	return id, true
}

func parsePage(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}

func mirrorURLPrefix(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil || c.Request.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + c.Request.Host + "/mirror/pximg"
}

func extractIllustDisplay(raw string) (string, string) {
	var payload struct {
		Illust struct {
			Title string `json:"title"`
			User  struct {
				Name string `json:"name"`
			} `json:"user"`
		} `json:"illust"`
	}
	_ = json.Unmarshal([]byte(raw), &payload)
	return payload.Illust.Title, payload.Illust.User.Name
}

func extractNovelDisplay(raw string) (string, string) {
	var payload struct {
		Novel struct {
			Title string `json:"title"`
			User  struct {
				Name string `json:"name"`
			} `json:"user"`
		} `json:"novel"`
	}
	_ = json.Unmarshal([]byte(raw), &payload)
	return payload.Novel.Title, payload.Novel.User.Name
}

func deleteMirroredIllust(ctx context.Context, illustID int64) (bool, error) {
	var record model.PixezMirrorIllust
	err := db.DB(ctx).Where("illust_id = ?", illustID).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var files []model.PixezMirrorImageFile
	_ = json.Unmarshal([]byte(record.ImageFilesJSON), &files)
	for _, file := range files {
		var upload model.Upload
		if err := db.DB(ctx).Where("id = ?", file.UploadID).First(&upload).Error; err == nil {
			_ = db.DB(ctx).Model(&upload).Update("status", model.UploadStatusDeleted).Error
			if upload.StorageDriver == "local" {
				_ = os.Remove(upload.FilePath)
			} else if upload.FilePath != "" {
				_ = storage.DeleteObject(ctx, upload.FilePath)
			}
		}
	}
	if err := db.DB(ctx).Where("illust_id = ?", illustID).Delete(&model.PixezMirrorIllust{}).Error; err != nil {
		return false, err
	}
	return true, nil
}
