// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package pixez

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/task"
	"github.com/Rain-kl/Wavelet/pkg/logger"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const artistScanFailed = "failed"

const artistScanLease = 15 * time.Minute
const artistCheckInterval = 24 * time.Hour

var artistIllustCategories = []string{"illust", "manga"}

// SetArtistSubscription changes one switch and reports a newly enabled subscription.
func SetArtistSubscription(ctx context.Context, id int64, target string, enabled bool) (bool, error) {
	changed := false
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		sub := model.PixezArtistSubscription{ArtistID: id, TargetType: target}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&sub).Error; err != nil {
			return err
		}
		var previous model.PixezArtistSubscription
		if err := tx.Where("artist_id = ? AND target_type = ?", id, target).First(&previous).Error; err != nil {
			return err
		}
		q := tx.Model(&model.PixezArtistSubscription{}).Where("artist_id = ? AND target_type = ? AND enabled <> ?", id, target, enabled)
		updates := map[string]any{"enabled": enabled, "revision": gorm.Expr("revision + 1"), "active_run_id": "", "lease_expires_at": nil}
		if enabled {
			updates["next_due_at"] = time.Now()
			updates["last_error"] = ""
		}
		r := q.Updates(updates)
		changed = r.RowsAffected > 0
		if r.Error != nil {
			return r.Error
		}
		if changed && previous.ActiveRunID != "" {
			return tx.Model(&model.PixezArtistSyncRun{}).Where("id = ? AND status NOT IN (?, ?)", previous.ActiveRunID, "success", "cancelled").Updates(map[string]any{"status": "cancelled", "finished_at": time.Now()}).Error
		}
		return nil
	})
	return changed && enabled, err
}

// CreateArtistRun reserves a scan before enqueue, or returns an existing active scan.
func CreateArtistRun(ctx context.Context, id int64, target string, download bool, trigger string) (model.PixezArtistSyncRun, bool, error) {
	var artist model.PixezArtist
	if err := db.DB(ctx).Where("artist_id = ?", id).First(&artist).Error; err != nil {
		return model.PixezArtistSyncRun{}, false, err
	}
	var run model.PixezArtistSyncRun
	created := false
	err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		sub := model.PixezArtistSubscription{ArtistID: id, TargetType: target}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&sub).Error; err != nil {
			return err
		}
		if err := tx.Where("artist_id = ? AND target_type = ?", id, target).First(&sub).Error; err != nil {
			return err
		}
		if download && !sub.Enabled {
			return errors.New("subscription is disabled")
		}
		now := time.Now()
		if sub.ActiveRunID != "" && sub.LeaseExpiresAt != nil && sub.LeaseExpiresAt.After(now) {
			return tx.Where("id = ?", sub.ActiveRunID).First(&run).Error
		}
		run = model.PixezArtistSyncRun{ID: uuid.NewString(), ArtistID: id, TargetType: target, Revision: sub.Revision, Download: download, TriggeredBy: trigger, Status: "pending", NextURL: artistInitialURL(id, target, 0)}
		r := tx.Model(&model.PixezArtistSubscription{}).Where("id = ? AND revision = ? AND (lease_expires_at IS NULL OR lease_expires_at < ?)", sub.ID, sub.Revision, now).
			Updates(map[string]any{"active_run_id": run.ID, "lease_expires_at": now.Add(artistScanLease), "last_attempt_at": now})
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return errors.New("artist scan reservation changed; retry")
		}
		if sub.ActiveRunID != "" {
			if err := tx.Model(&model.PixezArtistSyncRun{}).Where("id = ? AND status NOT IN (?, ?)", sub.ActiveRunID, "success", "cancelled").Updates(map[string]any{"status": artistScanFailed, "finished_at": now, "error_message": "扫描租约过期，已安排重新扫描"}).Error; err != nil {
				return err
			}
		}
		created = true
		return tx.Create(&run).Error
	})
	return run, created, err
}

func artistInitialURL(id int64, target string, phase int) string {
	values := url.Values{"user_id": {strconv.FormatInt(id, 10)}, "filter": {"for_android"}}
	path := "/v1/user/novels"
	if target == model.PixezMirrorTargetIllust {
		path = "/v1/user/illusts"
		values.Set("type", artistIllustCategories[phase])
	}
	return "https://" + pixivAPIHost + path + "?" + values.Encode()
}
func validateArtistURL(raw string, run model.PixezArtistSyncRun) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	expected, _ := url.Parse(artistInitialURL(run.ArtistID, run.TargetType, run.Phase))
	if u.Scheme != "https" || u.Host != pixivAPIHost || u.User != nil || u.Fragment != "" || u.Path != expected.Path ||
		len(u.Query()["user_id"]) != 1 || len(u.Query()["type"]) > 1 || u.Query().Get("user_id") != expected.Query().Get("user_id") || u.Query().Get("type") != expected.Query().Get("type") {
		return errors.New("invalid pixiv creator pagination URL")
	}
	return nil
}

