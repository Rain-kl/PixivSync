// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package oauth

import (
	"errors"
	"net/http"

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

// GetUserFromRequest 校验 Access Token 或 Session 并返回用户对象，如果未登录或用户失效则返回 error
func GetUserFromRequest(c *gin.Context) (*model.User, error) {
	ctx := c.Request.Context()

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
	var tokenAuth bool
	var tokenAdmin bool

	if tokenStr != "" {
		tokenHash := model.HashToken(tokenStr)
		var tokenRecord model.AccessToken
		if err := db.DB(ctx).Where("token_hash = ?", tokenHash).First(&tokenRecord).Error; err == nil {
			if err := db.DB(ctx).Where("id = ? AND is_active = ?", tokenRecord.UserID, true).First(&user).Error; err == nil {
				authenticated = true
				tokenAuth = true
				tokenAdmin = tokenRecord.IsAdmin
			}
		}
	}

	if !authenticated {
		// load user from session
		userID := GetUserIDFromContext(c)
		if userID <= 0 {
			return nil, errors.New("unauthorized")
		}

		// load user from db to make sure is active
		tx := db.DB(ctx).Where("id = ? AND is_active = ?", userID, true).First(&user)
		if tx.Error != nil {
			return nil, tx.Error
		}
	}

	// set keys in context
	util.SetToContext(c, TokenAuthKey, tokenAuth)
	util.SetToContext(c, TokenAdminKey, tokenAdmin)

	return &user, nil
}

// LoginRequired 返回登录鉴权中间件，校验 Access Token 或 Session
func LoginRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// init trace
		ctx, span := otel_trace.Start(c.Request.Context(), "LoginRequired")
		defer span.End()

		user, err := GetUserFromRequest(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error_msg": common.UnAuthorized, "data": nil})
			return
		}

		// log
		LogForAudit(ctx, user, c)

		// set user info
		util.SetToContext(c, UserObjKey, user)

		// next
		c.Next()
	}
}
