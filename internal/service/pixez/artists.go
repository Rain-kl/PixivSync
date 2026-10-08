// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package pixez

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/pkg/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const artistBatchSize = 100

// IndexIllust adds a creator and immutable illustration summary to the directory.
func IndexIllust(ctx context.Context, work Illust, mirror bool, runID string) error {
	if work.ID <= 0 || work.User.ID <= 0 || IsLimitUnknownIllust(work) {
		return nil
	}
	return indexArtistWork(ctx, work.User.ID, work.User.Name, work.User.ProfileImageUrls.Medium, model.PixezMirrorTargetIllust, work.ID, mustJSON(work), mirror, runID)
}

// IndexNovel adds a creator and immutable novel summary to the directory.
func IndexNovel(ctx context.Context, work Novel, mirror bool, runID string) error {
	if work.ID <= 0 || work.User.ID <= 0 || IsLimitUnknownNovel(BookmarkNovel{ImageUrls: work.ImageUrls}) {
		return nil
	}
	return indexArtistWork(ctx, work.User.ID, work.User.Name, work.User.ProfileImageUrls.Medium, model.PixezMirrorTargetNovel, work.ID, mustJSON(work), mirror, runID)
}

func indexArtistWork(ctx context.Context, artistID int64, name, avatar, target string, id int64, summary string, mirror bool, runID string) error {
	now := time.Now()
	return db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		artist := model.PixezArtist{ArtistID: artistID, Name: name, AvatarURL: avatar}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&artist).Error; err != nil {
			return err
		}
		profile := map[string]any{}
		if name != "" {
			profile["name"] = name
		}
		if avatar != "" {
			profile["avatar_url"] = avatar
		}
		if len(profile) > 0 && runID != "" {
			if err := tx.Model(&model.PixezArtist{}).Where("artist_id = ?", artistID).Updates(profile).Error; err != nil {
				return err
			}
		}
		work := model.PixezArtistWork{ArtistID: artistID, TargetType: target, TargetID: id, FirstSummaryJSON: summary, DiscoveredFromMirror: mirror, DiscoveredFromScan: runID != ""}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&work).Error; err != nil {
			return err
		}
		updates := map[string]any{}
		if mirror {
			updates["discovered_from_mirror"] = true
		}
		if runID != "" {
			updates["discovered_from_scan"] = true
			updates["last_seen_run_id"] = runID
			updates["last_seen_at"] = now
			updates["not_returned"] = false
		}
		if len(updates) == 0 {
			return nil
		}
		return tx.Model(&model.PixezArtistWork{}).Where("target_type = ? AND target_id = ?", target, id).Updates(updates).Error
	})
}

// BackfillArtistMirrors extracts creators in bounded batches without changing mirror snapshots.
func BackfillArtistMirrors(ctx context.Context) error {
	if err := backfillIllustArtists(ctx); err != nil {
		return err
	}
	return backfillNovelArtists(ctx)
}

//nolint:dupl // Each archive type needs its own typed snapshot validation.
func backfillIllustArtists(ctx context.Context) error {
	var cursor int64
	for {
		var rows []model.PixezMirrorIllust
		if err := db.DB(ctx).Where("illust_id > ? AND detail_json <> '' AND NOT EXISTS (SELECT 1 FROM pixez_artist_works w WHERE w.target_type = 'illust' AND w.target_id = mirror_illust.illust_id AND w.discovered_from_mirror = true)", cursor).Order("illust_id").Limit(artistBatchSize).Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		for _, row := range rows {
			cursor = row.IllustID
			var detail IllustDetail
			if err := json.Unmarshal([]byte(row.DetailJSON), &detail); err != nil {
				logger.WarnF(ctx, "[PixEz] skip invalid illustration snapshot id=%d", row.IllustID)
				continue
			}
			if detail.Illust.ID != row.IllustID {
				continue
			}
			if err := IndexIllust(ctx, detail.Illust, true, ""); err != nil {
				return err
			}
		}
	}
}

