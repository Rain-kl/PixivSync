// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package oauth

import (
	"context"
	"errors"
	"net/http"

	"github.com/Rain-kl/Wavelet/internal/common"
	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/util"
	otel_trace "github.com/Rain-kl/Wavelet/pkg/trace"
	"github.com/gin-contrib/sessions"
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

func getUserByToken(ctx context.Context, tokenStr string) (*model.User, *model.AccessToken, error) {
	tokenHash := model.HashToken(tokenStr)
	var tokenRecord model.AccessToken
	if err := db.DB(ctx).Where("token_hash = ?", tokenHash).First(&tokenRecord).Error; err != nil {
		return nil, nil, err
	}
	var user model.User
	if err := db.DB(ctx).Where("id = ? AND is_active = ?", tokenRecord.UserID, true).First(&user).Error; err != nil {
		return nil, nil, err
	}
	return &user, &tokenRecord, nil
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

	// 优先使用 Access Token 鉴权
	if tokenStr != "" {
		if user, tokenRecord, err := getUserByToken(ctx, tokenStr); err == nil {
			// 强行阻止 system 用户任何会话/Token 鉴权通过
			if user.Username == "system" {
				return nil, errors.New("system user is not allowed to login")
			}
			util.SetToContext(c, TokenAuthKey, true)
			util.SetToContext(c, TokenAdminKey, tokenRecord.IsAdmin)
			return user, nil
		}
	}

	// 降级使用 Session 鉴权
	userID := GetUserIDFromContext(c)
	if userID <= 0 {
		return nil, errors.New("unauthorized")
	}

	var user model.User
	// load user from db to make sure is active
	tx := db.DB(ctx).Where("id = ? AND is_active = ?", userID, true).First(&user)
	if tx.Error != nil {
		return nil, tx.Error
	}

	// 密码哈希校验：当用户存在本地密码时，要求 Session 中的密码哈希必须与当前数据库中一致
	if user.Password != "" {
		session := sessions.Default(c)
		sessionHash, _ := session.Get(PasswordHashKey).(string)
		if sessionHash != user.Password {
			return nil, errors.New("session expired due to password change")
		}
	}

	// set keys in context for session auth
	util.SetToContext(c, TokenAuthKey, false)
	util.SetToContext(c, TokenAdminKey, false)

	// 强行阻止 system 用户任何会话/Token 鉴权通过
	if user.Username == "system" {
		return nil, errors.New("system user is not allowed to login")
	}

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

// DisallowTokenAuth 拒绝使用 Access Token 进行身份验证的请求访问该端点
func DisallowTokenAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if tokenAuth, _ := util.GetFromContext[bool](c, TokenAuthKey); tokenAuth {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error_msg": ErrTokenAuthNotAllowed, "data": nil})
			return
		}
		c.Next()
	}
}
