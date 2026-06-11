// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package pixez contains PixEz companion-server business services.
package pixez

import (
	"time"
)

const (
	restrictPublic  = "public"
	restrictPrivate = "private"

	limitUnknownIllust = "limit_unknown_360"
	limitUnknownNovel  = "limit_unknown_100"
)

var bookmarkRestricts = []string{restrictPublic, restrictPrivate}

// IllustDetail mirrors Pixiv /v1/illust/detail.
type IllustDetail struct {
	Illust Illust `json:"illust"`
}

// BookmarkIllustResponse mirrors Pixiv /v1/user/bookmarks/illust.
type BookmarkIllustResponse struct {
	Illusts []Illust `json:"illusts"`
	NextURL string   `json:"next_url"`
}

// Illust is the subset of Pixiv illustration fields needed by PixEz sync.
type Illust struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Type      string `json:"type"`
	ImageUrls struct {
		SquareMedium string `json:"square_medium"`
		Medium       string `json:"medium"`
		Large        string `json:"large"`
	} `json:"image_urls"`
	Caption  string `json:"caption"`
	Restrict int    `json:"restrict"`
	User     struct {
		ID               int64  `json:"id"`
		Name             string `json:"name"`
		Account          string `json:"account"`
		ProfileImageUrls struct {
			Medium string `json:"medium"`
		} `json:"profile_image_urls"`
		IsFollowed      bool `json:"is_followed"`
		IsAcceptRequest bool `json:"is_accept_request"`
	} `json:"user"`
	Tags []struct {
		Name           string  `json:"name"`
		TranslatedName *string `json:"translated_name"`
	} `json:"tags"`
	Tools          []string `json:"tools"`
	CreateDate     string   `json:"create_date"`
	PageCount      int      `json:"page_count"`
	Width          int      `json:"width"`
	Height         int      `json:"height"`
	SanityLevel    int      `json:"sanity_level"`
	XRestrict      int      `json:"x_restrict"`
	Series         any      `json:"series"`
	MetaSinglePage struct {
		OriginalImageURL string `json:"original_image_url"`
	} `json:"meta_single_page"`
	MetaPages []struct {
		ImageUrls struct {
			SquareMedium string `json:"square_medium"`
			Medium       string `json:"medium"`
			Large        string `json:"large"`
			Original     string `json:"original"`
		} `json:"image_urls"`
	} `json:"meta_pages"`
	TotalView             int      `json:"total_view"`
	TotalBookmarks        int      `json:"total_bookmarks"`
	IsBookmarked          bool     `json:"is_bookmarked"`
	Visible               bool     `json:"visible"`
	IsMuted               bool     `json:"is_muted"`
	IllustAIType          int      `json:"illust_ai_type"`
	IllustBookStyle       int      `json:"illust_book_style"`
	Request               any      `json:"request"`
	RestrictionAttributes []string `json:"restriction_attributes"`
}

// NovelDetail mirrors Pixiv /v2/novel/detail.
type NovelDetail struct {
	Novel Novel `json:"novel"`
}

// Novel is the subset of Pixiv novel fields needed by PixEz sync.
type Novel struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	User  struct {
		ID               int64  `json:"id"`
		Name             string `json:"name"`
		Account          string `json:"account"`
		ProfileImageUrls struct {
			Medium string `json:"medium"`
		} `json:"profile_image_urls"`
		IsFollowed bool `json:"is_followed"`
	} `json:"user"`
	Caption    string `json:"caption"`
	CreateDate string `json:"create_date"`
	Tags       []struct {
		Name           string  `json:"name"`
		TranslatedName *string `json:"translated_name"`
	} `json:"tags"`
	PageCount      int  `json:"page_count"`
	TextLength     int  `json:"text_length"`
	TotalBookmarks int  `json:"total_bookmarks"`
	TotalView      int  `json:"total_view"`
	IsBookmarked   bool `json:"is_bookmarked"`
	IsMuted        bool `json:"is_muted"`
	NovelAIType    int  `json:"novel_ai_type"`
	ImageUrls      struct {
		SquareMedium string `json:"square_medium"`
		Medium       string `json:"medium"`
		Large        string `json:"large"`
	} `json:"image_urls"`
	Series *struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
	} `json:"series"`
}

// BookmarkNovelResponse mirrors Pixiv /v1/user/bookmarks/novel.
type BookmarkNovelResponse struct {
	Novels  []BookmarkNovel `json:"novels"`
	NextURL string          `json:"next_url"`
}

// BookmarkNovel is the bookmark-list representation returned by Pixiv.
type BookmarkNovel struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Caption    string `json:"caption"`
	Restrict   int    `json:"restrict"`
	XRestrict  int    `json:"x_restrict"`
	IsOriginal bool   `json:"is_original"`
	ImageUrls  struct {
		SquareMedium string `json:"square_medium"`
		Medium       string `json:"medium"`
		Large        string `json:"large"`
	} `json:"image_urls"`
	CreateDate     string `json:"create_date"`
	TextLength     int    `json:"text_length"`
	TotalView      int    `json:"total_view"`
	TotalBookmarks int    `json:"total_bookmarks"`
	IsBookmarked   bool   `json:"is_bookmarked"`
	Visible        bool   `json:"visible"`
	IsMuted        bool   `json:"is_muted"`
	NovelAIType    int    `json:"novel_ai_type"`
	User           struct {
		ID               int64  `json:"id"`
		Name             string `json:"name"`
		Account          string `json:"account"`
		ProfileImageUrls struct {
			Medium string `json:"medium"`
		} `json:"profile_image_urls"`
		IsFollowed      bool `json:"is_followed"`
		IsAcceptRequest bool `json:"is_accept_request"`
	} `json:"user"`
	Tags []struct {
		Name           string  `json:"name"`
		TranslatedName *string `json:"translated_name"`
	} `json:"tags"`
	Series *struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
	} `json:"series"`
	PageCount     int `json:"page_count"`
	TotalComments int `json:"total_comments"`
}

// NovelWebContent is the JSON object embedded in Pixiv /webview/v2/novel.
type NovelWebContent struct {
	Text string `json:"text"`
}

// ExportSummary summarizes one task-level bookmark export.
type ExportSummary struct {
	TargetType   string
	UserCount    int
	RunCount     int
	TotalCount   int
	NewCount     int
	UpdatedCount int
	RemovedCount int
}

// ImportLegacyRequest configures legacy PixEz server import.
type ImportLegacyRequest struct {
	SQLitePath string `json:"sqlite_path"`
	MirrorDir  string `json:"mirror_dir"`
	DryRun     bool   `json:"dry_run"`
}

// ImportLegacySummary summarizes legacy import output.
type ImportLegacySummary struct {
	DryRun          bool      `json:"dry_run"`
	PixivUsers      int       `json:"pixiv_users"`
	SyncRows        int       `json:"sync_rows"`
	BookmarkIllusts int       `json:"bookmark_illusts"`
	BookmarkNovels  int       `json:"bookmark_novels"`
	MirrorIllusts   int       `json:"mirror_illusts"`
	MirrorNovels    int       `json:"mirror_novels"`
	ImportedFiles   int       `json:"imported_files"`
	MissingFiles    int       `json:"missing_files"`
	StartedAt       time.Time `json:"started_at"`
	FinishedAt      time.Time `json:"finished_at"`
}
