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
