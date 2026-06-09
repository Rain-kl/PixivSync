/*
Copyright 2026 Arctel.net

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
