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

import "strings"

// emailPartsCount 邮箱地址由 @ 分割为两部分
const (
	emailPartsCount    = 2
	emailLocalMinChars = 2 // 邮箱 local 部分掩码显示的最小字符数
)

// DerefString 安全地解引用字符串指针，nil 返回空字符串
func DerefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// MaskEmail 安全脱敏用户的邮箱地址（例如 us***@example.com）
func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != emailPartsCount {
		return email
	}
	local := parts[0]
	domain := parts[1]
	if len(local) <= emailLocalMinChars {
		return "**@" + domain
	}
	return local[:2] + "***" + local[len(local)-1:] + "@" + domain
}
