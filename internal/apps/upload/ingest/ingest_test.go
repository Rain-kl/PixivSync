// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package ingest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"testing"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/storage"
	"github.com/Rain-kl/Wavelet/internal/testhelper"
)

func TestIngestPolicyCreateIncrementsStats(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	content := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01")
	hash := sha256.Sum256(content)

	restoreStorage, disableStorage := setupMockStorage(t, nil)
	defer restoreStorage()
	defer disableStorage()

	result, err := Ingest(ctx, Request{
		UserID:    1001,
		Reader:    bytes.NewReader(content),
		Size:      int64(len(content)),
		FileName:  "mirror.png",
		MimeType:  "image/png",
		Extension: "png",
		Hash:      hex.EncodeToString(hash[:]),
		Type:      "pixez_mirror",
		Policy:    PolicyCreate,
	})
	if err != nil {
		t.Fatalf("Ingest(PolicyCreate) returned error: %v", err)
	}
	if !result.Created || !result.Stored || result.Resolved {
		t.Fatalf("Ingest(PolicyCreate) = %+v, want Created+Stored without Resolved", result)
	}

	stats, err := loadTotalStats(ctx)
	if err != nil {
		t.Fatalf("loadTotalStats returned error: %v", err)
	}
	if stats.TotalCount != 1 || stats.TotalSize != int64(len(content)) {
		t.Fatalf("loadTotalStats() = count %d size %d, want count 1 size %d", stats.TotalCount, stats.TotalSize, len(content))
	}
}

func TestIngestPolicyResolveExistingSkipsStatsOnHit(t *testing.T) {
	dbConn, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	content := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01")
	hash := sha256.Sum256(content)
	hashStr := hex.EncodeToString(hash[:])

	existing := model.Upload{
		ID:        88001,
		UserID:    42,
		FileName:  "existing.png",
		FilePath:  "uploads/existing.png",
		FileSize:  int64(len(content)),
		MimeType:  "image/png",
		Extension: "png",
		Hash:      hashStr,
		Type:      "pixez_mirror",
		Status:    model.UploadStatusUsed,
		CreatedAt: time.Now(),
	}
	if err := dbConn.Create(&existing).Error; err != nil {
		t.Fatalf("seed upload failed: %v", err)
	}

	restoreStorage, disableStorage := setupMockStorage(t, nil)
	defer restoreStorage()
	defer disableStorage()

	result, err := Ingest(ctx, Request{
		UserID:    1001,
		Reader:    bytes.NewReader(content),
		Size:      int64(len(content)),
		FileName:  "mirror.png",
		MimeType:  "image/png",
		Extension: "png",
		Hash:      hashStr,
		Type:      "pixez_mirror",
		Policy:    PolicyResolveExisting,
	})
	if err != nil {
		t.Fatalf("Ingest(PolicyResolveExisting) returned error: %v", err)
	}
	if !result.Resolved || result.Created || result.Stored {
		t.Fatalf("Ingest(PolicyResolveExisting) = %+v, want Resolved only", result)
	}
	if result.Upload.ID != existing.ID {
		t.Fatalf("Ingest(PolicyResolveExisting).Upload.ID = %d, want %d", result.Upload.ID, existing.ID)
	}

	stats, err := loadTotalStats(ctx)
	if err != nil {
		t.Fatalf("loadTotalStats returned error: %v", err)
	}
	if stats.TotalCount != 0 || stats.TotalSize != 0 {
		t.Fatalf("loadTotalStats() = count %d size %d, want zero stats for resolved upload", stats.TotalCount, stats.TotalSize)
	}
}

