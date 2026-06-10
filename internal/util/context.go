// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package util 提供通用工具函数
package util

import "github.com/gin-gonic/gin"

// GetFromContext 从上下文获取指定类型的值
func GetFromContext[T any](c *gin.Context, key string) (T, bool) {
	value, exists := c.Get(key)
	if !exists {
		var zero T
		return zero, false
	}
	typed, ok := value.(T)
	return typed, ok
}

// SetToContext 设置值到上下文
func SetToContext[T any](c *gin.Context, key string, value T) {
	c.Set(key, value)
}
