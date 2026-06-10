// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package pixez

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	pixezsvc "github.com/Rain-kl/Wavelet/internal/service/pixez"
	"github.com/Rain-kl/Wavelet/internal/task"
)

type mirrorIllustPayload struct {
	IllustID int64 `json:"illust_id"`
}

type mirrorNovelPayload struct {
	NovelID int64 `json:"novel_id"`
}

type bookmarkExportPayload struct {
	PixivUserID string `json:"pixiv_user_id,omitempty"`
}

type autoMirrorPayload struct {
	TargetType string `json:"target_type,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

type bookmarkMirrorCandidate struct {
	RowID            uint
	TargetID         int64
	MirrorStatus     int
	MirrorRetryCount int
}

const (
	defaultAutoMirrorLimit = 50
	maxAutoMirrorLimit     = 500
)

// MirrorIllustTaskHandler mirrors one Pixiv illustration.
type MirrorIllustTaskHandler struct{}

// ValidatePayload validates illustration mirror payloads.
func (h *MirrorIllustTaskHandler) ValidatePayload(payload []byte) ([]byte, error) {
	var req mirrorIllustPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("invalid JSON payload: %w", err)
	}
	if req.IllustID <= 0 {
		return nil, errors.New("illust_id is required")
	}
	return json.Marshal(req)
}

// Execute runs the illustration mirror task.
func (h *MirrorIllustTaskHandler) Execute(ctx context.Context, payload []byte) (*task.TaskResult, error) {
	var req mirrorIllustPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("parse illust mirror payload: %w", err)
	}
	return executeMirrorTask(ctx, "插画", "illust", req.IllustID, func(taskID string) error {
		_, err := pixezsvc.EnsureMirrorIllustQueued(ctx, req.IllustID, taskID)
		return err
	}, func(taskID string) error {
		return pixezsvc.ProcessMirrorIllust(ctx, pixezsvc.DefaultClient, taskID, req.IllustID)
	})
}

// MirrorNovelTaskHandler mirrors one Pixiv novel.
type MirrorNovelTaskHandler struct{}

// ValidatePayload validates novel mirror payloads.
func (h *MirrorNovelTaskHandler) ValidatePayload(payload []byte) ([]byte, error) {
	var req mirrorNovelPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("invalid JSON payload: %w", err)
	}
	if req.NovelID <= 0 {
		return nil, errors.New("novel_id is required")
	}
	return json.Marshal(req)
}

// Execute runs the novel mirror task.
func (h *MirrorNovelTaskHandler) Execute(ctx context.Context, payload []byte) (*task.TaskResult, error) {
	var req mirrorNovelPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("parse novel mirror payload: %w", err)
	}
	return executeMirrorTask(ctx, "小说", "novel", req.NovelID, func(taskID string) error {
		_, err := pixezsvc.EnsureMirrorNovelQueued(ctx, req.NovelID, taskID)
		return err
	}, func(taskID string) error {
		return pixezsvc.ProcessMirrorNovel(ctx, pixezsvc.DefaultClient, taskID, req.NovelID)
	})
}

func executeMirrorTask(
	ctx context.Context,
	label string,
	idName string,
	targetID int64,
	ensure func(taskID string) error,
	process func(taskID string) error,
) (*task.TaskResult, error) {
	taskID := task.GetTaskID(ctx)
	if err := ensure(taskID); err != nil {
		return nil, fmt.Errorf("ensure %s mirror read-model: %w", idName, err)
	}
	task.AppendLog(ctx, "开始镜像 Pixiv %s %s_id=%d", label, idName, targetID)
	if err := process(taskID); err != nil {
		task.AppendLog(ctx, "%s镜像失败: %v", label, err)
		return nil, err
	}
	msg := fmt.Sprintf("PixEz %s镜像完成 %s_id=%d", label, idName, targetID)
	task.AppendLog(ctx, "%s", msg)
	return &task.TaskResult{Message: msg}, nil
}

// ExportIllustBookmarksTaskHandler exports Pixiv illustration bookmarks.
type ExportIllustBookmarksTaskHandler struct{}

// ValidatePayload validates optional export filters.
func (h *ExportIllustBookmarksTaskHandler) ValidatePayload(payload []byte) ([]byte, error) {
	return validateBookmarkExportPayload(payload)
}

// Execute runs illustration bookmark export.
func (h *ExportIllustBookmarksTaskHandler) Execute(ctx context.Context, payload []byte) (*task.TaskResult, error) {
	return executeExportBookmarksTask(ctx, payload, "插画", pixezsvc.ExportIllustBookmarks)
}

// ExportNovelBookmarksTaskHandler exports Pixiv novel bookmarks.
type ExportNovelBookmarksTaskHandler struct{}

// ValidatePayload validates optional export filters.
func (h *ExportNovelBookmarksTaskHandler) ValidatePayload(payload []byte) ([]byte, error) {
	return validateBookmarkExportPayload(payload)
}

// Execute runs novel bookmark export.
func (h *ExportNovelBookmarksTaskHandler) Execute(ctx context.Context, payload []byte) (*task.TaskResult, error) {
	return executeExportBookmarksTask(ctx, payload, "小说", pixezsvc.ExportNovelBookmarks)
}

func executeExportBookmarksTask(
	ctx context.Context,
	payload []byte,
	label string,
	exportFn func(context.Context, *pixezsvc.Client, string) (pixezsvc.ExportSummary, error),
) (*task.TaskResult, error) {
	req, err := parseBookmarkExportPayload(payload)
	if err != nil {
		return nil, err
	}
	task.AppendLog(ctx, "开始导出 PixEz %s收藏 pixiv_user_id=%s", label, emptyAsAll(req.PixivUserID))
	summary, err := exportFn(ctx, pixezsvc.DefaultClient, req.PixivUserID)
	if err != nil {
		task.AppendLog(ctx, "%s收藏导出失败: %v", label, err)
		return nil, err
	}
	msg := fmt.Sprintf("%s收藏导出完成 users=%d runs=%d total=%d new=%d updated=%d removed=%d",
		label, summary.UserCount, summary.RunCount, summary.TotalCount, summary.NewCount, summary.UpdatedCount, summary.RemovedCount)
	task.AppendLog(ctx, "%s", msg)
	return &task.TaskResult{Message: msg}, nil
}

// AutoEnqueueBookmarkMirrorsTaskHandler dispatches mirror tasks for bookmark rows.
type AutoEnqueueBookmarkMirrorsTaskHandler struct{}

// ValidatePayload validates auto enqueue payloads.
func (h *AutoEnqueueBookmarkMirrorsTaskHandler) ValidatePayload(payload []byte) ([]byte, error) {
	if len(payload) == 0 {
		return json.Marshal(autoMirrorPayload{Limit: defaultAutoMirrorLimit})
	}
	var req autoMirrorPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("invalid JSON payload: %w", err)
	}
	req.TargetType = strings.TrimSpace(req.TargetType)
	if req.TargetType != "" && req.TargetType != model.PixezMirrorTargetIllust && req.TargetType != model.PixezMirrorTargetNovel {
		return nil, errors.New("target_type must be illust or novel")
	}
	if req.Limit <= 0 {
		req.Limit = defaultAutoMirrorLimit
	}
	if req.Limit > maxAutoMirrorLimit {
		req.Limit = maxAutoMirrorLimit
	}
	return json.Marshal(req)
}

// Execute dispatches bookmark mirror tasks.
func (h *AutoEnqueueBookmarkMirrorsTaskHandler) Execute(ctx context.Context, payload []byte) (*task.TaskResult, error) {
	var req autoMirrorPayload
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, fmt.Errorf("parse auto mirror payload: %w", err)
		}
	}
	if req.Limit <= 0 {
		req.Limit = defaultAutoMirrorLimit
	}

	task.AppendLog(ctx, "开始自动入队收藏镜像任务，参数: target_type=%s, limit=%d", emptyAsAll(req.TargetType), req.Limit)

	enqueued := 0
	if req.TargetType == "" || req.TargetType == model.PixezMirrorTargetIllust {
		n, err := enqueueIllustBookmarkMirrors(ctx, req.Limit-enqueued)
		enqueued += n
		if err != nil {
			return nil, err
		}
	}
	if enqueued < req.Limit && (req.TargetType == "" || req.TargetType == model.PixezMirrorTargetNovel) {
		n, err := enqueueNovelBookmarkMirrors(ctx, req.Limit-enqueued)
		enqueued += n
		if err != nil {
			return nil, err
		}
	}
	msg := fmt.Sprintf("PixEz 收藏镜像入队完成 count=%d", enqueued)
	task.AppendLog(ctx, "%s", msg)
	return &task.TaskResult{Message: msg}, nil
}

// ImportLegacyServerTaskHandler imports legacy PixEz server data.
type ImportLegacyServerTaskHandler struct{}

// ValidatePayload validates legacy import payloads.
func (h *ImportLegacyServerTaskHandler) ValidatePayload(payload []byte) ([]byte, error) {
	req := pixezsvc.ImportLegacyRequest{
		SQLitePath: "server/pixez-sync.db",
		MirrorDir:  "server/data/mirror",
	}
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, fmt.Errorf("invalid JSON payload: %w", err)
		}
	}
	req.SQLitePath = strings.TrimSpace(req.SQLitePath)
	req.MirrorDir = strings.TrimSpace(req.MirrorDir)
	if req.SQLitePath == "" {
		req.SQLitePath = "server/pixez-sync.db"
	}
	if req.MirrorDir == "" {
		req.MirrorDir = "server/data/mirror"
	}
	return json.Marshal(req)
}

// Execute imports legacy PixEz server data.
func (h *ImportLegacyServerTaskHandler) Execute(ctx context.Context, payload []byte) (*task.TaskResult, error) {
	validated, err := h.ValidatePayload(payload)
	if err != nil {
		return nil, err
	}
	var req pixezsvc.ImportLegacyRequest
	if err := json.Unmarshal(validated, &req); err != nil {
		return nil, err
	}
	task.AppendLog(ctx, "开始导入旧 PixEz server sqlite=%s mirror_dir=%s dry_run=%v", req.SQLitePath, req.MirrorDir, req.DryRun)
	summary, err := pixezsvc.ImportLegacyServer(ctx, req)
	if err != nil {
		task.AppendLog(ctx, "旧 PixEz server 导入失败: %v", err)
		return nil, err
	}
	detail, _ := json.Marshal(summary)
	msg := fmt.Sprintf("旧 PixEz server 导入完成 users=%d bookmarks=%d/%d mirrors=%d/%d files=%d missing=%d",
		summary.PixivUsers, summary.BookmarkIllusts, summary.BookmarkNovels, summary.MirrorIllusts, summary.MirrorNovels, summary.ImportedFiles, summary.MissingFiles)
	task.AppendLog(ctx, "%s", msg)
	return &task.TaskResult{Message: msg, Detail: string(detail)}, nil
}

func validateBookmarkExportPayload(payload []byte) ([]byte, error) {
	req, err := parseBookmarkExportPayload(payload)
	if err != nil {
		return nil, err
	}
	return json.Marshal(req)
}

func parseBookmarkExportPayload(payload []byte) (bookmarkExportPayload, error) {
	if len(payload) == 0 {
		return bookmarkExportPayload{}, nil
	}
	var req bookmarkExportPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		return req, fmt.Errorf("invalid JSON payload: %w", err)
	}
	req.PixivUserID = strings.TrimSpace(req.PixivUserID)
	return req, nil
}

func emptyAsAll(value string) string {
	if value == "" {
		return "<all>"
	}
	return value
}

func enqueueIllustBookmarkMirrors(ctx context.Context, limit int) (int, error) {
	return enqueueBookmarkMirrors(ctx, limit, "bookmark_illusts", "illust_id", &model.PixezBookmarkIllust{}, task.TaskTypePixezMirrorIllust, func(targetID int64) []byte {
		payload, _ := json.Marshal(mirrorIllustPayload{IllustID: targetID})
		return payload
	}, func(targetID int64, taskID string) error {
		_, err := pixezsvc.EnsureMirrorIllustQueued(ctx, targetID, taskID)
		return err
	})
}

func enqueueNovelBookmarkMirrors(ctx context.Context, limit int) (int, error) {
	return enqueueBookmarkMirrors(ctx, limit, "bookmark_novels", "novel_id", &model.PixezBookmarkNovel{}, task.TaskTypePixezMirrorNovel, func(targetID int64) []byte {
		payload, _ := json.Marshal(mirrorNovelPayload{NovelID: targetID})
		return payload
	}, func(targetID int64, taskID string) error {
		_, err := pixezsvc.EnsureMirrorNovelQueued(ctx, targetID, taskID)
		return err
	})
}

func enqueueBookmarkMirrors(
	ctx context.Context,
	limit int,
	tableName string,
	targetColumn string,
	bookmarkModel any,
	taskType string,
	buildPayload func(targetID int64) []byte,
	ensure func(targetID int64, taskID string) error,
) (int, error) {
	if limit <= 0 {
		return 0, nil
	}
	candidates, err := loadBookmarkMirrorCandidates(ctx, tableName, targetColumn, limit)
	if err != nil {
		return 0, err
	}
	task.AppendLog(ctx, "表 %s 扫描到 %d 个待镜像候选", tableName, len(candidates))
	return enqueueBookmarkMirrorCandidates(ctx, candidates, bookmarkModel, taskType, buildPayload, ensure)
}

func loadBookmarkMirrorCandidates(ctx context.Context, tableName string, targetColumn string, limit int) ([]bookmarkMirrorCandidate, error) {
	var rows []struct {
		RowID            uint  `gorm:"column:id"`
		TargetID         int64 `gorm:"column:target_id"`
		MirrorStatus     int   `gorm:"column:mirror_status"`
		MirrorRetryCount int   `gorm:"column:mirror_retry_count"`
	}
	if err := db.DB(ctx).
		Table(tableName).
		Select("id, "+targetColumn+" AS target_id, mirror_status, mirror_retry_count").
		Where("removed = ? AND mirror_status IN ?", false, []int{model.PixezBookmarkMirrorNone, model.PixezBookmarkMirrorFailed}).
		Order("updated_at desc").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	candidates := make([]bookmarkMirrorCandidate, 0, len(rows))
	for _, row := range rows {
		candidates = append(candidates, bookmarkMirrorCandidate{
			RowID:            row.RowID,
			TargetID:         row.TargetID,
			MirrorStatus:     row.MirrorStatus,
			MirrorRetryCount: row.MirrorRetryCount,
		})
	}
	return candidates, nil
}

func enqueueBookmarkMirrorCandidates(
	ctx context.Context,
	candidates []bookmarkMirrorCandidate,
	bookmarkModel any,
	taskType string,
	buildPayload func(targetID int64) []byte,
	ensure func(targetID int64, taskID string) error,
) (int, error) {
	enqueued := 0
	for _, candidate := range candidates {
		taskID, err := task.DispatchTask(ctx, taskType, buildPayload(candidate.TargetID), "pixez-auto")
		if err != nil {
			return enqueued, err
		}
		if err := ensure(candidate.TargetID, taskID); err != nil {
			return enqueued, err
		}
		updates := map[string]any{
			"mirror_status": model.PixezBookmarkMirrorProcessing,
			"updated_at":    time.Now(),
		}
		if candidate.MirrorStatus == model.PixezBookmarkMirrorFailed {
			updates["mirror_retry_count"] = candidate.MirrorRetryCount + 1
		}
		if err := db.DB(ctx).Model(bookmarkModel).Where("id = ?", candidate.RowID).Updates(updates).Error; err != nil {
			return enqueued, err
		}
		enqueued++
	}
	return enqueued, nil
}
