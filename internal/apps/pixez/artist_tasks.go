// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package pixez

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	pixezsvc "github.com/Rain-kl/Wavelet/internal/service/pixez"
	"github.com/Rain-kl/Wavelet/internal/task"
	"github.com/Rain-kl/Wavelet/pkg/logger"
	"github.com/google/uuid"
)

const (
	// PixezArtistScanTask is the creator pagination task.
	PixezArtistScanTask = "pixez:artist_scan"
	// PixezArtistDueTask is the periodic due-subscription dispatcher.
	PixezArtistDueTask = "pixez:artist_due"
	// TaskTypePixezArtistScan names creator scans in the task management UI.
	TaskTypePixezArtistScan = "pixez_artist_scan"
	// TaskTypePixezArtistDue names the daily subscription checking dispatcher.
	TaskTypePixezArtistDue = "pixez_artist_due"
)

var (
	// PixezArtistScanMeta describes resumable author scans.
	PixezArtistScanMeta = task.TaskMeta{Type: TaskTypePixezArtistScan, AsynqTask: PixezArtistScanTask, Name: "订阅作者作品扫描", Description: "分页发现作品并只备份缺失内容", Queue: task.QueueDefault, MaxRetry: task.DefaultMaxRetry, Retryable: true, Params: []task.TaskParam{{Name: "run_id", Label: "扫描记录 ID", Type: "string", Required: true}}}
	// PixezArtistDueMeta checks daily deadlines without downloading in the scheduler.
	PixezArtistDueMeta = task.TaskMeta{Type: TaskTypePixezArtistDue, AsynqTask: PixezArtistDueTask, Name: "订阅每日同步检查", Description: "检查到期订阅并下发独立作者扫描", Queue: task.QueueDefault, MaxRetry: task.DefaultMaxRetry, Retryable: true}
)

type artistScanPayload struct {
	RunID string `json:"run_id"`
}

// ArtistScanTaskHandler runs one persisted creator scan.
type ArtistScanTaskHandler struct{}

// ValidatePayload validates the persisted scan identifier.
func (h *ArtistScanTaskHandler) ValidatePayload(payload []byte) ([]byte, error) {
	var req artistScanPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, err
	}
	if _, err := uuid.Parse(req.RunID); err != nil {
		return nil, errors.New("run_id must be a valid UUID")
	}
	return json.Marshal(req)
}

// Execute resumes one scan through the registered task framework.
func (h *ArtistScanTaskHandler) Execute(ctx context.Context, payload []byte) (*task.TaskResult, error) {
	normalized, err := h.ValidatePayload(payload)
	if err != nil {
		return nil, err
	}
	var req artistScanPayload
	if err = json.Unmarshal(normalized, &req); err != nil {
		return nil, err
	}
	task.AppendLog(ctx, "开始作者扫描 run=%s", req.RunID)
	if err = pixezsvc.ScanArtistRun(ctx, pixezsvc.DefaultClient, req.RunID, enqueueArtistMirror); err != nil {
		task.AppendLog(ctx, "作者扫描失败: %v", err)
		if cleanupErr := pixezsvc.FailArtistRun(context.WithoutCancel(ctx), req.RunID, task.IsFinalAttempt(ctx)); cleanupErr != nil {
			logger.ErrorF(ctx, "[PixEz] update scan failure: %v", cleanupErr)
		}
		return nil, err
	}
	run, err := pixezsvc.GetArtistRun(ctx, req.RunID)
	if err != nil {
		return nil, err
	}
	task.AppendLog(ctx, "作者扫描结束 status=%s discovered=%d queued=%d", run.Status, run.DiscoveredCount, run.QueuedCount)
	detail, err := json.Marshal(run)
	if err != nil {
		return nil, err
	}
	return &task.TaskResult{Message: "作者扫描结束", Detail: string(detail)}, nil
}

// ArtistDueTaskHandler dispatches enabled subscriptions whose 24-hour deadline has passed.
type ArtistDueTaskHandler struct{}

