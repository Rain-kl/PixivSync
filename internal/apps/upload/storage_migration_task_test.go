// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package upload

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/storage"
	"github.com/Rain-kl/Wavelet/internal/testhelper"
)

func TestMigrationHandlerExecute(t *testing.T) {
	dbConn, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	sourceRoot := t.TempDir()
	sourcePath := filepath.Join(sourceRoot, "uploads", "test.txt")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0755); err != nil {
		t.Fatalf("MkdirAll(%q) returned error: %v", sourcePath, err)
	}
	const content = "storage migration"
	if err := os.WriteFile(sourcePath, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile(%q) returned error: %v", sourcePath, err)
	}

	ctx := context.Background()
	active := storage.DefaultConfig()
	active.Local.Root = sourceRoot
	if err := storage.SaveActiveConfig(ctx, active); err != nil {
		t.Fatalf("SaveActiveConfig() returned error: %v", err)
	}
	target := storage.DefaultConfig()
	target.Driver = storage.DriverS3
	target.S3 = storage.ObjectConfig{
		Region:          "us-east-1",
		Bucket:          "target",
		AccessKeyID:     "key",
		SecretAccessKey: "secret",
	}
	payload, err := json.Marshal(storageMigrationPayload{Target: target})
	if err != nil {
		t.Fatalf("Marshal(storageMigrationPayload) returned error: %v", err)
	}

	upload := model.Upload{
		ID:            99101,
		UserID:        1,
		FileName:      "test.txt",
		FilePath:      "uploads/test.txt",
		FileSize:      int64(len(content)),
		MimeType:      "text/plain",
		Extension:     "txt",
		Hash:          "hash",
		StorageDriver: string(storage.DriverLocal),
		Type:          "attachment",
		Status:        model.UploadStatusUsed,
	}
	if err := dbConn.Create(&upload).Error; err != nil {
		t.Fatalf("Create(upload) returned error: %v", err)
	}

	var copied bytes.Buffer
	restore := storage.MockStorage(
		func(_ context.Context, _ string, body io.Reader, _ int64, _ string) error {
			_, err := io.Copy(&copied, body)
			return err
		},
		func(context.Context, string) (*storage.Object, error) {
			return nil, nil
		},
		func(context.Context, string) error {
			return nil
		},
	)
	defer restore()

	result, err := (&MigrationHandler{}).Execute(ctx, payload)
	if err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}
	if result == nil {
		t.Fatal("Execute() result = nil, want non-nil")
	}
	if copied.String() != content {
		t.Errorf("migrated content = %q, want %q", copied.String(), content)
	}

	var migrated model.Upload
	if err := dbConn.First(&migrated, upload.ID).Error; err != nil {
		t.Fatalf("First(upload) returned error: %v", err)
	}
	if migrated.StorageDriver != string(storage.DriverS3) {
		t.Errorf("StorageDriver = %q, want %q", migrated.StorageDriver, storage.DriverS3)
	}
	current, err := storage.LoadConfig(ctx)
	if err != nil {
		t.Fatalf("LoadConfig() returned error: %v", err)
	}
	if current.Driver != storage.DriverS3 {
		t.Errorf("active driver = %q, want %q", current.Driver, storage.DriverS3)
	}
}
