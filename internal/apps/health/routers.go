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