// ValidatePayload accepts an empty periodic-check payload.
func (h *ArtistDueTaskHandler) ValidatePayload(payload []byte) ([]byte, error) {
	var req struct{}
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &req); err != nil {
			return nil, err
		}
	}
	return []byte("{}"), nil
}

// Execute discovers authors and dispatches due subscriptions.
func (h *ArtistDueTaskHandler) Execute(ctx context.Context, payload []byte) (*task.TaskResult, error) {
	if _, err := h.ValidatePayload(payload); err != nil {
		return nil, err
	}
	if err := pixezsvc.BackfillArtistMirrors(ctx); err != nil {
		return nil, err
	}
	subs, err := pixezsvc.DueArtistSubscriptions(ctx)
	if err != nil {
		return nil, err
	}
	enqueued := 0
	var firstErr error
	for _, sub := range subs {
		_, e := dispatchArtistScan(ctx, sub.ArtistID, sub.TargetType, true, "daily")
		if e != nil {
			logger.ErrorF(ctx, "[PixEz] dispatch subscription %d: %v", sub.ID, e)
			if firstErr == nil {
				firstErr = e
			}
			continue
		}
		enqueued++
	}
	task.AppendLog(ctx, "到期订阅检查完成 dispatched=%d", enqueued)
	if firstErr != nil {
		return nil, firstErr
	}
	return &task.TaskResult{Message: fmt.Sprintf("已安排 %d 个订阅扫描", enqueued)}, nil
}

func dispatchArtistScan(ctx context.Context, id int64, target string, download bool, trigger string) (model.PixezArtistSyncRun, error) {
	run, created, err := pixezsvc.CreateArtistRun(ctx, id, target, download, trigger)
	if err != nil || !created {
		return run, err
	}
	payload, err := json.Marshal(artistScanPayload{RunID: run.ID})
	if err != nil {
		return run, err
	}
	taskID, err := task.DispatchTask(ctx, TaskTypePixezArtistScan, payload, "api")
	if err != nil {
		if cleanupErr := pixezsvc.FailArtistDispatch(context.WithoutCancel(ctx), run); cleanupErr != nil {
			logger.ErrorF(ctx, "[PixEz] release failed dispatch: %v", cleanupErr)
		}
		return run, err
	}
	run.TaskID = taskID
	err = db.DB(ctx).Model(&model.PixezArtistSyncRun{}).Where("id = ?", run.ID).Update("task_id", taskID).Error
	return run, err
}

func enqueueArtistMirror(ctx context.Context, target string, id int64) (bool, error) {
	needs, err := pixezsvc.MirrorNeedsWork(ctx, target, id)
	if err != nil || !needs {
		return false, err
	}
	table, column := "mirror_illust", "illust_id"
	kind := TargetTypeIllust
	if target == model.PixezMirrorTargetNovel {
		table, column = "mirror_novel", "novel_id"
		kind = TargetTypeNovel
	}
	var existing struct{ TaskID string }
	if err = db.DB(ctx).Table(table).Where(column+" = ?", id).Scan(&existing).Error; err != nil {
		return false, err
	}
	if existing.TaskID != "" {
		var active int64
		if err = db.DB(ctx).Model(&model.TaskExecution{}).Where("task_id = ? AND status IN (?, ?) AND updated_at > ?", existing.TaskID, model.TaskExecutionStatusPending, model.TaskExecutionStatusRunning, time.Now().Add(-15*time.Minute)).Count(&active).Error; err != nil {
			return false, err
		}
		if active > 0 {
			return false, nil
		}
	}
	payload, err := json.Marshal(mirrorPayload{TargetType: kind, TargetID: id})
	if err != nil {
		return false, err
	}
	taskID, err := task.DispatchTask(ctx, TaskTypePixezMirror, payload, "system")
	if err != nil {
		return false, err
	}
	if target == model.PixezMirrorTargetIllust {
		_, err = pixezsvc.EnsureMirrorIllustQueued(ctx, id, taskID)
	} else {
		_, err = pixezsvc.EnsureMirrorNovelQueued(ctx, id, taskID)
	}
	return err == nil, err
}
