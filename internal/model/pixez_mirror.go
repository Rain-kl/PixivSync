// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package model

import "time"

// PixEz mirror, bookmark export, and bookmark mirror state constants.
const (
	PixezMirrorTargetIllust = "illust"
	PixezMirrorTargetNovel  = "novel"

	PixezMirrorStatusQueued     = "queued"
	PixezMirrorStatusProcessing = "processing"
	PixezMirrorStatusSuccess    = "success"
	PixezMirrorStatusFailed     = "failed"

	PixezBookmarkExportStatusRunning = "running"
	PixezBookmarkExportStatusSuccess = "success"
	PixezBookmarkExportStatusFailed  = "failed"

	PixezBookmarkMirrorNone       = 0
	PixezBookmarkMirrorDone       = 1
	PixezBookmarkMirrorProcessing = 2
	PixezBookmarkMirrorFailed     = -1
)

// PixezMirrorImageFile stores the Upload mapping for one mirrored Pixiv image.
type PixezMirrorImageFile struct {
	PixivURL   string `json:"pixiv_url"`
	Page       int    `json:"page"`
	UploadID   uint64 `json:"upload_id,string"`
	FileName   string `json:"file_name"`
	Hash       string `json:"hash"`
	Mime       string `json:"mime"`
	Size       int64  `json:"size"`
	StorageKey string `json:"storage_key"`
}

