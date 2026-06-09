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
