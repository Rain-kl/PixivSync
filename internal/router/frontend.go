//go:build !embed_frontend

// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

// Package router 提供 HTTP 路由注册与服务启动
package router

import "github.com/gin-gonic/gin"

func registerFrontend(_ *gin.Engine) {
	// No-op when not embedding frontend
}
