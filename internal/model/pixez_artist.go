// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package model

import "time"

// PixezArtist is a globally shared creator discovered from saved works.
type PixezArtist struct {
	ArtistID  int64     `gorm:"primaryKey;autoIncrement:false" json:"artist_id,string"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName returns the shared creator table.
func (PixezArtist) TableName() string { return "pixez_artists" }

// PixezArtistWork preserves the first known work summary separately from bookmarks.
type PixezArtistWork struct {
	ID                   uint64     `gorm:"primaryKey" json:"id,string"`
	ArtistID             int64      `gorm:"index" json:"artist_id,string"`
	TargetType           string     `gorm:"uniqueIndex:idx_artist_work_target" json:"target_type"`
	TargetID             int64      `gorm:"uniqueIndex:idx_artist_work_target" json:"target_id,string"`
	FirstSummaryJSON     string     `json:"-"`
	DiscoveredFromMirror bool       `json:"discovered_from_mirror"`
	DiscoveredFromScan   bool       `json:"discovered_from_scan"`
	LastSeenRunID        string     `json:"-"`
	LastSeenAt           *time.Time `json:"last_seen_at"`
	NotReturned          bool       `json:"not_returned"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// TableName returns the immutable work directory table.
func (PixezArtistWork) TableName() string { return "pixez_artist_works" }

// PixezArtistSubscription independently enables one creator's work category.
type PixezArtistSubscription struct {
	ID             uint64     `gorm:"primaryKey" json:"id,string"`
	ArtistID       int64      `gorm:"uniqueIndex:idx_artist_subscription_target" json:"artist_id,string"`
	TargetType     string     `gorm:"uniqueIndex:idx_artist_subscription_target" json:"target_type"`
	Enabled        bool       `json:"enabled"`
	Revision       int64      `json:"revision"`
	ActiveRunID    string     `json:"active_run_id"`
	LeaseExpiresAt *time.Time `json:"-"`
	NextDueAt      *time.Time `json:"next_due_at"`
	LastAttemptAt  *time.Time `json:"last_attempt_at"`
	LastSuccessAt  *time.Time `json:"last_success_at"`
	LastError      string     `json:"last_error"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// TableName returns the category subscription table.
func (PixezArtistSubscription) TableName() string { return "pixez_artist_subscriptions" }

// PixezArtistSyncRun records a resumable directory scan, independently of downloads.
type PixezArtistSyncRun struct {
	ExecutionToken  string     `json:"-"`
	LeaseExpiresAt  *time.Time `json:"-"`
	ID              string     `gorm:"primaryKey" json:"id"`
	ArtistID        int64      `json:"artist_id,string"`
	TargetType      string     `json:"target_type"`
	Revision        int64      `json:"-"`
	Download        bool       `json:"download"`
	TriggeredBy     string     `json:"triggered_by"`
	AccountID       string     `json:"-"`
	TaskID          string     `json:"task_id"`
	Status          string     `json:"status"`
	Phase           int        `json:"-"`
	NextURL         string     `json:"-"`
	DiscoveredCount int        `json:"discovered_count"`
	QueuedCount     int        `json:"queued_count"`
	SkippedCount    int        `json:"skipped_count"`
	ErrorMessage    string     `json:"error_message"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	FinishedAt      *time.Time `json:"finished_at"`
}

// TableName returns the scan history table.
func (PixezArtistSyncRun) TableName() string { return "pixez_artist_sync_runs" }
