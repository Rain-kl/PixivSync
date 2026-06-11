// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package upload 提供文件上传与下载功能
package upload

// 文件管理常量
const (
	ErrNoFileSelected                  = "请选择要上传的文件"
	ErrUnsupportedFormat               = "只支持 JPG、PNG、WEBP 格式的图片"
	ErrProcessFileFailed               = "处理文件失败"
	ErrSaveFileFailed                  = "保存文件失败"
	ErrOpenFileFailed                  = "打开文件失败"
	ErrSaveUploadRecordFailed          = "保存上传记录失败"
	ErrGenericFileTooLarge             = "文件大小不能超过 32MB"
	ErrFileContentExtensionMismatch    = "文件内容与扩展名不匹配，可能包含安全风险"
	ErrFileValidationFailed            = "文件校验失败"
	ErrInvalidMetadataJSON             = "元数据 JSON 格式不合法"
	ErrInvalidFileID                   = "无效的文件 ID"
	ErrQueryUploadRecordFailed         = "查询文件记录失败"
	ErrInvalidBatchDownloadRequest     = "参数绑定失败，请传入有效的文件 ID 数组"
	ErrInvalidIDValueFormat            = "无效的 ID 值: %s"
	ErrRetrieveUploadRecordsFailed     = "检索文件记录失败"
	ErrNoValidFilesForArchive          = "没有找到任何有效的文件记录进行打包"
	ErrInvalidParams                   = "参数错误"
	ErrQueryFileCountFailed            = "查询文件数量失败"
	ErrQueryFileListFailed             = "查询文件列表失败"
	ErrDeleteFileFailed                = "删除文件失败"
	ErrS3KeyRequired                   = "s3 key must not be empty"
	ErrS3KeyTooLongFormat              = "s3 key exceeds maximum length of %d"
	ErrS3KeyStartsWithSlash            = "s3 key must not start with /"
	ErrS3KeyContainsNullBytes          = "s3 key must not contain null bytes"
	ErrQueryUnusedUploadsFailed        = "查询未使用的上传文件失败: %w"
	errImageCacheWarmupPayloadRequired = "图片缓存预热参数不能为空"
	errInvalidImageCacheWarmupPayload  = "图片缓存预热参数格式无效: %w"
	errInvalidImageCacheWarmupQuality  = "图片质量仅支持 low、medium、high"
	errParseImageCacheWarmupPayload    = "解析图片缓存预热参数失败: %w"
	errQueryImagesForCacheWarmup       = "查询待预热图片失败: %w"
)
