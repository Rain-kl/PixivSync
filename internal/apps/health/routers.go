// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

// Package health 提供健康检查端点
package health

import (
	"net/http"

	"github.com/Rain-kl/Wavelet/internal/util"
	"github.com/gin-gonic/gin"
)

// Health 健康检查
// @Summary 健康检查
// @Description 检查服务是否正常运行，可用于负载均衡存活探测
// @Tags health
// @Produce json
// @Success 200 {object} util.ResponseAny{data=string} "服务正常"
// @Router /api/v1/health [get]
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, util.OKNil())
}
