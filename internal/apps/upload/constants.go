// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package upload

const (
	maxUploadSize      = 32 * 1024 * 1024 // 32MB
	detectContentBytes = 512              // http.DetectContentType 需要的最小字节数
	uploadDirPerm      = 0755             // 上传目录权限
	uploadFilePerm     = 0644             // 上传文件权限
	imageQualityLow    = "low"
	imageQualityMedium = "medium"
	imageQualityHigh   = "high"
	imageQualityOrigin = "origin"
	storageDriverLocal = "local"
)
