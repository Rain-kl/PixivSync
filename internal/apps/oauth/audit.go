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

// Package oauth 提供 OAuth/OIDC 认证与会话管理
package oauth

import (
	"context"
	"encoding/json"

	"github.com/Rain-kl/Wavelet/internal/logger"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/gin-gonic/gin"
)

// LogForAudit 将登录鉴权审计日志写入 Logger
func LogForAudit(ctx context.Context, user *model.User, c *gin.Context) {
	auditLog := loginRequiredAuditLog{
		UserID:     user.ID,
		Username:   user.Username,
		ClientIP:   c.ClientIP(),
		Method:     c.Request.Method,
		Path:       c.Request.URL.Path,
		RequestURI: c.Request.RequestURI,
		UserAgent:  c.Request.UserAgent(),
		Referer:    c.Request.Referer(),
	}
	auditJSON, err := json.Marshal(auditLog)
	if err != nil {
		logger.ErrorF(ctx, "[LoginRequiredAudit] marshal failed: %v", err)
		logger.InfoF(ctx, "[LoginRequiredAudit] %s %d %s", c.ClientIP(), user.ID, user.Username)
	} else {
		logger.InfoF(ctx, "[LoginRequiredAudit] %s", auditJSON)
	}
}
