// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package pixez

const (
	// TargetTypeIllust represents the illustration target type.
	TargetTypeIllust = 0
	// TargetTypeNovel represents the novel target type.
	TargetTypeNovel = 1
)

const (
	defaultAutoMirrorLimit = 50
	maxAutoMirrorLimit     = 500
)

const (
	defaultRemovedBookmarkLimit = 30
	maxRemovedBookmarkLimit     = 100
)

const (
	defaultManagementPageSize = 24
	maxExportRunPageSize      = 50
	maxManagementPageSize     = 100
	dashboardRecentRunLimit   = 8
	percentScale              = 100
)

const maxBatchMirrorIDs = 500

const (
	keyItems         = "items"
	keyTotal         = "total"
	keyPage          = "page"
	keyPageSize      = "page_size"
	keyError         = "error"
	statusSuccess    = "success"
	statusProcessing = "processing"
	statusFailed     = "failed"
	paramTypeNumber  = "number"
	paramTypeString  = "string"
	keyTargetType    = "target_type"
	labelTargetType  = "目标类型"
)

const (
	defaultSQLitePath = "server/pixez-sync.db"
	defaultMirrorDir  = "server/data/mirror"
)
