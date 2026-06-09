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

package router

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Rain-kl/Wavelet/internal/config"
	"github.com/Rain-kl/Wavelet/internal/logger"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/otel_trace"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 初始化 Trace
		ctx, span := otel_trace.Start(c.Request.Context(), "LoggerMiddleware")
		defer span.End()

		// 开始计时
		start := time.Now()

		// 记录请求路径和 Query
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		if raw != "" {
			path = path + "?" + raw
		}

		// 执行请求
		c.Next()

		// 停止计时
		end := time.Now()
		latency := end.Sub(start)

		// 打印日志
		// 排除健康检查接口
		healthPath := config.Config.App.APIPrefix + "/v1/health"
		if c.Request.URL.Path != healthPath {
			logger.InfoF(
				ctx,
				"[LoggerMiddleware] %s %s\nStartTime: %s\nEndTime: %s\nLatency: %d\nClientIP: %s\nResponse: %d %d",
				c.Request.Method,
				path,
				start.Format(time.RFC3339),
				end.Format(time.RFC3339),
				latency.Milliseconds(),
				c.ClientIP(),
				c.Writer.Status(),
				c.Writer.Size(),
			)
		}

		// 设置 Span 状态
		if c.Writer.Status() >= http.StatusBadRequest {
			span := trace.SpanFromContext(ctx)
			span.SetStatus(codes.Error, strconv.Itoa(c.Writer.Status()))
		}
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			var sc model.SystemConfig
			// Fetch from system config. We use request context which supports trace
			if err := sc.GetByKey(c.Request.Context(), model.ConfigKeyServerAddress); err == nil && sc.Value != "" {
				c.Writer.Header().Set("Access-Control-Allow-Origin", sc.Value)
			} else {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			}
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Access-Token, X-Cap-Token")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
