// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

// Package storage 提供文件存储抽象层，包括 S3 兼容存储和本地缓存。
package storage

// ErrS3InitializationFailed S3 存储初始化失败错误
type ErrS3InitializationFailed struct{}

func (e ErrS3InitializationFailed) Error() string {
	return errS3InitializationFailed
}

// LocalCacheError 本地缓存错误
type LocalCacheError struct{}

func (e LocalCacheError) Error() string {
	return errLocalCache
}

const (
	errS3InitializationFailed = "S3存储初始化失败"
	errLocalCache             = "本地缓存错误"
	errS3PutObjectFailed      = "s3 put object failed: %w"
	errS3GetObjectFailed      = "s3 get object failed: %w"
	errCDNRequestFailed       = "cdn request failed: %w"
	errCDNStatusFailed        = "cdn returned status %d"
	errS3DeleteObjectFailed   = "s3 delete object failed: %w"
)
