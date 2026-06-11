// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

// Package custom provides custom business handlers
package custom

import (
	"net/http"

	"github.com/Rain-kl/Wavelet/internal/util"
	"github.com/gin-gonic/gin"
)

// Hello is a sample handler for custom business logic
// @Summary Sample Hello API
// @Description A sample business API for customization
// @Tags custom
// @Produce json
// @Success 200 {object} util.ResponseAny{data=string} "成功"
// @Router /api/v1/custom/hello [get]
func Hello(c *gin.Context) {
	c.JSON(http.StatusOK, util.OK("Hello from custom business module!"))
}