// ArtistMirrorEnqueuer uses the existing application task dispatcher.
type ArtistMirrorEnqueuer func(context.Context, string, int64) (bool, error)

// ScanArtistRun resumes a page cursor while preserving all existing backups.
func ScanArtistRun(ctx context.Context, client *Client, runID string, enqueue ArtistMirrorEnqueuer) error {
	if client == nil {
		client = DefaultClient
	}
	var run model.PixezArtistSyncRun
	if err := db.DB(ctx).Where("id = ?", runID).First(&run).Error; err != nil {
		return err
	}
	claimed, err := claimArtistExecution(ctx, &run)
	if err != nil || !claimed {
		return err
	}
	defer releaseArtistExecution(context.WithoutCancel(ctx), run)
	return executeArtistPages(ctx, client, &run, enqueue)
}

func claimArtistExecution(ctx context.Context, run *model.PixezArtistSyncRun) (bool, error) {
	if run.Status == "success" || run.Status == "cancelled" || run.Status == artistScanFailed {
		return false, nil
	}
	if run.Phase < 0 || (run.TargetType == model.PixezMirrorTargetIllust && run.Phase >= len(artistIllustCategories)) {
		return false, errors.New("invalid scan phase")
	}
	run.ExecutionToken = uuid.NewString()
	now := time.Now()
	r := db.DB(ctx).Model(&model.PixezArtistSyncRun{}).Where("id = ? AND status IN (?, ?, ?) AND (lease_expires_at IS NULL OR lease_expires_at < ?)", run.ID, "pending", "running", "retrying", now).Updates(map[string]any{"execution_token": run.ExecutionToken, "lease_expires_at": now.Add(artistScanLease)})
	return r.RowsAffected > 0, r.Error
}
func releaseArtistExecution(ctx context.Context, run model.PixezArtistSyncRun) {
	if err := artistExecutionQuery(db.DB(ctx), run).Updates(map[string]any{"execution_token": "", "lease_expires_at": nil}).Error; err != nil {
		logger.ErrorF(ctx, "[PixEz] release scan execution: %v", err)
	}
}
func artistExecutionQuery(tx *gorm.DB, run model.PixezArtistSyncRun) *gorm.DB {
	return tx.Model(&model.PixezArtistSyncRun{}).Where("id = ? AND execution_token = ?", run.ID, run.ExecutionToken)
}
func executeArtistPages(ctx context.Context, client *Client, run *model.PixezArtistSyncRun, enqueue ArtistMirrorEnqueuer) error {
	user, err := artistScanAccount(ctx, run)
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for run.NextURL != "" {
		active, e := renewArtistRun(ctx, *run)
		if e != nil {
			return e
		}
		if !active {
			return cancelArtistRun(ctx, *run)
		}
		if seen[run.NextURL] {
			return errors.New("pixiv repeated a pagination cursor")
		}
		seen[run.NextURL] = true
		if err = validateArtistURL(run.NextURL, *run); err != nil {
			return err
		}
		next, e := scanArtistPage(ctx, client, user, run, enqueue)
		var cancelled artistScanCancelledError
		if errors.As(e, &cancelled) {
			return cancelArtistRun(ctx, *run)
		}
		if e != nil {
			return e
		}
		if next != "" {
			if err = validateArtistURL(next, *run); err != nil {
				return err
			}
		}
		if next == "" && run.TargetType == model.PixezMirrorTargetIllust && run.Phase+1 < len(artistIllustCategories) {
			run.Phase++
			next = artistInitialURL(run.ArtistID, run.TargetType, run.Phase)
		}
		run.NextURL = next
		var count int64
		if err = db.DB(ctx).Model(&model.PixezArtistWork{}).Where("last_seen_run_id = ?", run.ID).Count(&count).Error; err != nil {
			return err
		}
		run.DiscoveredCount = int(count)
		if err = saveArtistProgress(ctx, *run); err != nil {
			return err
		}
		task.AppendLog(ctx, "作者目录扫描 artist=%d type=%s discovered=%d queued=%d", run.ArtistID, run.TargetType, run.DiscoveredCount, run.QueuedCount)
		if next != "" {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Second):
			}
		}
	}
	return finishArtistRun(ctx, *run)
}

