// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package pixez

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/db/idgen"
	"github.com/Rain-kl/Wavelet/internal/model"
	"gorm.io/gorm"
)

type bookmarkSaveStatus int

const (
	bookmarkSaveSkipped bookmarkSaveStatus = iota
	bookmarkSaveCreated
	bookmarkSaveUpdated
	bookmarkSaveRemoved
)

// ExportIllustBookmarks exports Pixiv illustration bookmarks for one or all saved Pixiv users.
func ExportIllustBookmarks(ctx context.Context, client *Client, pixivUserID string) (ExportSummary, error) {
	return exportBookmarks(ctx, client, pixivUserID, model.PixezMirrorTargetIllust, exportIllustRestrict)
}

// ExportNovelBookmarks exports Pixiv novel bookmarks for one or all saved Pixiv users.
func ExportNovelBookmarks(ctx context.Context, client *Client, pixivUserID string) (ExportSummary, error) {
	return exportBookmarks(ctx, client, pixivUserID, model.PixezMirrorTargetNovel, exportNovelRestrict)
}

func exportBookmarks(
	ctx context.Context,
	client *Client,
	pixivUserID string,
	targetType string,
	exportRestrict func(context.Context, *Client, model.PixezPixivUser, string) (ExportSummary, error),
) (ExportSummary, error) {
	if client == nil {
		client = DefaultClient
	}
	users, err := exportUsers(ctx, pixivUserID)
	if err != nil {
		return ExportSummary{TargetType: targetType}, err
	}

	summary := ExportSummary{TargetType: targetType, UserCount: len(users)}
	for _, user := range users {
		for _, restrict := range bookmarkRestricts {
			runSummary, err := exportRestrict(ctx, client, user, restrict)
			summary.RunCount++
			summary.TotalCount += runSummary.TotalCount
			summary.NewCount += runSummary.NewCount
			summary.UpdatedCount += runSummary.UpdatedCount
			summary.RemovedCount += runSummary.RemovedCount
			if err != nil {
				return summary, err
			}
		}
	}
	return summary, nil
}