//nolint:dupl // Each archive type needs its own typed snapshot validation.
func backfillNovelArtists(ctx context.Context) error {
	var cursor int64
	for {
		var rows []model.PixezMirrorNovel
		if err := db.DB(ctx).Where("novel_id > ? AND detail_json <> '' AND NOT EXISTS (SELECT 1 FROM pixez_artist_works w WHERE w.target_type = 'novel' AND w.target_id = mirror_novel.novel_id AND w.discovered_from_mirror = true)", cursor).Order("novel_id").Limit(artistBatchSize).Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		for _, row := range rows {
			cursor = row.NovelID
			var detail NovelDetail
			if err := json.Unmarshal([]byte(row.DetailJSON), &detail); err != nil {
				logger.WarnF(ctx, "[PixEz] skip invalid novel snapshot id=%d", row.NovelID)
				continue
			}
			if detail.Novel.ID != row.NovelID {
				continue
			}
			if err := IndexNovel(ctx, detail.Novel, true, ""); err != nil {
				return err
			}
		}
	}
}

// ArtistItem is a directory entry with category-specific subscription state.
type ArtistItem struct {
	model.PixezArtist
	Subscription model.PixezArtistSubscription `json:"subscription"`
	WorkCount    int64                         `json:"work_count"`
	BackupCount  int64                         `json:"backup_count"`
}

// ArtistQuery describes directory pagination and filtering.
type ArtistQuery struct {
	TargetType   string `form:"type"`
	Q            string `form:"q"`
	Subscription string `form:"subscription"`
	Page         int    `form:"page"`
	PageSize     int    `form:"page_size"`
	Status       string `form:"status"`
}