func artistScanAccount(ctx context.Context, run *model.PixezArtistSyncRun) (model.PixezPixivUser, error) {
	if run.AccountID == "" {
		user, err := latestMirrorUser(ctx)
		if err != nil {
			return user, err
		}
		run.AccountID = user.PixivUserID
		err = artistExecutionQuery(db.DB(ctx), *run).Updates(map[string]any{"account_id": run.AccountID, "status": "running"}).Error
		return user, err
	}
	var user model.PixezPixivUser
	err := db.DB(ctx).Where("pixiv_user_id = ?", run.AccountID).First(&user).Error
	return user, err
}
func renewArtistRun(ctx context.Context, run model.PixezArtistSyncRun) (bool, error) {
	lease := artistExecutionQuery(db.DB(ctx), run).Where("status NOT IN (?, ?)", "cancelled", artistScanFailed).Update("lease_expires_at", time.Now().Add(artistScanLease))
	if lease.Error != nil || lease.RowsAffected == 0 {
		return false, lease.Error
	}
	q := db.DB(ctx).Model(&model.PixezArtistSubscription{}).Where("artist_id = ? AND target_type = ? AND active_run_id = ? AND revision = ?", run.ArtistID, run.TargetType, run.ID, run.Revision)
	if run.Download {
		q = q.Where("enabled = ?", true)
	}
	r := q.Update("lease_expires_at", time.Now().Add(artistScanLease))
	return r.RowsAffected > 0, r.Error
}
func scanArtistPage(ctx context.Context, client *Client, user model.PixezPixivUser, run *model.PixezArtistSyncRun, enqueue ArtistMirrorEnqueuer) (string, error) {
	if run.TargetType == model.PixezMirrorTargetIllust {
		return scanArtistIllustPage(ctx, client, user, run, enqueue)
	}
	return scanArtistNovelPage(ctx, client, user, run, enqueue)
}
func scanArtistIllustPage(ctx context.Context, client *Client, user model.PixezPixivUser, run *model.PixezArtistSyncRun, enqueue ArtistMirrorEnqueuer) (string, error) {
	var page BookmarkIllustResponse
	if _, err := client.getJSONWithAuth(ctx, user, run.NextURL, &page); err != nil {
		return "", err
	}
	if page.Illusts == nil {
		return "", errors.New("missing creator illustrations response")
	}
	for _, work := range page.Illusts {
		if work.User.ID != run.ArtistID {
			return "", errors.New("pixiv returned another creator's work")
		}
		if IsLimitUnknownIllust(work) {
			continue
		}
		if err := IndexIllust(ctx, work, false, run.ID); err != nil {
			return "", err
		}
		if err := enqueueArtistWork(ctx, run, work.ID, enqueue); err != nil {
			return "", err
		}
	}
	return page.NextURL, nil
}
func scanArtistNovelPage(ctx context.Context, client *Client, user model.PixezPixivUser, run *model.PixezArtistSyncRun, enqueue ArtistMirrorEnqueuer) (string, error) {
	var page struct {
		Novels  []Novel `json:"novels"`
		NextURL string  `json:"next_url"`
	}
	if _, err := client.getJSONWithAuth(ctx, user, run.NextURL, &page); err != nil {
		return "", err
	}
	if page.Novels == nil {
		return "", errors.New("missing creator novels response")
	}
	for _, work := range page.Novels {
		if IsLimitUnknownNovel(BookmarkNovel{ImageUrls: work.ImageUrls}) {
			continue
		}
		if work.User.ID != run.ArtistID {
			return "", errors.New("pixiv returned another creator's work")
		}
		if err := IndexNovel(ctx, work, false, run.ID); err != nil {
			return "", err
		}
		if err := enqueueArtistWork(ctx, run, work.ID, enqueue); err != nil {
			return "", err
		}
	}
	return page.NextURL, nil
}

type artistScanCancelledError struct{}

func (artistScanCancelledError) Error() string { return "subscription changed during scan" }

