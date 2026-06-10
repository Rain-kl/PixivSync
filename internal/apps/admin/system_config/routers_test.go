// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package system_config

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/Rain-kl/Wavelet/internal/apps/oauth"
	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/testhelper"
	"github.com/Rain-kl/Wavelet/internal/util"
	"github.com/gin-gonic/gin"
)

func setupTestRouter(authUser *model.User) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	adminGroup := r.Group("/api/v1/admin")

	// Mock authentication middleware
	adminGroup.Use(func(c *gin.Context) {
		if authUser != nil {
			util.SetToContext(c, oauth.UserObjKey, authUser)
		}
		c.Next()
	})

	adminGroup.POST("/system-configs", CreateSystemConfig)
	adminGroup.GET("/system-configs", ListSystemConfigs)

	systemConfigRouter := adminGroup.Group("/system-configs/:key")
	{
		systemConfigRouter.GET("", GetSystemConfig)
		systemConfigRouter.PUT("", UpdateSystemConfig)
	}

	return r
}

func TestCreateSystemConfig(t *testing.T) {
	dbConn, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	adminUser := &model.User{ID: 1001, Username: "admin", IsAdmin: true}
	router := setupTestRouter(adminUser)

	t.Run("create successfully", func(t *testing.T) {
		payload := CreateSystemConfigRequest{
			Key:         "custom_key",
			Value:       "custom_value",
			Type:        "system",
			Visibility:  model.ConfigVisibilityVisible,
			Description: "desc",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/v1/admin/system-configs", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		// Verify database
		var cfg model.SystemConfig
		err := dbConn.Where("key = ?", "custom_key").First(&cfg).Error
		if err != nil {
			t.Fatalf("failed to find system config in DB: %v", err)
		}

		// Verify Redis Cache
		var redisConfig model.SystemConfig
		err = db.HGetJSON(context.Background(), model.SystemConfigRedisHashKey, "custom_key", &redisConfig)
		if err != nil {
			t.Fatalf("failed to find system config in Redis: %v", err)
		}
		if redisConfig.Value != "custom_value" {
			t.Errorf("CreateSystemConfig(custom_key).Value = %q, want %q", redisConfig.Value, "custom_value")
		}
		if redisConfig.Visibility != model.ConfigVisibilityVisible {
			t.Errorf("CreateSystemConfig(custom_key).Visibility = %d, want %d", redisConfig.Visibility, model.ConfigVisibilityVisible)
		}
	})

	t.Run("create duplicate key error", func(t *testing.T) {
		// Key "custom_key" already exists from previous test
		payload := CreateSystemConfigRequest{
			Key:         "custom_key",
			Value:       "another_value",
			Type:        "system",
			Description: "desc",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/v1/admin/system-configs", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request on duplicate key, got %d", w.Code)
		}
	})
}

func TestListSystemConfigs(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	adminUser := &model.User{ID: 1001, Username: "admin", IsAdmin: true}
	router := setupTestRouter(adminUser)

	t.Run("list all seeded configurations", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/admin/system-configs", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}

		var resp util.ResponseAny
		_ = json.Unmarshal(w.Body.Bytes(), &resp)

		dataBytes, _ := json.Marshal(resp.Data)
		var configs []model.SystemConfig
		_ = json.Unmarshal(dataBytes, &configs)

		const expectedDefaultConfigCount = 26
		if len(configs) != expectedDefaultConfigCount {
			t.Errorf("expected %d default configs, got %d", expectedDefaultConfigCount, len(configs))
		}
	})

	t.Run("filter by type business", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/admin/system-configs?type=business", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var resp util.ResponseAny
		_ = json.Unmarshal(w.Body.Bytes(), &resp)

		dataBytes, _ := json.Marshal(resp.Data)
		var configs []model.SystemConfig
		_ = json.Unmarshal(dataBytes, &configs)

		expectedKeys := map[string]bool{
			model.ConfigKeyMaxAPIKeysPerUser:            true,
			model.ConfigKeyPixezMirrorDownloadInterval:  true,
			model.ConfigKeyPixezMirrorIllustConcurrency: true,
			model.ConfigKeyPixezMirrorNovelConcurrency:  true,
		}
		for _, config := range configs {
			delete(expectedKeys, config.Key)
		}
		if len(expectedKeys) != 0 {
			t.Errorf("missing business configs: %v; got %v", expectedKeys, configs)
		}
		const expectedBusinessConfigCount = 4
		if len(configs) != expectedBusinessConfigCount {
			t.Errorf("expected %d business configs, got %d: %v", expectedBusinessConfigCount, len(configs), configs)
		}
	})
}