// ListArtists lists discovered or previously subscribed creators of one category.
func ListArtists(ctx context.Context, req ArtistQuery) ([]ArtistItem, int64, error) {
	q := db.DB(ctx).Model(&model.PixezArtist{}).Where("EXISTS (SELECT 1 FROM pixez_artist_works w WHERE w.artist_id = pixez_artists.artist_id AND w.target_type = ?) OR EXISTS (SELECT 1 FROM pixez_artist_subscriptions s WHERE s.artist_id = pixez_artists.artist_id AND s.target_type = ?)", req.TargetType, req.TargetType)
	if req.Q != "" {
		q = q.Where("name LIKE ? OR CAST(artist_id AS TEXT) LIKE ?", "%"+req.Q+"%", "%"+req.Q+"%")
	}
	if req.Subscription == "enabled" {
		q = q.Where("EXISTS (SELECT 1 FROM pixez_artist_subscriptions s WHERE s.artist_id = pixez_artists.artist_id AND s.target_type = ? AND s.enabled = ?)", req.TargetType, true)
	}
	if req.Subscription == "disabled" {
		q = q.Where("NOT EXISTS (SELECT 1 FROM pixez_artist_subscriptions s WHERE s.artist_id = pixez_artists.artist_id AND s.target_type = ? AND s.enabled = ?)", req.TargetType, true)
	}
	if req.Subscription == "error" {
		q = q.Where("EXISTS (SELECT 1 FROM pixez_artist_subscriptions s WHERE s.artist_id = pixez_artists.artist_id AND s.target_type = ? AND s.last_error <> '')", req.TargetType)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var artists []model.PixezArtist
	if err := q.Order("artist_id").Offset((req.Page - 1) * req.PageSize).Limit(req.PageSize).Find(&artists).Error; err != nil {
		return nil, 0, err
	}
	items := make([]ArtistItem, 0, len(artists))
	for _, artist := range artists {
		item := ArtistItem{PixezArtist: artist}
		subs, err := ArtistSubscriptions(ctx, artist.ArtistID)
		if err != nil {
			return nil, 0, err
		}
		for _, sub := range subs {
			if sub.TargetType == req.TargetType {
				item.Subscription = sub
			}
		}
		counts := artistWorkQuery(ctx, artist.ArtistID, req.TargetType)
		if err := counts.Count(&item.WorkCount).Error; err != nil {
			return nil, 0, err
		}
		if err := counts.Where(completeWorkSQL(req.TargetType)).Count(&item.BackupCount).Error; err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, nil
}

// ArtistSubscriptions returns both category switches without creating empty records.
func ArtistSubscriptions(ctx context.Context, id int64) ([]model.PixezArtistSubscription, error) {
	subs := make([]model.PixezArtistSubscription, 0)
	err := db.DB(ctx).Where("artist_id = ?", id).Find(&subs).Error
	return subs, err
}

func artistWorkQuery(ctx context.Context, id int64, target string) *gorm.DB {
	q := db.DB(ctx).Table("pixez_artist_works w").Where("w.artist_id = ? AND w.target_type = ?", id, target)
	if target == model.PixezMirrorTargetIllust {
		return q.Joins("LEFT JOIN mirror_illust m ON m.illust_id = w.target_id")
	}
	return q.Joins("LEFT JOIN mirror_novel m ON m.novel_id = w.target_id")
}
func completeWorkSQL(target string) string {
	if target == model.PixezMirrorTargetIllust {
		return "m.detail_json <> '' AND m.total_count > 0 AND m.success_count >= m.total_count AND m.failed_count = 0"
	}
	return "m.detail_json <> '' AND m.text_json <> '' AND m.success_count > 0"
}

// ArtistWorkItem exposes immutable summary metadata and current backup state.
type ArtistWorkItem struct {
	model.PixezArtistWork
	Title        string `json:"title"`
	PreviewURL   string `json:"preview_url"`
	BackupStatus string `json:"backup_status"`
}

// ListArtistWorks returns the union of archived and online-discovered work IDs.
func ListArtistWorks(ctx context.Context, id int64, req ArtistQuery) ([]ArtistWorkItem, int64, error) {
	q := artistWorkQuery(ctx, id, req.TargetType)
	switch req.Status {
	case "complete":
		q = q.Where(completeWorkSQL(req.TargetType))
	case "none":
		q = q.Where("m.task_id IS NULL")
	case "failed":
		q = q.Where("m.status = ?", model.PixezMirrorStatusFailed)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.PixezArtistWork
	if err := q.Select("w.*").Order("w.target_id DESC").Offset((req.Page - 1) * req.PageSize).Limit(req.PageSize).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]ArtistWorkItem, 0, len(rows))
	for _, row := range rows {
		item, err := artistWorkItem(ctx, row)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, nil
}
func artistWorkItem(ctx context.Context, row model.PixezArtistWork) (ArtistWorkItem, error) {
	item := ArtistWorkItem{PixezArtistWork: row, BackupStatus: "none"}
	var summary struct {
		Title     string `json:"title"`
		ImageURLs struct {
			Medium string `json:"medium"`
		} `json:"image_urls"`
	}
	if err := json.Unmarshal([]byte(row.FirstSummaryJSON), &summary); err != nil {
		return item, fmt.Errorf("decode indexed work: %w", err)
	}
	item.Title = summary.Title
	item.PreviewURL = summary.ImageURLs.Medium
	table, column := mirrorTable(row.TargetType)
	var backup struct {
		Status       string
		TotalCount   int
		SuccessCount int
		FailedCount  int
	}
	r := db.DB(ctx).Table(table).Where(column+" = ?", row.TargetID).Scan(&backup)
	if r.Error != nil {
		return item, r.Error
	}
	if r.RowsAffected == 0 {
		return item, nil
	}
	item.BackupStatus = backup.Status
	if backup.SuccessCount > 0 && backup.SuccessCount < backup.TotalCount {
		item.BackupStatus = "partial"
	}
	needed, err := MirrorNeedsWork(ctx, row.TargetType, row.TargetID)
	if err != nil {
		return item, err
	}
	if !needed && backup.Status == model.PixezMirrorStatusSuccess {
		item.BackupStatus = "complete"
	}
	return item, nil
}

// GetArtist loads one creator from the shared directory.
func GetArtist(ctx context.Context, id int64) (model.PixezArtist, error) {
	var artist model.PixezArtist
	err := db.DB(ctx).Where("artist_id = ?", id).First(&artist).Error
	return artist, err
}

// ListArtistRuns returns bounded scan history for a creator.
func ListArtistRuns(ctx context.Context, id int64) ([]model.PixezArtistSyncRun, error) {
	runs := make([]model.PixezArtistSyncRun, 0)
	err := db.DB(ctx).Where("artist_id = ?", id).Order("created_at DESC").Limit(artistBatchSize).Find(&runs).Error
	return runs, err
}

// EnsureArtistDirectory lazily initializes historical creators until the scheduler runs.
func EnsureArtistDirectory(ctx context.Context) error {
	var n int64
	if err := db.DB(ctx).Model(&model.PixezArtist{}).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	return BackfillArtistMirrors(ctx)
}
