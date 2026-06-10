// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package cap

import (
	"net/http"

	"github.com/Rain-kl/Wavelet/internal/util"
	caputil "github.com/Rain-kl/Wavelet/internal/util/cap"
	"github.com/gin-gonic/gin"
)

// VerifyMiddleware returns a Gin middleware that checks and consumes the X-Cap-Token header.
// enabledFunc is an optional callback allowing dynamic check of whether captcha protection is turned on.
func VerifyMiddleware(mgr *caputil.Manager, scope string, enabledFunc func() bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if enabledFunc != nil && !enabledFunc() {
			c.Next()
			return
		}

		token := c.GetHeader("X-Cap-Token")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, util.Err(errCapTokenMissing))
			return
		}

		valid, err := mgr.VerifyToken(c.Request.Context(), token, scope)
		if err != nil || !valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, util.Err(errCapTokenInvalidOrExpired))
			return
		}

		c.Next()
	}
}
