// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

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

// 异步任务名称与管理类型定义
const (
	// PixezMirrorTask 镜像资源任务标识
	PixezMirrorTask = "pixez:mirror"
	// PixezExportBookmarksTask 导出收藏任务标识
	PixezExportBookmarksTask = "pixez:export_bookmarks"
	// PixezAutoMirrorTask 自动入队收藏镜像任务标识
	PixezAutoMirrorTask = "pixez:auto_enqueue_bookmark_mirrors"
	// PixezImportLegacyTask 导入旧后端任务标识
	PixezImportLegacyTask = "pixez:import_legacy_server"

	// TaskTypePixezMirror 镜像资源管理类型
	TaskTypePixezMirror = "pixez_mirror"
	// TaskTypePixezExportBookmarks 导出收藏管理类型
	TaskTypePixezExportBookmarks = "pixez_export_bookmarks"
	// TaskTypePixezAutoEnqueueBookmarkMirrors 自动入队收藏镜像管理类型
	TaskTypePixezAutoEnqueueBookmarkMirrors = "pixez_auto_enqueue_bookmark_mirrors"
	// TaskTypePixezImportLegacyServer 导入旧后端管理类型
	TaskTypePixezImportLegacyServer = "pixez_import_legacy_server"
)

var (
	// PixezMirrorMeta task metadata
	PixezMirrorMeta = task.TaskMeta{
		Type:         TaskTypePixezMirror,
		AsynqTask:    PixezMirrorTask,
		Name:         "PixEz 镜像资源",
		Description:  "抓取 Pixiv 插画或小说详情并保存镜像记录",
		SupportsTime: false,
		MaxRetry:     task.DefaultMaxRetry,
		Queue:        task.QueueDefault,
		Retryable:    true,
		Params: []task.TaskParam{
			{Name: "target_type", Label: "目标类型", Type: "number", Required: true, Placeholder: "0 或 1", Description: "0 表示插画，1 表示小说"},
			{Name: "target_id", Label: "资源 ID", Type: "number", Required: true, Placeholder: "123456", Description: "Pixiv 插画或小说 ID"},
		},
	}

	// PixezExportBookmarksMeta task metadata
	PixezExportBookmarksMeta = task.TaskMeta{
		Type:         TaskTypePixezExportBookmarks,
		AsynqTask:    PixezExportBookmarksTask,
		Name:         "PixEz 导出收藏",
		Description:  "导出已同步 Pixiv 账号的插画或小说收藏并增量维护 removed 状态",
		SupportsTime: false,
		MaxRetry:     task.DefaultMaxRetry,
		Queue:        task.QueueDefault,
		Retryable:    true,
		Params: []task.TaskParam{
			{Name: "target_type", Label: "目标类型", Type: "number", Required: false, Placeholder: "0 或 1，留空表示全部", Description: "0 表示插画，1 表示小说，留空表示全部"},
			{Name: "pixiv_user_id", Label: "Pixiv 用户 ID", Type: "string", Required: false, Placeholder: "留空表示全部账号", Description: "只导出指定 Pixiv 用户时填写"},
		},
	}

	// PixezAutoMirrorMeta task metadata
	PixezAutoMirrorMeta = task.TaskMeta{
		Type:         TaskTypePixezAutoEnqueueBookmarkMirrors,
		AsynqTask:    PixezAutoMirrorTask,
		Name:         "PixEz 收藏自动入队镜像",
		Description:  "扫描收藏 read-model，把未镜像或失败的收藏批量下发镜像任务",
		SupportsTime: false,
		MaxRetry:     task.DefaultMaxRetry,
		Queue:        task.QueueDefault,
		Retryable:    true,
		Params: []task.TaskParam{
			{Name: "target_type", Label: "目标类型", Type: "number", Required: false, Placeholder: "0 或 1，留空表示全部", Description: "0 表示插画，1 表示小说，留空表示全部"},
			{Name: "limit", Label: "数量上限", Type: "number", Required: false, Placeholder: "50", Description: "本次最多入队数量"},
		},
	}

	// PixezImportLegacyMeta task metadata
	PixezImportLegacyMeta = task.TaskMeta{
		Type:         TaskTypePixezImportLegacyServer,
		AsynqTask:    PixezImportLegacyTask,
		Name:         "PixEz 导入旧后端",
		Description:  "从旧 server/pixez-sync.db 和 server/data/mirror 导入业务数据",
		SupportsTime: false,
		MaxRetry:     1,
		Queue:        task.QueueDefault,
		Retryable:    false,
		Params: []task.TaskParam{
			{Name: "sqlite_path", Label: "旧 SQLite 路径", Type: "string", Required: false, Placeholder: "server/pixez-sync.db", Description: "旧 PixEz Sync SQLite 文件"},
			{Name: "mirror_dir", Label: "旧镜像目录", Type: "string", Required: false, Placeholder: "server/data/mirror", Description: "旧插画镜像文件目录"},
			{Name: "dry_run", Label: "只预览", Type: "boolean", Required: false, Placeholder: "false", Description: "是否只统计不写入"},
		},
	}
)

