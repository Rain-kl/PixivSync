// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package upload

const (
	maxUploadSize      = 32 * 1024 * 1024 // 32MB
	detectContentBytes = 512              // http.DetectContentType 需要的最小字节数
	uploadDirPerm      = 0755             // 上传目录权限
	uploadFilePerm     = 0644             // 上传文件权限
)
