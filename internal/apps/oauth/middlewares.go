// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package oauth

import (
	"net/http"
	"time"

	"github.com/Rain-kl/Wavelet/internal/common"
	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/otel_trace"
	"github.com/Rain-kl/Wavelet/internal/util"
	"github.com/gin-gonic/gin"
)

type loginRequiredAuditLog struct {
	UserID     uint64 `json:"user_id"`
	Username   string `json:"username"`
	ClientIP   string `json:"client_ip"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	RequestURI string `json:"request_uri"`
	UserAgent  string `json:"user_agent"`
	Referer    string `json:"referer"`
}

// LoginRequired 返回登录鉴权中间件，校验 Access Token 或 Session
func LoginRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// init trace
		ctx, span := otel_trace.Start(c.Request.Context(), "LoginRequired")
		defer span.End()

		// check token in headers
		tokenStr := c.GetHeader("X-Access-Token")
		if tokenStr == "" {
			authHeader := c.GetHeader("Authorization")
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				tokenStr = authHeader[7:]
			}
		}

		var user model.User
		var authenticated bool

		if tokenStr != "" {
			tokenHash := model.HashToken(tokenStr)
			var tokenRecord model.AccessToken
			if err := db.DB(ctx).Where("token_hash = ?", tokenHash).First(&tokenRecord).Error; err == nil {
				if err := db.DB(ctx).Where("id = ? AND is_active = ?", tokenRecord.UserID, true).First(&user).Error; err == nil {
					authenticated = true
					// update token last used time
					now := time.Now()
					db.DB(ctx).Model(&tokenRecord).Update("last_used_at", &now)
				}
			}
		}

		if !authenticated {
			// load user from session
			userID := GetUserIDFromContext(c)
			if userID <= 0 {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error_msg": common.UnAuthorized, "data": nil})
				return
			}

			// load user from db to make sure is active
			tx := db.DB(ctx).Where("id = ? AND is_active = ?", userID, true).First(&user)
			if tx.Error != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error_msg": common.UnAuthorized, "data": nil})
				return
			}
		}

		// log
		LogForAudit(ctx, &user, c)

		// set user info
		util.SetToContext(c, UserObjKey, &user)

		// next
		c.Next()
	}
}