func exportUsers(ctx context.Context, pixivUserID string) ([]model.PixezPixivUser, error) {
	var users []model.PixezPixivUser
	query := db.DB(ctx).Order("updated_at desc")
	if pixivUserID != "" {
		query = query.Where("pixiv_user_id = ?", pixivUserID)
	}
	if err := query.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func exportIllustRestrict(ctx context.Context, client *Client, user model.PixezPixivUser, restrict string) (ExportSummary, error) {
	run := newBookmarkRun(model.PixezMirrorTargetIllust, user.PixivUserID, restrict)
	if err := db.DB(ctx).Create(&run).Error; err != nil {
		return ExportSummary{}, err
	}

	summary := ExportSummary{TargetType: model.PixezMirrorTargetIllust}
	reqURL := InitialBookmarkIllustURL(user.PixivUserID, restrict)
	for reqURL != "" {
		raw, payload, err := client.GetBookmarkIllusts(ctx, user, reqURL)
		if err != nil {
			markBookmarkRunFailed(ctx, run.ID, reqURL, err)
			return summary, fmt.Errorf("export illust bookmarks user=%s restrict=%s: %w", user.PixivUserID, restrict, err)
		}
		_ = raw

		summary.TotalCount += len(payload.Illusts)
		for _, illust := range payload.Illusts {
			status, err := upsertBookmarkIllust(ctx, user.PixivUserID, restrict, run.ID, illust)
			if err != nil {
				markBookmarkRunFailed(ctx, run.ID, reqURL, err)
				return summary, err
			}
			addBookmarkStatus(&summary, status)
		}

		if err := db.DB(ctx).Model(&model.PixezBookmarkExportRun{}).
			Where("id = ?", run.ID).
			Updates(map[string]any{
				"total_count":      summary.TotalCount,
				"new_count":        summary.NewCount,
				"updated_count":    summary.UpdatedCount,
				"removed_count":    summary.RemovedCount,
				"next_url":         payload.NextURL,
				"last_request_url": reqURL,
				"updated_at":       time.Now(),
			}).Error; err != nil {
			return summary, err
		}
		reqURL = payload.NextURL
	}

	removed, err := markMissingIllustBookmarksRemoved(ctx, user.PixivUserID, restrict, run.ID)
	if err != nil {
		markBookmarkRunFailed(ctx, run.ID, "", err)
		return summary, err
	}
	summary.RemovedCount += removed
	if err := markBookmarkRunSuccess(ctx, run.ID, summary); err != nil {
		return summary, err
	}
	return summary, nil
}

func exportNovelRestrict(ctx context.Context, client *Client, user model.PixezPixivUser, restrict string) (ExportSummary, error) {
	run := newBookmarkRun(model.PixezMirrorTargetNovel, user.PixivUserID, restrict)
	if err := db.DB(ctx).Create(&run).Error; err != nil {
		return ExportSummary{}, err
	}

	summary := ExportSummary{TargetType: model.PixezMirrorTargetNovel}
	reqURL := InitialBookmarkNovelURL(user.PixivUserID, restrict)
	for reqURL != "" {
		_, payload, err := client.GetBookmarkNovels(ctx, user, reqURL)
		if err != nil {
			markBookmarkRunFailed(ctx, run.ID, reqURL, err)
			return summary, fmt.Errorf("export novel bookmarks user=%s restrict=%s: %w", user.PixivUserID, restrict, err)
		}

		summary.TotalCount += len(payload.Novels)
		for _, novel := range payload.Novels {
			status, err := upsertBookmarkNovel(ctx, user.PixivUserID, restrict, run.ID, novel)
			if err != nil {
				markBookmarkRunFailed(ctx, run.ID, reqURL, err)
				return summary, err
			}
			addBookmarkStatus(&summary, status)
		}

		if err := db.DB(ctx).Model(&model.PixezBookmarkExportRun{}).
			Where("id = ?", run.ID).
			Updates(map[string]any{
				"total_count":      summary.TotalCount,
				"new_count":        summary.NewCount,
				"updated_count":    summary.UpdatedCount,
				"removed_count":    summary.RemovedCount,
				"next_url":         payload.NextURL,
				"last_request_url": reqURL,
				"updated_at":       time.Now(),
			}).Error; err != nil {
			return summary, err
		}
		reqURL = payload.NextURL
	}

	removed, err := markMissingNovelBookmarksRemoved(ctx, user.PixivUserID, restrict, run.ID)
	if err != nil {
		markBookmarkRunFailed(ctx, run.ID, "", err)
		return summary, err
	}
	summary.RemovedCount += removed
	if err := markBookmarkRunSuccess(ctx, run.ID, summary); err != nil {
		return summary, err
	}
	return summary, nil
}

func newBookmarkRun(targetType string, pixivUserID string, restrict string) model.PixezBookmarkExportRun {
	now := time.Now()
	return model.PixezBookmarkExportRun{
		ID:          fmt.Sprintf("pixez_%s_%d", targetType, idgen.NextUint64ID()),
		TargetType:  targetType,
		PixivUserID: pixivUserID,
		Restrict:    restrict,
		Status:      model.PixezBookmarkExportStatusRunning,
		StartedAt:   now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

//nolint:dupl // Illust and novel bookmark flows intentionally stay parallel because Pixiv payload schemas differ.
func upsertBookmarkIllust(ctx context.Context, pixivUserID string, restrict string, runID string, illust Illust) (bookmarkSaveStatus, error) {
	if illust.ID <= 0 {
		return bookmarkSaveSkipped, fmt.Errorf("bookmark export received illust without id")
	}
	now := time.Now()
	illustJSON := mustJSON(illust)
	if IsLimitUnknownIllust(illust) {
		return markBookmarkIllustRemoved(ctx, pixivUserID, restrict, illust.ID, runID, illustJSON, now)
	}

	var existing model.PixezBookmarkIllust
	err := db.DB(ctx).Where("pixiv_user_id = ? AND restrict = ? AND illust_id = ?", pixivUserID, restrict, illust.ID).First(&existing).Error
	if err == nil {
		updates := map[string]any{
			"last_export_run_id": runID,
			"last_seen_at":       now,
			"removed":            false,
			"removed_at":         nil,
			"updated_at":         now,
		}
		status := bookmarkSaveSkipped
		if existing.Removed || existing.IllustJSON != illustJSON {
			fillIllustBookmarkUpdates(updates, illust, illustJSON)
			status = bookmarkSaveUpdated
		}
		if err := db.DB(ctx).Model(&model.PixezBookmarkIllust{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
			return bookmarkSaveSkipped, err
		}
		return status, nil
	}
	if err != nil && !errorsIsRecordNotFound(err) {
		return bookmarkSaveSkipped, err
	}

	record := model.PixezBookmarkIllust{
		PixivUserID:      pixivUserID,
		Restrict:         restrict,
		IllustID:         illust.ID,
		LastExportRunID:  runID,
		LastSeenAt:       now,
		MirrorStatus:     model.PixezBookmarkMirrorNone,
		MirrorRetryCount: 0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	fillIllustBookmarkRecord(&record, illust, illustJSON)
	if err := db.DB(ctx).Create(&record).Error; err != nil {
		return bookmarkSaveSkipped, err
	}
	return bookmarkSaveCreated, nil
}

func markBookmarkIllustRemoved(ctx context.Context, pixivUserID string, restrict string, illustID int64, runID string, illustJSON string, now time.Time) (bookmarkSaveStatus, error) {
	var existing model.PixezBookmarkIllust
	err := db.DB(ctx).Where("pixiv_user_id = ? AND restrict = ? AND illust_id = ?", pixivUserID, restrict, illustID).First(&existing).Error
	if err == nil {
		updates := map[string]any{
			"illust_json":        illustJSON,
			"last_export_run_id": runID,
			"last_seen_at":       now,
			"removed":            true,
			"removed_at":         &now,
			"updated_at":         now,
		}
		if err := db.DB(ctx).Model(&model.PixezBookmarkIllust{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
			return bookmarkSaveSkipped, err
		}
		if existing.Removed {
			return bookmarkSaveSkipped, nil
		}
		return bookmarkSaveRemoved, nil
	}
	if err != nil && !errorsIsRecordNotFound(err) {
		return bookmarkSaveSkipped, err
	}
	record := model.PixezBookmarkIllust{
		PixivUserID:      pixivUserID,
		Restrict:         restrict,
		IllustID:         illustID,
		IllustJSON:       illustJSON,
		LastExportRunID:  runID,
		LastSeenAt:       now,
		Removed:          true,
		RemovedAt:        &now,
		MirrorStatus:     model.PixezBookmarkMirrorNone,
		MirrorRetryCount: 0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := db.DB(ctx).Create(&record).Error; err != nil {
		return bookmarkSaveSkipped, err
	}
	return bookmarkSaveRemoved, nil
}

//nolint:dupl // Illust and novel bookmark flows intentionally stay parallel because Pixiv payload schemas differ.
func upsertBookmarkNovel(ctx context.Context, pixivUserID string, restrict string, runID string, novel BookmarkNovel) (bookmarkSaveStatus, error) {
	if novel.ID <= 0 {
		return bookmarkSaveSkipped, fmt.Errorf("novel bookmark export received novel without id")
	}
	now := time.Now()
	novelJSON := mustJSON(novel)
	if IsLimitUnknownNovel(novel) {
		return markBookmarkNovelRemoved(ctx, pixivUserID, restrict, novel.ID, runID, novelJSON, now)
	}

	var existing model.PixezBookmarkNovel
	err := db.DB(ctx).Where("pixiv_user_id = ? AND restrict = ? AND novel_id = ?", pixivUserID, restrict, novel.ID).First(&existing).Error
	if err == nil {
		updates := map[string]any{
			"last_export_run_id": runID,
			"last_seen_at":       now,
			"removed":            false,
			"removed_at":         nil,
			"updated_at":         now,
		}
		status := bookmarkSaveSkipped
		if existing.Removed || existing.NovelJSON != novelJSON {
			fillNovelBookmarkUpdates(updates, novel, novelJSON)
			status = bookmarkSaveUpdated
		}
		if err := db.DB(ctx).Model(&model.PixezBookmarkNovel{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
			return bookmarkSaveSkipped, err
		}
		return status, nil
	}
	if err != nil && !errorsIsRecordNotFound(err) {
		return bookmarkSaveSkipped, err
	}

	record := model.PixezBookmarkNovel{
		PixivUserID:      pixivUserID,
		Restrict:         restrict,
		NovelID:          novel.ID,
		LastExportRunID:  runID,
		LastSeenAt:       now,
		MirrorStatus:     model.PixezBookmarkMirrorNone,
		MirrorRetryCount: 0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	fillNovelBookmarkRecord(&record, novel, novelJSON)
	if err := db.DB(ctx).Create(&record).Error; err != nil {
		return bookmarkSaveSkipped, err
	}
	return bookmarkSaveCreated, nil
}

func markBookmarkNovelRemoved(ctx context.Context, pixivUserID string, restrict string, novelID int64, runID string, novelJSON string, now time.Time) (bookmarkSaveStatus, error) {
	var existing model.PixezBookmarkNovel
	err := db.DB(ctx).Where("pixiv_user_id = ? AND restrict = ? AND novel_id = ?", pixivUserID, restrict, novelID).First(&existing).Error
	if err == nil {
		updates := map[string]any{
			"novel_json":         novelJSON,
			"last_export_run_id": runID,
			"last_seen_at":       now,
			"removed":            true,
			"removed_at":         &now,
			"updated_at":         now,
		}
		if err := db.DB(ctx).Model(&model.PixezBookmarkNovel{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
			return bookmarkSaveSkipped, err
		}
		if existing.Removed {
			return bookmarkSaveSkipped, nil
		}
		return bookmarkSaveRemoved, nil
	}
	if err != nil && !errorsIsRecordNotFound(err) {
		return bookmarkSaveSkipped, err
	}
	record := model.PixezBookmarkNovel{
		PixivUserID:      pixivUserID,
		Restrict:         restrict,
		NovelID:          novelID,
		NovelJSON:        novelJSON,
		LastExportRunID:  runID,
		LastSeenAt:       now,
		Removed:          true,
		RemovedAt:        &now,
		MirrorStatus:     model.PixezBookmarkMirrorNone,
		MirrorRetryCount: 0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := db.DB(ctx).Create(&record).Error; err != nil {
		return bookmarkSaveSkipped, err
	}
	return bookmarkSaveRemoved, nil
}

func fillIllustBookmarkRecord(record *model.PixezBookmarkIllust, illust Illust, illustJSON string) {
	record.Title = illust.Title
	record.Type = illust.Type
	record.UserID = illust.User.ID
	record.UserName = illust.User.Name
	record.PageCount = illust.PageCount
	record.Width = illust.Width
	record.Height = illust.Height
	record.SanityLevel = illust.SanityLevel
	record.XRestrict = illust.XRestrict
	record.TotalView = illust.TotalView
	record.TotalBookmarks = illust.TotalBookmarks
	record.Visible = illust.Visible
	record.IsMuted = illust.IsMuted
	record.IllustAIType = illust.IllustAIType
	record.IllustJSON = illustJSON
}

func fillIllustBookmarkUpdates(updates map[string]any, illust Illust, illustJSON string) {
	updates["title"] = illust.Title
	updates["type"] = illust.Type
	updates["user_id"] = illust.User.ID
	updates["user_name"] = illust.User.Name
	updates["page_count"] = illust.PageCount
	updates["width"] = illust.Width
	updates["height"] = illust.Height
	updates["sanity_level"] = illust.SanityLevel
	updates["x_restrict"] = illust.XRestrict
	updates["total_view"] = illust.TotalView
	updates["total_bookmarks"] = illust.TotalBookmarks
	updates["visible"] = illust.Visible
	updates["is_muted"] = illust.IsMuted
	updates["illust_ai_type"] = illust.IllustAIType
	updates["illust_json"] = illustJSON
}

func fillNovelBookmarkRecord(record *model.PixezBookmarkNovel, novel BookmarkNovel, novelJSON string) {
	record.Title = novel.Title
	record.Caption = novel.Caption
	record.UserID = novel.User.ID
	record.UserName = novel.User.Name
	record.TextLength = novel.TextLength
	record.XRestrict = novel.XRestrict
	record.TotalView = novel.TotalView
	record.TotalBookmarks = novel.TotalBookmarks
	record.IsOriginal = novel.IsOriginal
	record.Visible = novel.Visible
	record.IsMuted = novel.IsMuted
	record.NovelAIType = novel.NovelAIType
	if novel.Series != nil {
		record.SeriesID = &novel.Series.ID
		record.SeriesTitle = &novel.Series.Title
	}
	record.CoverURL = novel.ImageUrls.Large
	if record.CoverURL == "" {
		record.CoverURL = novel.ImageUrls.Medium
	}
	record.NovelJSON = novelJSON
}

func fillNovelBookmarkUpdates(updates map[string]any, novel BookmarkNovel, novelJSON string) {
	updates["title"] = novel.Title
	updates["caption"] = novel.Caption
	updates["user_id"] = novel.User.ID
	updates["user_name"] = novel.User.Name
	updates["text_length"] = novel.TextLength
	updates["x_restrict"] = novel.XRestrict
	updates["total_view"] = novel.TotalView
	updates["total_bookmarks"] = novel.TotalBookmarks
	updates["is_original"] = novel.IsOriginal
	updates["visible"] = novel.Visible
	updates["is_muted"] = novel.IsMuted
	updates["novel_ai_type"] = novel.NovelAIType
	if novel.Series != nil {
		updates["series_id"] = novel.Series.ID
		updates["series_title"] = novel.Series.Title
	} else {
		updates["series_id"] = nil
		updates["series_title"] = nil
	}
	coverURL := novel.ImageUrls.Large
	if coverURL == "" {
		coverURL = novel.ImageUrls.Medium
	}
	updates["cover_url"] = coverURL
	updates["novel_json"] = novelJSON
}

func markMissingIllustBookmarksRemoved(ctx context.Context, pixivUserID string, restrict string, runID string) (int, error) {
	now := time.Now()
	result := db.DB(ctx).Model(&model.PixezBookmarkIllust{}).
		Where("pixiv_user_id = ? AND restrict = ? AND removed = ? AND last_export_run_id <> ?", pixivUserID, restrict, false, runID).
		Updates(map[string]any{
			"removed":    true,
			"removed_at": &now,
			"updated_at": now,
		})
	return int(result.RowsAffected), result.Error
}

func markMissingNovelBookmarksRemoved(ctx context.Context, pixivUserID string, restrict string, runID string) (int, error) {
	now := time.Now()
	result := db.DB(ctx).Model(&model.PixezBookmarkNovel{}).
		Where("pixiv_user_id = ? AND restrict = ? AND removed = ? AND last_export_run_id <> ?", pixivUserID, restrict, false, runID).
		Updates(map[string]any{
			"removed":    true,
			"removed_at": &now,
			"updated_at": now,
		})
	return int(result.RowsAffected), result.Error
}

func markBookmarkRunFailed(ctx context.Context, runID string, reqURL string, err error) {
	now := time.Now()
	_ = db.DB(ctx).Model(&model.PixezBookmarkExportRun{}).
		Where("id = ?", runID).
		Updates(map[string]any{
			"status":           model.PixezBookmarkExportStatusFailed,
			"error_message":    err.Error(),
			"last_request_url": reqURL,
			"finished_at":      &now,
			"updated_at":       now,
		}).Error
}

func markBookmarkRunSuccess(ctx context.Context, runID string, summary ExportSummary) error {
	now := time.Now()
	return db.DB(ctx).Model(&model.PixezBookmarkExportRun{}).
		Where("id = ?", runID).
		Updates(map[string]any{
			"status":        model.PixezBookmarkExportStatusSuccess,
			"total_count":   summary.TotalCount,
			"new_count":     summary.NewCount,
			"updated_count": summary.UpdatedCount,
			"removed_count": summary.RemovedCount,
			"finished_at":   &now,
			"updated_at":    now,
		}).Error
}

func addBookmarkStatus(summary *ExportSummary, status bookmarkSaveStatus) {
	switch status {
	case bookmarkSaveCreated:
		summary.NewCount++
	case bookmarkSaveUpdated:
		summary.UpdatedCount++
	case bookmarkSaveRemoved:
		summary.RemovedCount++
	}
}

func errorsIsRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// RemovedBookmarkIllustPayload converts removed bookmark rows into Pixiv list shape.
func RemovedBookmarkIllustPayload(records []model.PixezBookmarkIllust) []json.RawMessage {
	illusts := make([]json.RawMessage, 0, len(records))
	for _, record := range records {
		if json.Valid([]byte(record.IllustJSON)) {
			illusts = append(illusts, json.RawMessage(record.IllustJSON))
		}
	}
	return illusts
}
