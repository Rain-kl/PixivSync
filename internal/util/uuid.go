/*
Copyright 2025 linux.do
Modified by Arctel.net, 2026

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

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"

	"github.com/google/uuid"
)

// uniqueIDBytes 生成唯一 ID 所需的随机字节长度
const uniqueIDBytes = 32

// GenerateUniqueIDSimple 生成 64 位唯一标识符
func GenerateUniqueIDSimple() string {
	randomBytes := make([]byte, uniqueIDBytes)
	if _, err := io.ReadFull(rand.Reader, randomBytes); err != nil {
		// 如果随机数生成失败，使用 UUID 作为后备
		uuidBytes := []byte(uuid.NewString())
		hash := sha256.Sum256(uuidBytes)
		copy(randomBytes, hash[:])
	}
	return hex.EncodeToString(randomBytes)
}
