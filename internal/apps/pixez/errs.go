// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

// Package pixez provides PixEz companion sync APIs on top of Wavelet auth.
package pixez

const (
	errPixivUserIDRequired         = "pixiv_user_id is required"
	errUserNotFound                = "user not found"
	errInvalidRequestBody          = "invalid request body"
	errFetchUsersFailed            = "failed to fetch users"
	errFetchUserFailed             = "failed to fetch user"
	errSaveUserFailed              = "failed to save user credentials"
	errDeleteUserFailed            = "failed to delete user credentials and data"
	errFetchSyncDataFailed         = "failed to fetch sync data"
	errSaveSyncDataFailed          = "failed to save sync data"
	errFetchHashesFailed           = "failed to fetch sync data hashes"
	errDispatchMirrorTaskFailed    = "failed to dispatch mirror task"
	errQueryMirrorStatusFailed     = "failed to query mirror status"
	errFetchMirrorDetailFailed     = "failed to fetch mirror detail"
	errTooManyMirrorIDs            = "mirror ID list must not exceed 500 items"
	errDeleteMirrorFailed          = "failed to delete mirror record"
	errFetchRemovedBookmarksFailed = "failed to fetch removed bookmarks"
	errFetchDashboardFailed        = "failed to fetch PixEz dashboard"
	errFetchExportRunsFailed       = "failed to fetch bookmark export runs"
	errFetchBookmarksFailed        = "failed to fetch bookmark mirror records"
	errFetchBookmarkDetailFailed   = "failed to fetch bookmark mirror detail"
	errRefreshPixivAccountFailed   = "failed to refresh Pixiv account state"
	errAddUserFailed               = "failed to add Pixiv user by refresh token"
)
