// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package util

const (
	errCreateHTTPRequestFailed = "创建HTTP请求失败: %w"
	errHTTPRequestFailed       = "请求%s接口失败: %w"
	errInvalidCustomValue      = "invalid value: %v"
	errInvalidSignKey          = "invalid sign key: %w"
	errSignKeyLengthInvalid    = "sign key must be 32 bytes (64 hex characters)"
	errCreateCipherFailed      = "failed to create cipher: %w"
	errCreateGCMFailed         = "failed to create GCM: %w"
	errGenerateNonceFailed     = "failed to generate nonce: %w"
	errDecodeCiphertextFailed  = "failed to decode ciphertext: %w"
	errCiphertextTooShort      = "ciphertext too short"
	errDecryptFailed           = "failed to decrypt: %w"
)
