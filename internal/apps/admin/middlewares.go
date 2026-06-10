// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package admin

import (
	"net/http"

	"github.com/Rain-kl/Wavelet/internal/logger"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/otel_trace"
	"github.com/Rain-kl/Wavelet/internal/util"

	"github.com/Rain-kl/Wavelet/internal/apps/oauth"
	"github.com/gin-gonic/gin"
)

// LoginAdminRequired 返回管理员权限校验中间件
func LoginAdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// init trace
		ctx, span := otel_trace.Start(c.Request.Context(), "LoginAdminRequired")
		defer span.End()

		user, _ := util.GetFromContext[*model.User](c, oauth.UserObjKey)

		if !user.IsAdmin {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error_msg": AdminRequired, "data": nil})
			return
		}

		// log
		logger.InfoF(ctx, "[LoginAdminRequired] %d %s", user.ID, user.Username)

		// next
		c.Next()
	}
}