type mirrorPayload struct {
	TargetType int   `json:"target_type"`
	TargetID   int64 `json:"target_id"`
}

type exportBookmarksPayload struct {
	TargetType  *int   `json:"target_type,omitempty"`
	PixivUserID string `json:"pixiv_user_id,omitempty"`
}

type autoMirrorPayload struct {
	TargetType *int `json:"target_type,omitempty"`
	Limit      int  `json:"limit,omitempty"`
}

type bookmarkMirrorCandidate struct {
	RowID            uint
	TargetID         int64
	MirrorStatus     int
	MirrorRetryCount int
}

// MirrorTaskHandler mirrors one Pixiv illustration or novel.
type MirrorTaskHandler struct{}

// ValidatePayload validates mirror payloads.
func (h *MirrorTaskHandler) ValidatePayload(payload []byte) ([]byte, error) {
	var req mirrorPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("invalid JSON payload: %w", err)
	}
	if req.TargetType != TargetTypeIllust && req.TargetType != TargetTypeNovel {
		return nil, errors.New("target_type must be 0 (illust) or 1 (novel)")
	}
	if req.TargetID <= 0 {
		return nil, errors.New("target_id is required")
	}
	return json.Marshal(req)
}

// Execute runs the mirror task.
func (h *MirrorTaskHandler) Execute(ctx context.Context, payload []byte) (*task.TaskResult, error) {
	var req mirrorPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("parse mirror payload: %w", err)
	}
	if req.TargetType == TargetTypeIllust {
		return executeMirrorTask(ctx, "插画", "illust", req.TargetID, func(taskID string) error {
			_, err := pixezsvc.EnsureMirrorIllustQueued(ctx, req.TargetID, taskID)
			return err
		}, func(taskID string) error {
			return pixezsvc.ProcessMirrorIllust(ctx, pixezsvc.DefaultClient, taskID, req.TargetID)
		})
	}
	return executeMirrorTask(ctx, "小说", "novel", req.TargetID, func(taskID string) error {
		_, err := pixezsvc.EnsureMirrorNovelQueued(ctx, req.TargetID, taskID)
		return err
	}, func(taskID string) error {
		return pixezsvc.ProcessMirrorNovel(ctx, pixezsvc.DefaultClient, taskID, req.TargetID)
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

// ExportBookmarksTaskHandler exports Pixiv bookmarks (illustrations or novels).
type ExportBookmarksTaskHandler struct{}

// ValidatePayload validates export bookmarks payloads.
func (h *ExportBookmarksTaskHandler) ValidatePayload(payload []byte) ([]byte, error) {
	var req exportBookmarksPayload
	if len(payload) == 0 {
		return json.Marshal(exportBookmarksPayload{})
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("invalid JSON payload: %w", err)
	}
	if req.TargetType != nil {
		if *req.TargetType != TargetTypeIllust && *req.TargetType != TargetTypeNovel {
			return nil, errors.New("target_type must be 0 (illust) or 1 (novel)")
		}
	}
	req.PixivUserID = strings.TrimSpace(req.PixivUserID)
	return json.Marshal(req)
}

// Execute runs the bookmarks export task.
func (h *ExportBookmarksTaskHandler) Execute(ctx context.Context, payload []byte) (*task.TaskResult, error) {
	var req exportBookmarksPayload
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, fmt.Errorf("parse export bookmarks payload: %w", err)
		}
	}

	if req.TargetType == nil {
		task.AppendLog(ctx, "未指定目标类型，同时导出插画和小说收藏")
		illustResult, err := executeExportBookmarksTask(ctx, req.PixivUserID, "插画", pixezsvc.ExportIllustBookmarks)
		if err != nil {
			return nil, err
		}
		novelResult, err := executeExportBookmarksTask(ctx, req.PixivUserID, "小说", pixezsvc.ExportNovelBookmarks)
		if err != nil {
			return nil, err
		}
		msg := fmt.Sprintf("%s; %s", illustResult.Message, novelResult.Message)
		return &task.TaskResult{Message: msg}, nil
	}

	if *req.TargetType == TargetTypeIllust {
		return executeExportBookmarksTask(ctx, req.PixivUserID, "插画", pixezsvc.ExportIllustBookmarks)
	}
	return executeExportBookmarksTask(ctx, req.PixivUserID, "小说", pixezsvc.ExportNovelBookmarks)
}

