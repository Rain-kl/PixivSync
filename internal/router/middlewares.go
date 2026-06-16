// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

// Package router 提供 HTTP 路由注册与服务启动
package router

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Rain-kl/Wavelet/internal/common/response"
	"github.com/Rain-kl/Wavelet/internal/config"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/pkg/logger"
	otel_trace "github.com/Rain-kl/Wavelet/pkg/trace"
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
		healthPath := config.Config.App.APIPrefix + "/health"
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

func isOriginAllowed(ctx context.Context, origin string) bool {
	var sc model.SystemConfig
	if err := sc.GetByKey(ctx, model.ConfigKeyServerAddress); err != nil || sc.Value == "" {
		return false
	}
	allowedOrigins := strings.Split(sc.Value, ",")
	for _, allowed := range allowedOrigins {
		allowed = strings.TrimRight(strings.TrimSpace(allowed), "/")
		if allowed != "" && strings.EqualFold(allowed, origin) {
			return true
		}
	}
	return false
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" && isOriginAllowed(c.Request.Context(), origin) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
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

// errorHandlerMiddleware 捕获 c.Errors 并统一格式化为 JSON 返回给客户端，同时将其记录到 Span 异常中
func errorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			span := trace.SpanFromContext(c.Request.Context())

			// 1. 如果有活跃的 Span，将错误信息记录到 Trace 中，并把 Span 状态置为 Error
			if span.IsRecording() {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			}

			// 2. 将错误转化为统一的 JSON 格式响应给客户端
			var apiErr *response.APIError
			if errors.As(err, &apiErr) {
				c.JSON(apiErr.Code, response.Err(apiErr.Msg))
			} else {
				// 兜底策略：未知的系统级错误
				c.JSON(http.StatusInternalServerError, response.Err("内部系统错误"))
			}
		}
	}
}