func TestIngestPolicyDedupNewRecordCreatesSecondRecord(t *testing.T) {
	dbConn, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	content := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01")
	hash := sha256.Sum256(content)
	hashStr := hex.EncodeToString(hash[:])
	putCount := 0

	restoreStorage, disableStorage := setupMockStorage(t, &putCount)
	defer restoreStorage()
	defer disableStorage()

	first, err := Ingest(ctx, Request{
		UserID:    1001,
		Reader:    bytes.NewReader(content),
		Size:      int64(len(content)),
		FileName:  "first.png",
		MimeType:  "image/png",
		Extension: "png",
		Hash:      hashStr,
		Type:      "avatar",
		Policy:    PolicyDedupNewRecord,
	})
	if err != nil {
		t.Fatalf("first Ingest returned error: %v", err)
	}
	if putCount != 1 {
		t.Fatalf("putCount after first ingest = %d, want 1", putCount)
	}

	second, err := Ingest(ctx, Request{
		UserID:    1002,
		Reader:    bytes.NewReader(content),
		Size:      int64(len(content)),
		FileName:  "second.png",
		MimeType:  "image/png",
		Extension: "png",
		Hash:      hashStr,
		Type:      "avatar",
		Policy:    PolicyDedupNewRecord,
	})
	if err != nil {
		t.Fatalf("second Ingest returned error: %v", err)
	}
	if putCount != 1 {
		t.Fatalf("putCount after dedup ingest = %d, want 1", putCount)
	}
	if first.Upload.FilePath != second.Upload.FilePath {
		t.Fatalf("dedup file paths differ: %s vs %s", first.Upload.FilePath, second.Upload.FilePath)
	}
	if first.Upload.ID == second.Upload.ID {
		t.Fatal("dedup records should have unique IDs")
	}

	var count int64
	if err := dbConn.Model(&model.Upload{}).Where("hash = ?", hashStr).Count(&count).Error; err != nil {
		t.Fatalf("count uploads failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("upload count = %d, want 2", count)
	}
}

func TestRemoveDecrementsStats(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	content := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01")
	hash := sha256.Sum256(content)

	restoreStorage, disableStorage := setupMockStorage(t, nil)
	defer restoreStorage()
	defer disableStorage()

	result, err := Ingest(ctx, Request{
		UserID:    1001,
		Reader:    bytes.NewReader(content),
		Size:      int64(len(content)),
		FileName:  "delete-me.png",
		MimeType:  "image/png",
		Extension: "png",
		Hash:      hex.EncodeToString(hash[:]),
		Type:      "generic",
		Policy:    PolicyCreate,
	})
	if err != nil {
		t.Fatalf("Ingest returned error: %v", err)
	}

	if _, err := Remove(ctx, result.Upload.ID); err != nil {
		t.Fatalf("Remove(%d) returned error: %v", result.Upload.ID, err)
	}

	stats, err := loadTotalStats(ctx)
	if err != nil {
		t.Fatalf("loadTotalStats returned error: %v", err)
	}
	if stats.TotalCount != 0 || stats.TotalSize != 0 {
		t.Fatalf("loadTotalStats() after remove = count %d size %d, want zero", stats.TotalCount, stats.TotalSize)
	}
}

type totalStatsSnapshot struct {
	TotalCount int64
	TotalSize  int64
}

func loadTotalStats(ctx context.Context) (totalStatsSnapshot, error) {
	var rows []model.UploadStat
	if err := db.DB(ctx).Where("dimension = ?", model.UploadStatDimensionTotal).Find(&rows).Error; err != nil {
		return totalStatsSnapshot{}, err
	}
	if len(rows) == 0 {
		return totalStatsSnapshot{}, nil
	}
	return totalStatsSnapshot{
		TotalCount: rows[0].FileCount,
		TotalSize:  rows[0].FileSize,
	}, nil
}

func setupMockStorage(t *testing.T, putCount *int) (restore func(), disable func()) {
	t.Helper()
	mockFiles := make(map[string][]byte)
	restore = storage.MockStorage(
		func(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
			data, err := io.ReadAll(body)
			if err != nil {
				return err
			}
			mockFiles[key] = data
			if putCount != nil {
				*putCount++
			}
			return nil
		},
		func(ctx context.Context, key string) (*storage.Object, error) {
			data, ok := mockFiles[key]
			if !ok {
				return nil, os.ErrNotExist
			}
			return &storage.Object{
				Body:          io.NopCloser(bytes.NewReader(data)),
				ContentLength: int64(len(data)),
				ContentType:   "application/octet-stream",
			}, nil
		},
		func(ctx context.Context, key string) error {
			delete(mockFiles, key)
			return nil
		},
	)
	storage.IsEnabledFunc = func() bool { return true }
	storage.ResetCache()
	disable = func() {
		storage.IsEnabledFunc = func() bool { return false }
		storage.ResetCache()
	}
	return restore, disable
}