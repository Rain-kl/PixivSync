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

// Package model 定义数据模型与 GORM 实体
package model

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

const (
	tokenByteLength = 24 // Token 随机字节长度
	maskThreshold   = 8  // 脱敏显示阈值
)

// AccessToken 个人访问令牌实体
type AccessToken struct {
	ID          uint64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID      uint64     `json:"user_id" gorm:"index;not null"`
	Name        string     `json:"name" gorm:"size:128;not null"`
	TokenHash   string     `json:"-" gorm:"size:64;uniqueIndex;not null"`
	MaskedToken string     `json:"masked_token" gorm:"size:64;not null"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

// GenerateTokenString 生成加密安全的随机 Token 值
func GenerateTokenString() (string, error) {
	bytes := make([]byte, tokenByteLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("at_%s", hex.EncodeToString(bytes)), nil
}

// HashToken 计算 Token 的 SHA-256 哈希值用于数据库存储与查询
func HashToken(token string) string {
	h := sha256.New()
	h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil))
}

// MaskTokenString 生成脱敏显示的 Token，仅保留前缀和最后四位
func MaskTokenString(token string) string {
	if len(token) <= maskThreshold {
		return "at_****"
	}
	return fmt.Sprintf("%s...%s", token[:7], token[len(token)-4:])
}
