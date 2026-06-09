// Package pixez provides PixEz companion sync APIs on top of Wavelet auth.
package pixez

const (
	errPixivUserIDRequired = "pixiv_user_id is required"
	errUserNotFound        = "user not found"
	errInvalidRequestBody  = "invalid request body"
	errFetchUsersFailed    = "failed to fetch users"
	errFetchUserFailed     = "failed to fetch user"
	errSaveUserFailed      = "failed to save user credentials"
	errDeleteUserFailed    = "failed to delete user credentials and data"
	errFetchSyncDataFailed = "failed to fetch sync data"
	errSaveSyncDataFailed  = "failed to save sync data"
	errFetchHashesFailed   = "failed to fetch sync data hashes"
)