func TestGetSystemConfig(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	adminUser := &model.User{ID: 1001, Username: "admin", IsAdmin: true}
	router := setupTestRouter(adminUser)

	t.Run("get existing configuration", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/admin/system-configs/"+model.ConfigKeySiteName, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", w.Code)
		}

		var resp util.ResponseAny
		_ = json.Unmarshal(w.Body.Bytes(), &resp)

		dataBytes, _ := json.Marshal(resp.Data)
		var cfg model.SystemConfig
		_ = json.Unmarshal(dataBytes, &cfg)

		if cfg.Value != "Wavelet" {
			t.Errorf("expected 'Wavelet', got '%s'", cfg.Value)
		}
	})

	t.Run("get non-existent config", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/admin/system-configs/non_existent_key", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found, got %d", w.Code)
		}
	})
}

func TestUpdateSystemConfig(t *testing.T) {
	dbConn, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	adminUser := &model.User{ID: 1001, Username: "admin", IsAdmin: true}
	router := setupTestRouter(adminUser)

	t.Run("update successfully", func(t *testing.T) {
		hidden := model.ConfigVisibilityHidden
		payload := UpdateSystemConfigRequest{
			Value:       "Super Site Name",
			Visibility:  &hidden,
			Description: "Updated Description",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("PUT", "/api/v1/admin/system-configs/"+model.ConfigKeySiteName, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		// Verify database
		var cfg model.SystemConfig
		dbConn.Where("key = ?", model.ConfigKeySiteName).First(&cfg)
		if cfg.Value != "Super Site Name" || cfg.Description != "Updated Description" || cfg.Visibility != model.ConfigVisibilityHidden {
			t.Errorf("database values not updated: %+v", cfg)
		}

		// Verify Redis
		var redisConfig model.SystemConfig
		_ = db.HGetJSON(context.Background(), model.SystemConfigRedisHashKey, model.ConfigKeySiteName, &redisConfig)
		if redisConfig.Value != "Super Site Name" {
			t.Errorf("redis cache value not updated, got '%s'", redisConfig.Value)
		}
		if redisConfig.Visibility != model.ConfigVisibilityHidden {
			t.Errorf("redis cache visibility = %d, want %d", redisConfig.Visibility, model.ConfigVisibilityHidden)
		}
	})

	t.Run("update non-existent config", func(t *testing.T) {
		payload := UpdateSystemConfigRequest{
			Value: "New Value",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("PUT", "/api/v1/admin/system-configs/invalid_key", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found, got %d", w.Code)
		}
	})
}

func TestTestSMTP(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	adminUser := &model.User{ID: 1001, Username: "admin", IsAdmin: true}
	r := setupTestRouter(adminUser)
	r.POST("/api/v1/admin/system-configs/smtp/test", TestSMTP)

	// Start a mock SMTP server
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start mock smtp server: %v", err)
	}
	defer func() { _ = l.Close() }()

	port := l.Addr().(*net.TCPAddr).Port

	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		writer := bufio.NewWriter(conn)
		reader := bufio.NewReader(conn)
		tp := textproto.NewReader(reader)

		// 220 Ready
		_, _ = writer.WriteString("220 mock.smtp.com SMTP Ready\r\n")
		_ = writer.Flush()

		// Read HELO/EHLO
		_, _ = tp.ReadLine()
		_, _ = writer.WriteString("250-mock.smtp.com\r\n250 AUTH PLAIN\r\n")
		_ = writer.Flush()

		// Read AUTH PLAIN
		_, _ = tp.ReadLine()
		_, _ = writer.WriteString("235 Authentication successful\r\n")
		_ = writer.Flush()

		// Read MAIL FROM
		_, _ = tp.ReadLine()
		_, _ = writer.WriteString("250 OK\r\n")
		_ = writer.Flush()

		// Read RCPT TO
		_, _ = tp.ReadLine()
		_, _ = writer.WriteString("250 OK\r\n")
		_ = writer.Flush()

		// Read DATA
		_, _ = tp.ReadLine()
		_, _ = writer.WriteString("354 Start mail input\r\n")
		_ = writer.Flush()

		// Read body lines until dot
		for {
			line, err := tp.ReadLine()
			if err != nil || line == "." {
				break
			}
		}
		_, _ = writer.WriteString("250 OK\r\n")
		_ = writer.Flush()

		// Read QUIT
		_, _ = tp.ReadLine()
		_, _ = writer.WriteString("221 Bye\r\n")
		_ = writer.Flush()
	}()

	payload := TestSMTPRequest{
		SMTPHost:     "127.0.0.1",
		SMTPPort:     port,
		SMTPUsername: "sender@example.com",
		SMTPPassword: "password",
		To:           "recipient@example.com",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/v1/admin/system-configs/smtp/test", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp util.ResponseAny
	json.Unmarshal(w.Body.Bytes(), &resp)

	dataBytes, _ := json.Marshal(resp.Data)
	var testResp TestSMTPResponse
	json.Unmarshal(dataBytes, &testResp)

	if !testResp.Success {
		t.Errorf("expected test success, got failed: %s. Log: %s", testResp.Error, testResp.Log)
	}
}