func executeExportBookmarksTask(
	ctx context.Context,
	pixivUserID string,
	label string,
	exportFn func(context.Context, *pixezsvc.Client, string) (pixezsvc.ExportSummary, error),
) (*task.TaskResult, error) {
	task.AppendLog(ctx, "开始导出 PixEz %s收藏 pixiv_user_id=%s", label, emptyAsAll(pixivUserID))
	summary, err := exportFn(ctx, pixezsvc.DefaultClient, pixivUserID)
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
	if req.TargetType != nil {
		if *req.TargetType != TargetTypeIllust && *req.TargetType != TargetTypeNovel {
			return nil, errors.New("target_type must be 0 (illust) or 1 (novel)")
		}
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

	targetTypeStr := "<all>"
	if req.TargetType != nil {
		if *req.TargetType == TargetTypeIllust {
			targetTypeStr = "illust"
		} else {
			targetTypeStr = "novel"
		}
	}
	task.AppendLog(ctx, "开始自动入队收藏镜像任务，参数: target_type=%s, limit=%d", targetTypeStr, req.Limit)

	enqueued := 0
	if req.TargetType == nil || *req.TargetType == TargetTypeIllust {
		n, err := enqueueIllustBookmarkMirrors(ctx, req.Limit-enqueued)
		enqueued += n
		if err != nil {
			return nil, err
		}
	}
	if enqueued < req.Limit && (req.TargetType == nil || *req.TargetType == TargetTypeNovel) {
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

// validateBookmarkExportPayload and parseBookmarkExportPayload removed as they are integrated into ExportBookmarksTaskHandler

func emptyAsAll(value string) string {
	if value == "" {
		return "<all>"
	}
	return value
}

func enqueueIllustBookmarkMirrors(ctx context.Context, limit int) (int, error) {
	return enqueueBookmarkMirrors(ctx, limit, "bookmark_illusts", "illust_id", &model.PixezBookmarkIllust{}, TaskTypePixezMirror, func(targetID int64) []byte {
		payload, _ := json.Marshal(mirrorPayload{TargetType: TargetTypeIllust, TargetID: targetID})
		return payload
	}, func(targetID int64, taskID string) error {
		_, err := pixezsvc.EnsureMirrorIllustQueued(ctx, targetID, taskID)
		return err
	})
}

func enqueueNovelBookmarkMirrors(ctx context.Context, limit int) (int, error) {
	return enqueueBookmarkMirrors(ctx, limit, "bookmark_novels", "novel_id", &model.PixezBookmarkNovel{}, TaskTypePixezMirror, func(targetID int64) []byte {
		payload, _ := json.Marshal(mirrorPayload{TargetType: TargetTypeNovel, TargetID: targetID})
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
