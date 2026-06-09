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

// Package config 提供公开配置查询接口
package config

import (
	"net/http"

	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/util"
	"github.com/gin-gonic/gin"
)

// GetPublicConfig 获取公共配置
// @Summary 获取公共配置
// @Description 返回系统配置表中 visibility 为 1 的配置键值集合
// @Tags config
// @Accept json
// @Produce json
// @Success 200 {object} util.ResponseAny
// @Router /api/v1/config/public [get]
func GetPublicConfig(c *gin.Context) {
	ctx := c.Request.Context()
	configs, err := model.ListVisibleSystemConfigs(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
		return
	}

	response := make(map[string]string, len(configs))
	for _, config := range configs {
		response[config.Key] = config.Value
	}

	c.JSON(http.StatusOK, util.OK(response))
}

// GetRobotsTXT 动态生成 robots.txt
// @Summary 获取 robots.txt
// @Description 根据系统配置决定是否允许搜索引擎检索，并返回相应的 robots.txt 文件内容
// @Tags config
// @Produce text/plain
// @Success 200 {string} string "robots.txt 内容"
// @Router /robots.txt [get]
func GetRobotsTXT(c *gin.Context) {
	ctx := c.Request.Context()
	enabled, err := model.GetBoolByKey(ctx, model.ConfigKeySearchEngineIndexingEnabled)
	content := "User-Agent: *\nDisallow: /\n"
	if err == nil && enabled {
		content = "User-Agent: *\nAllow: /\n"
	}
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(content))
}
