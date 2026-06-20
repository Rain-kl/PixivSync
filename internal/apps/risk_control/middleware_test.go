// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package risk_control

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Rain-kl/Wavelet/internal/apps/oauth"
	"github.com/Rain-kl/Wavelet/internal/config"
	"github.com/Rain-kl/Wavelet/internal/db/batchwriter"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/model/analytics"
	"github.com/Rain-kl/Wavelet/internal/testhelper"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func newTestAccessLogWriter(t *testing.T, cfg batchwriter.Config) (*batchwriter.Writer[*analytics.UserAccessLog], func() []*analytics.UserAccessLog) {
	t.Helper()

	var (
		mu       sync.Mutex
		captured []*analytics.UserAccessLog
	)
	writer, err := batchwriter.New(cfg, func(_ context.Context, items []*analytics.UserAccessLog) error {
		mu.Lock()
		captured = append(captured, items...)
		mu.Unlock()
		return nil
	})
	if err != nil {
		t.Fatalf("batchwriter.New() error = %v", err)
	}

	writer.Start(context.Background())
	restore := SetLogWriterForTest(writer)
	t.Cleanup(func() {
		restore()
		stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = writer.Stop(stopCtx)
	})

	return writer, func() []*analytics.UserAccessLog {
		mu.Lock()
		defer mu.Unlock()
		return append([]*analytics.UserAccessLog(nil), captured...)
	}
}

func drainAccessLogWriter(t *testing.T, writer *batchwriter.Writer[*analytics.UserAccessLog]) {
	t.Helper()

	stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := writer.Stop(stopCtx); err != nil {
		t.Fatalf("writer.Stop() error = %v", err)
	}
}

func TestRiskControlMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("ClickHouse disabled", func(t *testing.T) {
		config.Config.ClickHouse.Enabled = false
		defer func() { config.Config.ClickHouse.Enabled = false }()

		r := testhelper.NewTestGinEngine(RiskControlMiddleware())
		r.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "ok", w.Body.String())
	})

	t.Run("ClickHouse enabled - Normal Authenticated Request", func(t *testing.T) {
		config.Config.ClickHouse.Enabled = true
		defer func() { config.Config.ClickHouse.Enabled = false }()

		cfg := batchwriter.DefaultConfig()
		cfg.MaxBatchSize = 100
		cfg.FlushInterval = time.Hour

		writer, getCaptured := newTestAccessLogWriter(t, cfg)

		r := gin.New()
		r.Use(func(c *gin.Context) {
			user := &model.User{ID: 12345}
			oauth.SetToContext(c, oauth.UserObjKey, user)
			c.Next()
		})
		r.Use(RiskControlMiddleware())
		r.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Test-Header", "hello")
		req.Header.Set("Cookie", "session_id=abcdef123456")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "ok", w.Body.String())

		drainAccessLogWriter(t, writer)

		captured := getCaptured()
		if len(captured) != 1 {
			t.Fatalf("captured access logs = %d, want 1", len(captured))
		}
		logItem := captured[0]
		assert.Equal(t, uint64(12345), logItem.UserID)
		assert.Equal(t, "/test", logItem.Path)
		assert.Equal(t, http.MethodGet, logItem.Method)
		assert.Equal(t, int32(http.StatusOK), logItem.Status)
		assert.NotEmpty(t, logItem.Headers)
		assert.Contains(t, logItem.Headers, "X-Test-Header")
		assert.NotContains(t, logItem.Headers, "Cookie")
	})

	t.Run("ClickHouse enabled - Unauthenticated Request", func(t *testing.T) {
		config.Config.ClickHouse.Enabled = true
		defer func() { config.Config.ClickHouse.Enabled = false }()

		cfg := batchwriter.DefaultConfig()
		cfg.MaxBatchSize = 100
		cfg.FlushInterval = time.Hour

		writer, getCaptured := newTestAccessLogWriter(t, cfg)

		r := testhelper.NewTestGinEngine(RiskControlMiddleware())
		r.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "ok", w.Body.String())

		drainAccessLogWriter(t, writer)

		if len(getCaptured()) != 0 {
			t.Fatal("expected no log item for unauthenticated request")
		}
	})

	t.Run("ClickHouse enabled - Buffer Full Rate Limiting", func(t *testing.T) {
		config.Config.ClickHouse.Enabled = true
		defer func() { config.Config.ClickHouse.Enabled = false }()

		cfg := batchwriter.DefaultConfig()
		cfg.QueueSize = 2
		cfg.MaxBatchSize = 100
		cfg.FlushInterval = time.Hour

		writer, _ := newTestAccessLogWriter(t, cfg)

		for range cfg.QueueSize {
			writer.TryEnqueue(&analytics.UserAccessLog{})
		}
		if !IsBufferFull() {
			t.Fatal("IsBufferFull() = false, want true")
		}

		r := testhelper.NewTestGinEngine(RiskControlMiddleware())
		r.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp["error_msg"], "系统繁忙")
	})
}