// PixezMirrorIllust stores the read-model for a mirrored Pixiv illustration.
type PixezMirrorIllust struct {
	IllustID        int64     `gorm:"primaryKey;column:illust_id" json:"illust_id"`
	TaskID          string    `gorm:"column:task_id;index" json:"task_id"`
	Status          string    `gorm:"column:status;not null;default:queued;index" json:"status"`
	DetailJSON      string    `gorm:"column:detail_json;type:text" json:"detail_json"`
	ImageFilesJSON  string    `gorm:"column:image_files_json;type:text;not null;default:'[]'" json:"image_files_json"`
	RequestURLsJSON string    `gorm:"column:request_urls_json;type:text;not null;default:'[]'" json:"request_urls_json"`
	RetryURLsJSON   string    `gorm:"column:retry_urls_json;type:text;not null;default:'[]'" json:"retry_urls_json"`
	ErrorMessage    string    `gorm:"column:error_message;type:text" json:"error_message"`
	TotalCount      int       `gorm:"column:total_count;not null;default:0" json:"total_count"`
	SuccessCount    int       `gorm:"column:success_count;not null;default:0" json:"success_count"`
	FailedCount     int       `gorm:"column:failed_count;not null;default:0" json:"failed_count"`
	CreatedAt       time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName returns the PixEz read-model table name.
func (PixezMirrorIllust) TableName() string { return "mirror_illust" }

// PixezMirrorNovel stores the read-model for a mirrored Pixiv novel.
type PixezMirrorNovel struct {
	NovelID         int64     `gorm:"primaryKey;column:novel_id" json:"novel_id"`
	TaskID          string    `gorm:"column:task_id;index" json:"task_id"`
	Status          string    `gorm:"column:status;not null;default:queued;index" json:"status"`
	DetailJSON      string    `gorm:"column:detail_json;type:text" json:"detail_json"`
	TextJSON        string    `gorm:"column:text_json;type:text" json:"text_json"`
	RequestURLsJSON string    `gorm:"column:request_urls_json;type:text;not null;default:'[]'" json:"request_urls_json"`
	RetryURLsJSON   string    `gorm:"column:retry_urls_json;type:text;not null;default:'[]'" json:"retry_urls_json"`
	ErrorMessage    string    `gorm:"column:error_message;type:text" json:"error_message"`
	TotalCount      int       `gorm:"column:total_count;not null;default:0" json:"total_count"`
	SuccessCount    int       `gorm:"column:success_count;not null;default:0" json:"success_count"`
	FailedCount     int       `gorm:"column:failed_count;not null;default:0" json:"failed_count"`
	CreatedAt       time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName returns the PixEz read-model table name.
func (PixezMirrorNovel) TableName() string { return "mirror_novel" }

// PixezBookmarkExportRun records one full bookmark export pass.
type PixezBookmarkExportRun struct {
	ID             string     `gorm:"primaryKey;column:id" json:"id"`
	TargetType     string     `gorm:"column:target_type;not null;index" json:"target_type"`
	PixivUserID    string     `gorm:"column:pixiv_user_id;not null;index" json:"pixiv_user_id"`
	Restrict       string     `gorm:"column:restrict;not null;index" json:"restrict"`
	Status         string     `gorm:"column:status;not null;index" json:"status"`
	TotalCount     int        `gorm:"column:total_count;not null;default:0" json:"total_count"`
	NewCount       int        `gorm:"column:new_count;not null;default:0" json:"new_count"`
	UpdatedCount   int        `gorm:"column:updated_count;not null;default:0" json:"updated_count"`
	RemovedCount   int        `gorm:"column:removed_count;not null;default:0" json:"removed_count"`
	ErrorMessage   string     `gorm:"column:error_message;type:text" json:"error_message"`
	StartedAt      time.Time  `gorm:"column:started_at;not null" json:"started_at"`
	FinishedAt     *time.Time `gorm:"column:finished_at" json:"finished_at"`
	NextURL        string     `gorm:"column:next_url;type:text" json:"next_url"`
	LastRequestURL string     `gorm:"column:last_request_url;type:text" json:"last_request_url"`
	CreatedAt      time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

// TableName returns the PixEz bookmark export table name.
func (PixezBookmarkExportRun) TableName() string { return "bookmark_export_runs" }

// PixezBookmarkIllust stores the latest known Pixiv bookmark illustration payload.
type PixezBookmarkIllust struct {
	ID               uint       `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	PixivUserID      string     `gorm:"column:pixiv_user_id;not null;uniqueIndex:idx_bookmark_illust_user_restrict_illust" json:"pixiv_user_id"`
	Restrict         string     `gorm:"column:restrict;not null;uniqueIndex:idx_bookmark_illust_user_restrict_illust" json:"restrict"`
	IllustID         int64      `gorm:"column:illust_id;not null;uniqueIndex:idx_bookmark_illust_user_restrict_illust" json:"illust_id"`
	Title            string     `gorm:"column:title" json:"title"`
	Type             string     `gorm:"column:type" json:"type"`
	UserID           int64      `gorm:"column:user_id" json:"user_id"`
	UserName         string     `gorm:"column:user_name" json:"user_name"`
	PageCount        int        `gorm:"column:page_count" json:"page_count"`
	Width            int        `gorm:"column:width" json:"width"`
	Height           int        `gorm:"column:height" json:"height"`
	SanityLevel      int        `gorm:"column:sanity_level" json:"sanity_level"`
	XRestrict        int        `gorm:"column:x_restrict" json:"x_restrict"`
	TotalView        int        `gorm:"column:total_view" json:"total_view"`
	TotalBookmarks   int        `gorm:"column:total_bookmarks" json:"total_bookmarks"`
	Visible          bool       `gorm:"column:visible;not null;default:false" json:"visible"`
	IsMuted          bool       `gorm:"column:is_muted;not null;default:false" json:"is_muted"`
	IllustAIType     int        `gorm:"column:illust_ai_type" json:"illust_ai_type"`
	IllustJSON       string     `gorm:"column:illust_json;type:text;not null" json:"illust_json"`
	LastExportRunID  string     `gorm:"column:last_export_run_id;not null;index" json:"last_export_run_id"`
	LastSeenAt       time.Time  `gorm:"column:last_seen_at;not null" json:"last_seen_at"`
	MirrorStatus     int        `gorm:"column:mirror_status;not null;default:0;index" json:"mirror_status"`
	MirrorRetryCount int        `gorm:"column:mirror_retry_count;not null;default:0" json:"mirror_retry_count"`
	Removed          bool       `gorm:"column:removed;not null;default:false;index" json:"removed"`
	RemovedAt        *time.Time `gorm:"column:removed_at" json:"removed_at"`
	CreatedAt        time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

// TableName returns the PixEz bookmark table name.
func (PixezBookmarkIllust) TableName() string { return "bookmark_illusts" }

// PixezBookmarkNovel stores the latest known Pixiv bookmark novel payload.
type PixezBookmarkNovel struct {
	ID               uint       `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	PixivUserID      string     `gorm:"column:pixiv_user_id;not null;uniqueIndex:idx_bookmark_novel_user_restrict_novel" json:"pixiv_user_id"`
	Restrict         string     `gorm:"column:restrict;not null;uniqueIndex:idx_bookmark_novel_user_restrict_novel" json:"restrict"`
	NovelID          int64      `gorm:"column:novel_id;not null;uniqueIndex:idx_bookmark_novel_user_restrict_novel" json:"novel_id"`
	Title            string     `gorm:"column:title" json:"title"`
	Caption          string     `gorm:"column:caption" json:"caption"`
	UserID           int64      `gorm:"column:user_id" json:"user_id"`
	UserName         string     `gorm:"column:user_name" json:"user_name"`
	TextLength       int        `gorm:"column:text_length" json:"text_length"`
	XRestrict        int        `gorm:"column:x_restrict" json:"x_restrict"`
	TotalView        int        `gorm:"column:total_view" json:"total_view"`
	TotalBookmarks   int        `gorm:"column:total_bookmarks" json:"total_bookmarks"`
	IsOriginal       bool       `gorm:"column:is_original;not null;default:false" json:"is_original"`
	Visible          bool       `gorm:"column:visible;not null;default:false" json:"visible"`
	IsMuted          bool       `gorm:"column:is_muted;not null;default:false" json:"is_muted"`
	NovelAIType      int        `gorm:"column:novel_ai_type" json:"novel_ai_type"`
	SeriesID         *int64     `gorm:"column:series_id" json:"series_id"`
	SeriesTitle      *string    `gorm:"column:series_title" json:"series_title"`
	CoverURL         string     `gorm:"column:cover_url" json:"cover_url"`
	NovelJSON        string     `gorm:"column:novel_json;type:text;not null" json:"novel_json"`
	LastExportRunID  string     `gorm:"column:last_export_run_id;not null;index" json:"last_export_run_id"`
	LastSeenAt       time.Time  `gorm:"column:last_seen_at;not null" json:"last_seen_at"`
	MirrorStatus     int        `gorm:"column:mirror_status;not null;default:0;index" json:"mirror_status"`
	MirrorRetryCount int        `gorm:"column:mirror_retry_count;not null;default:0" json:"mirror_retry_count"`
	Removed          bool       `gorm:"column:removed;not null;default:false;index" json:"removed"`
	RemovedAt        *time.Time `gorm:"column:removed_at" json:"removed_at"`
	CreatedAt        time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

// TableName returns the PixEz bookmark table name.
func (PixezBookmarkNovel) TableName() string { return "bookmark_novels" }