func enqueueArtistWork(ctx context.Context, run *model.PixezArtistSyncRun, id int64, enqueue ArtistMirrorEnqueuer) error {
	if !run.Download {
		return nil
	}
	active, err := renewArtistRun(ctx, *run)
	if err != nil {
		return err
	}
	if !active {
		return artistScanCancelledError{}
	}
	needs, err := MirrorNeedsWork(ctx, run.TargetType, id)
	if err != nil {
		return err
	}
	if !needs {
		run.SkippedCount++
		return nil
	}
	queued, err := enqueue(ctx, run.TargetType, id)
	if queued {
		run.QueuedCount++
	}
	return err
}
func saveArtistProgress(ctx context.Context, run model.PixezArtistSyncRun) error {
	return artistExecutionQuery(db.DB(ctx), run).Where("status NOT IN (?, ?)", "cancelled", artistScanFailed).Updates(map[string]any{"status": "running", "phase": run.Phase, "next_url": run.NextURL, "discovered_count": run.DiscoveredCount, "queued_count": run.QueuedCount, "skipped_count": run.SkippedCount}).Error
}
func cancelArtistRun(ctx context.Context, run model.PixezArtistSyncRun) error {
	return artistExecutionQuery(db.DB(ctx), run).Updates(map[string]any{"status": "cancelled", "finished_at": time.Now()}).Error
}
func finishArtistRun(ctx context.Context, run model.PixezArtistSyncRun) error {
	return db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		// Claim the final write before changing the subscription or missing-work flags.
		// The row update also prevents an expired worker from finishing a newer execution.
		completion := artistExecutionQuery(tx, run).Where("status NOT IN (?, ?)", "cancelled", artistScanFailed).
			Updates(map[string]any{"status": "success", "finished_at": now, "error_message": ""})
		if completion.Error != nil || completion.RowsAffected == 0 {
			return completion.Error
		}
		q := tx.Model(&model.PixezArtistSubscription{}).Where("artist_id = ? AND target_type = ? AND active_run_id = ? AND revision = ?", run.ArtistID, run.TargetType, run.ID, run.Revision)
		if run.Download {
			q = q.Where("enabled = ?", true)
		}
		updates := map[string]any{"active_run_id": "", "lease_expires_at": nil, "last_error": ""}
		if run.Download {
			updates["last_success_at"] = now
			updates["next_due_at"] = now.Add(artistCheckInterval)
		}
		r := q.Updates(updates)
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return artistExecutionQuery(tx, run).Updates(map[string]any{"status": "cancelled", "finished_at": now}).Error
		}
		if err := tx.Model(&model.PixezArtistWork{}).Where("artist_id = ? AND target_type = ? AND last_seen_run_id <> ?", run.ArtistID, run.TargetType, run.ID).Update("not_returned", true).Error; err != nil {
			return err
		}
		return nil
	})
}

// FailArtistRun releases only the reservation owned by this run, with bounded retries.
func FailArtistRun(ctx context.Context, id string, final bool) error {
	var run model.PixezArtistSyncRun
	if err := db.DB(ctx).Where("id = ?", id).First(&run).Error; err != nil {
		return err
	}
	if run.Status == "success" || run.Status == "cancelled" || (run.LeaseExpiresAt != nil && run.LeaseExpiresAt.After(time.Now())) {
		return nil
	}
	status := "retrying"
	if final {
		status = artistScanFailed
	}
	if err := db.DB(ctx).Model(&model.PixezArtistSyncRun{}).Where("id = ? AND status NOT IN (?, ?)", id, "success", "cancelled").Updates(map[string]any{"status": status, "error_message": "扫描失败，请查看任务日志"}).Error; err != nil {
		return err
	}
	updates := map[string]any{"last_error": "扫描失败，请查看任务日志"}
	if final {
		updates["active_run_id"] = ""
		updates["lease_expires_at"] = nil
		updates["next_due_at"] = time.Now().Add(artistCheckInterval)
	} else {
		updates["lease_expires_at"] = time.Now().Add(artistScanLease)
	}
	return db.DB(ctx).Model(&model.PixezArtistSubscription{}).Where("active_run_id = ?", id).Updates(updates).Error
}

// FailArtistDispatch leaves a persisted subscription due for the next scheduler check.
func FailArtistDispatch(ctx context.Context, run model.PixezArtistSyncRun) error {
	if err := FailArtistRun(ctx, run.ID, true); err != nil {
		return err
	}
	return db.DB(ctx).Model(&model.PixezArtistSubscription{}).
		Where("artist_id = ? AND target_type = ? AND active_run_id = ? AND enabled = ?", run.ArtistID, run.TargetType, "", true).
		Update("next_due_at", time.Now()).Error
}

// DueArtistSubscriptions finds bounded batches of scans due for daily checking.
func DueArtistSubscriptions(ctx context.Context) ([]model.PixezArtistSubscription, error) {
	var subs []model.PixezArtistSubscription
	now := time.Now()
	err := db.DB(ctx).Where("enabled = ? AND (next_due_at IS NULL OR next_due_at <= ?) AND (lease_expires_at IS NULL OR lease_expires_at <= ?)", true, now, now).Order("next_due_at").Limit(artistBatchSize).Find(&subs).Error
	return subs, err
}

// GetArtistRun returns safe scan status; credentials are never included in the model JSON.
func GetArtistRun(ctx context.Context, id string) (model.PixezArtistSyncRun, error) {
	var run model.PixezArtistSyncRun
	err := db.DB(ctx).Where("id = ?", id).First(&run).Error
	return run, err
}
