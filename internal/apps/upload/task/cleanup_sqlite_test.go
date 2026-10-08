// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package task

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rain-kl/Wavelet/internal/apps/upload/ingest"
	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/repository"
	"github.com/Rain-kl/Wavelet/internal/storage"
	"github.com/Rain-kl/Wavelet/internal/testhelper"
)

func TestSystemCleanupWithColdStorageConfigAndOneSQLiteConnection(t *testing.T) {
	database, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	// This regression uses the DB-backed config lookup, without Redis listeners.
	previousRedis := db.Redis
	db.Redis = nil
	defer func() { db.Redis = previousRedis }()
	storage.ResetCache()
	defer storage.ResetCache()
	if err := database.AutoMigrate(&model.PushHistory{}); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	config := storage.DefaultConfig()
	config.Local.Root = root
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if err := storage.SaveActiveConfig(ctx, config); err != nil {
		t.Fatal(err)
	}
	result, err := ingest.Ingest(ctx, ingest.Request{
		UserID: 1001, Reader: strings.NewReader("test"), Size: 4,
		FileName: "test.txt", Extension: "txt", MimeType: "text/plain",
		Hash: "cleanup-regression", Type: "attachment", Policy: ingest.PolicyCreate,
		Status: model.UploadStatusPending, SkipExtensionCheck: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Model(&model.Upload{}).Where("id = ?", result.Upload.ID).
		Update("created_at", time.Now().Add(-2*time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	// Force storage.Active to load its configuration through the single SQL connection.
	storage.ResetCache()
	repository.ResetSystemConfigRAMCacheForTest()
	report, err := (&SystemCleanupHandler{}).Execute(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(report.Message, "成功清理未使用的上传文件 1/1 个") {
		t.Fatal(report.Message)
	}
	var upload model.Upload
	if err := database.First(&upload, "id = ?", result.Upload.ID).Error; err != nil {
		t.Fatal(err)
	}
	if upload.Status != model.UploadStatusDeleted {
		t.Fatal("cleanup did not mark the upload deleted")
	}
	if _, err := os.Stat(filepath.Join(root, upload.FilePath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cleanup did not remove the stored file: %v", err)
	}
}
