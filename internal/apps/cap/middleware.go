// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package cap

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Rain-kl/Wavelet/internal/common/response"
)

// VerifyMiddleware returns a Gin middleware that checks and consumes the X-Cap-Token header.
// enabledFunc is an optional callback allowing dynamic check of whether captcha protection is turned on.
func VerifyMiddleware(mgr *Manager, scope string, enabledFunc func() bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if enabledFunc != nil && !enabledFunc() {
			c.Next()
			return
		}

		token := c.GetHeader("X-Cap-Token")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Err(errCapTokenMissing))
			return
		}

		valid, err := mgr.VerifyToken(c.Request.Context(), token, scope)
		if err != nil || !valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Err(errCapTokenInvalidOrExpired))
			return
		}

		c.Next()
	}
}
