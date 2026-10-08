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
	"github.com/Rain-kl/Wavelet/pkg/logger"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const mirrorLeaseDuration = 15 * time.Minute

type mirrorTokenKey struct{}

func mirrorTable(target string) (string, string) {
	if target == model.PixezMirrorTargetIllust {
		return "mirror_illust", "illust_id"
	}
	return "mirror_novel", "novel_id"
}

func validMirrorFile(ctx context.Context, file model.PixezMirrorImageFile) bool {
	if file.UploadID == 0 {
		return false
	}
	var n int64
	err := db.DB(ctx).Model(&model.Upload{}).Where("id = ? AND status IN (?, ?)", file.UploadID, model.UploadStatusPending, model.UploadStatusUsed).Count(&n).Error
	return err == nil && n == 1
}

func illustComplete(ctx context.Context, row model.PixezMirrorIllust) bool {
	var detail IllustDetail
	if json.Unmarshal([]byte(row.DetailJSON), &detail) != nil || detail.Illust.ID != row.IllustID || IsLimitUnknownIllust(detail.Illust) {
		return false
	}
	urls := CollectIllustImageURLs(detail)
	if len(urls) == 0 || (detail.Illust.PageCount > 0 && len(urls) < detail.Illust.PageCount) {
		return false
	}
	var files []model.PixezMirrorImageFile
	if json.Unmarshal([]byte(row.ImageFilesJSON), &files) != nil {
		return false
	}
	for page, u := range urls {
		found := false
		for _, f := range files {
			if f.Page == page && f.PixivURL == u && validMirrorFile(ctx, f) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func novelComplete(row model.PixezMirrorNovel) bool {
	var detail NovelDetail
	var content NovelWebContent
	return json.Unmarshal([]byte(row.DetailJSON), &detail) == nil && detail.Novel.ID == row.NovelID &&
		json.Unmarshal([]byte(row.TextJSON), &content) == nil && content.Text != ""
}

// MirrorNeedsWork checks content completeness rather than the historical success flag.
func MirrorNeedsWork(ctx context.Context, target string, id int64) (bool, error) {
	if target == model.PixezMirrorTargetIllust {
		r, err := GetMirrorIllust(ctx, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		if illustComplete(ctx, r) {
			return false, nil
		}
		return r.LeaseExpiresAt == nil || r.LeaseExpiresAt.Before(time.Now()), nil
	}
	r, err := GetMirrorNovel(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if novelComplete(r) {
		return false, nil
	}
	return r.LeaseExpiresAt == nil || r.LeaseExpiresAt.Before(time.Now()), nil
}

func runProtectedMirror(ctx context.Context, target string, id int64, taskID string, process func(context.Context) error) error {
	needed, err := MirrorNeedsWork(ctx, target, id)
	if err != nil || !needed {
		return err
	}
	table, column := mirrorTable(target)
	token := uuid.NewString()
	now := time.Now()
	result := db.DB(ctx).Table(table).Where(column+" = ? AND (lease_expires_at IS NULL OR lease_expires_at < ?)", id, now).
		Updates(map[string]any{"execution_token": token, "lease_expires_at": now.Add(mirrorLeaseDuration), "status": model.PixezMirrorStatusProcessing, "task_id": taskID})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return nil
	}
	leaseCtx, cancel := context.WithCancel(context.WithValue(ctx, mirrorTokenKey{}, token))
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-leaseCtx.Done():
				return
			case <-ticker.C:
				r := db.DB(leaseCtx).Table(table).Where(column+" = ? AND execution_token = ?", id, token).
					Update("lease_expires_at", time.Now().Add(mirrorLeaseDuration))
				if r.Error != nil || r.RowsAffected == 0 {
					cancel()
					return
				}
			}
		}
	}()
	err = process(leaseCtx)
	cancel()
	<-done
	// Cleanup is token-conditional, including when the caller cancelled its context.
	cleanupCtx := context.WithoutCancel(ctx)
	updates := map[string]any{"execution_token": "", "lease_expires_at": nil}
	if err != nil {
		updates["status"] = model.PixezMirrorStatusFailed
		updates["error_message"] = err.Error()
	}
	if e := db.DB(cleanupCtx).Table(table).Where(column+" = ? AND execution_token = ?", id, token).Updates(updates).Error; e != nil {
		logger.ErrorF(cleanupCtx, "[PixEz] release mirror lease: %v", e)
		if err == nil {
			err = e
		}
	}
	return err
}

func protectedMirrorUpdate(ctx context.Context, target string, id int64, updates map[string]any) error {
	table, column := mirrorTable(target)
	q := db.DB(ctx).Table(table).Where(column+" = ?", id)
	if token, ok := ctx.Value(mirrorTokenKey{}).(string); ok {
		q = q.Where("execution_token = ?", token)
	}
	r := q.Updates(updates)
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected == 0 {
		return fmt.Errorf("mirror execution no longer owns work %s/%d", target, id)
	}
	return nil
}
