// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package upload

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rain-kl/Wavelet/internal/common"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/testhelper"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func TestServeFileByIDAccessControl(t *testing.T) {
	dbConn, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	// Ensure uploads dir is cleaned up
	defer func() { _ = os.RemoveAll("uploads") }()

	// Create a user in DB
	user := model.User{
		ID:       12345,
		Username: "file_test_user",
		IsActive: true,
	}
	if err := dbConn.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create an access token for this user
	tokenStr := "test-secret-token-123"
	tokenHash := model.HashToken(tokenStr)
	tokenRecord := model.AccessToken{
		UserID:    user.ID,
		Name:      "test_token",
		TokenHash: tokenHash,
	}
	if err := dbConn.Create(&tokenRecord).Error; err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	// Create two files: one in whitelist (avatar), one not in whitelist (attachment)
	avatarFile := model.Upload{
		ID:            8001,
		UserID:        user.ID,
		FileName:      "avatar.png",
		FilePath:      "uploads/avatar.png",
		FileSize:      5,
		MimeType:      "image/png",
		Extension:     "png",
		StorageDriver: "local",
		Type:          "avatar",
		Status:        model.UploadStatusUsed,
	}
	attachmentFile := model.Upload{
		ID:            8002,
		UserID:        user.ID,
		FileName:      "doc.pdf",
		FilePath:      "uploads/doc.pdf",
		FileSize:      5,
		MimeType:      "application/pdf",
		Extension:     "pdf",
		StorageDriver: "local",
		Type:          "attachment",
		Status:        model.UploadStatusUsed,
	}

	_ = os.MkdirAll("uploads", 0755)
	_ = os.WriteFile(avatarFile.FilePath, []byte("image"), 0644)
	_ = os.WriteFile(attachmentFile.FilePath, []byte("bytes"), 0644)

	dbConn.Create(&avatarFile)
	dbConn.Create(&attachmentFile)

	// Set up router
	gin.SetMode(gin.TestMode)
	r := gin.New()
	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("test_session", store))
	r.GET("/f/:id", ServeFileByID)

	t.Run("whitelisted file type (avatar) accessed without authentication", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/f/8001", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
		}
		if w.Body.String() != "image" {
			t.Errorf("expected 'image', got %q", w.Body.String())
		}
	})

	t.Run("non-whitelisted file type (attachment) accessed without authentication returns 401", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/f/8002", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d. Body: %s", w.Code, w.Body.String())
		}

		var body map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}
		if body["error_msg"] != common.UnAuthorized {
			t.Errorf("expected error_msg %q, got %v", common.UnAuthorized, body["error_msg"])
		}
	})

	t.Run("non-whitelisted file type (attachment) accessed with valid token succeeds", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/f/8002", nil)
		req.Header.Set("X-Access-Token", tokenStr)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
		}
		if w.Body.String() != "bytes" {
			t.Errorf("expected 'bytes', got %q", w.Body.String())
		}
	})

	t.Run("accessing non-existent file returns 404", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/f/9999", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", w.Code)
		}
	})
}

func TestGetDistinctUploadTypes(t *testing.T) {
	dbConn, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	// Seed some uploads with new custom types
	user := model.User{ID: 2222, Username: "test_user_2"}
	dbConn.Create(&user)

	customUpload := model.Upload{
		ID:            9001,
		UserID:        user.ID,
		FileName:      "custom.txt",
		FilePath:      "uploads/custom.txt",
		FileSize:      10,
		MimeType:      "text/plain",
		Extension:     "txt",
		StorageDriver: "local",
		Type:          "custom_type_xyz",
		Status:        model.UploadStatusUsed,
	}
	dbConn.Create(&customUpload)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/admin/uploads/types", GetDistinctUploadTypes)

	req, _ := http.NewRequest("GET", "/api/v1/admin/uploads/types", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		ErrorMsg string   `json:"error_msg"`
		Data     []string `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if resp.ErrorMsg != "" {
		t.Fatalf("unexpected error: %s", resp.ErrorMsg)
	}

	// Verify that only custom_type_xyz is present
	if len(resp.Data) != 1 || resp.Data[0] != "custom_type_xyz" {
		t.Errorf("expected only custom_type_xyz in types list, got: %v", resp.Data)
	}
}

func TestImageCompression(t *testing.T) {
	dbConn, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	// Ensure uploads dir is cleaned up
	defer func() {
		_ = os.RemoveAll("uploads")
	}()

	// Create test user
	user := model.User{
		ID:       555,
		Username: "compress_tester",
		IsActive: true,
	}
	dbConn.Create(&user)

	// Create a 1x1 pixel PNG image
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		t.Fatalf("failed to encode test png: %v", err)
	}

	_ = os.MkdirAll("uploads", 0755)
	filePath := "uploads/test_image.png"
	if err := os.WriteFile(filePath, pngBuf.Bytes(), 0644); err != nil {
		t.Fatalf("failed to write test png: %v", err)
	}

	// Save upload record to DB
	uploadRecord := model.Upload{
		ID:            3001,
		UserID:        user.ID,
		FileName:      "test_image.png",
		FilePath:      filePath,
		FileSize:      int64(pngBuf.Len()),
		MimeType:      "image/png",
		Extension:     "png",
		StorageDriver: "local",
		Type:          "avatar", // Whitelisted by default
		Status:        model.UploadStatusUsed,
	}
	dbConn.Create(&uploadRecord)

	// Setup Router
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/f/:id", ServeFileByID)

	t.Run("serve original file without compress parameter", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/f/3001", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
		// Content-Type should be image/png (default local serving type)
		if w.Header().Get("Content-Type") != "image/png" {
			t.Errorf("expected Content-Type image/png, got %s", w.Header().Get("Content-Type"))
		}
		if len(w.Body.Bytes()) != pngBuf.Len() {
			t.Errorf("expected body size %d, got %d", pngBuf.Len(), len(w.Body.Bytes()))
		}
	})

	t.Run("serve compressed WebP file with compress=true", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/f/3001?compress=true&level=medium", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}
		// Content-Type should be image/webp
		if w.Header().Get("Content-Type") != "image/webp" {
			t.Errorf("expected Content-Type image/webp, got %s", w.Header().Get("Content-Type"))
		}

		// Check if local cache file was created
		cachePath := filepath.Join("uploads", "cache", "compressed_3001_medium.webp")
		if _, err := os.Stat(cachePath); os.IsNotExist(err) {
			t.Errorf("expected cached webp file to be created at %s, but it doesn't exist", cachePath)
		}

		// Subsequent request should hit the cache (modify the cached file to verify)
		testBytes := []byte("cached webp content")
		if err := os.WriteFile(cachePath, testBytes, 0644); err != nil {
			t.Fatalf("failed to write test bytes to cache: %v", err)
		}

		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, req)
		if w2.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w2.Code)
		}
		if string(w2.Body.Bytes()) != "cached webp content" {
			t.Errorf("expected cached content, got %s", string(w2.Body.Bytes()))
		}
	})

	t.Run("serve compressed with default quality high", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/f/3001?compress=true", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
		// Check if local cache file with "high" was created
		cachePath := filepath.Join("uploads", "cache", "compressed_3001_high.webp")
		if _, err := os.Stat(cachePath); os.IsNotExist(err) {
			t.Errorf("expected cached webp file to be created at %s for default level", cachePath)
		}
	})
}
