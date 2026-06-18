// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package pixez

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/testhelper"
)

func TestProcessMirrorIllustRegistersUpload(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	t.Chdir(t.TempDir())

	ctx := context.Background()
	seedMirrorTestUser(t, ctx)
	if err := db.DB(ctx).Create(&model.PixezPixivUser{
		PixivUserID:  "100",
		Name:         "Pixiv User",
		Account:      "pixiv_user",
		AccessToken:  "pixiv-token",
		RefreshToken: "pixiv-refresh",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}).Error; err != nil {
		t.Fatalf("seed pixiv user failed: %v", err)
	}
	if _, err := EnsureMirrorIllustQueued(ctx, 123, "task-1"); err != nil {
		t.Fatalf("EnsureMirrorIllustQueued() error = %v", err)
	}

	client := NewClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Host {
		case pixivAPIHost:
			if got := req.Header.Get("Authorization"); got != "Bearer pixiv-token" {
				t.Fatalf("Authorization = %q, want pixiv token", got)
			}
			return jsonResponse(http.StatusOK, `{"illust":{"id":123,"title":"mirror","meta_single_page":{"original_image_url":"https://i.pximg.net/img-original/img/2026/01/01/00/00/00/123_p0.jpg"}}}`), nil
		case "i.pximg.net":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"image/jpeg"}},
				Body:       io.NopCloser(strings.NewReader("jpeg-bytes")),
			}, nil
		default:
			t.Fatalf("unexpected request host: %s", req.URL.Host)
			return nil, nil
		}
	})})

	if err := ProcessMirrorIllust(ctx, client, "task-1", 123); err != nil {
		t.Fatalf("ProcessMirrorIllust() error = %v", err)
	}

	record, err := GetMirrorIllust(ctx, 123)
	if err != nil {
		t.Fatalf("GetMirrorIllust() error = %v", err)
	}
	if record.Status != model.PixezMirrorStatusSuccess || record.SuccessCount != 1 || record.TotalCount != 1 {
		t.Fatalf("unexpected mirror record: %+v", record)
	}

	var files []model.PixezMirrorImageFile
	if err := json.Unmarshal([]byte(record.ImageFilesJSON), &files); err != nil {
		t.Fatalf("decode image_files_json failed: %v", err)
	}
	if len(files) != 1 || files[0].UploadID == 0 || files[0].FileName != "123_p0.jpg" {
		t.Fatalf("unexpected image files: %+v", files)
	}

	var upload model.Upload
	if err := db.DB(ctx).Where("id = ?", files[0].UploadID).First(&upload).Error; err != nil {
		t.Fatalf("load upload failed: %v", err)
	}
	if upload.Type != pixezMirrorUploadType || upload.Status != model.UploadStatusUsed || upload.MimeType != "image/jpeg" {
		t.Fatalf("unexpected upload record: %+v", upload)
	}
	if _, err := os.Stat(upload.FilePath); err != nil {
		t.Fatalf("mirrored local file missing: %v", err)
	}

	trendDate := upload.CreatedAt.Format("2006-01-02")
	var trendStat model.UploadStat
	if err := db.DB(ctx).
		Where("dimension = ? AND stat_key = ?", model.UploadStatDimensionTrend, trendDate).
		First(&trendStat).Error; err != nil {
		t.Fatalf("load trend stat failed: %v", err)
	}
	if trendStat.FileCount != 1 || trendStat.FileSize != upload.FileSize {
		t.Fatalf("unexpected trend stat: %+v", trendStat)
	}
}

func TestFindMirroredImageUploadMapsDerivedFilenames(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()
	seedMirrorTestUser(t, ctx)
	upload := model.Upload{
		ID:            9001,
		UserID:        1001,
		FileName:      "123_p0.jpg",
		FilePath:      "uploads/2026/06/10/9001.jpg",
		FileSize:      10,
		MimeType:      "image/jpeg",
		Extension:     "jpg",
		Hash:          "hash",
		Type:          pixezMirrorUploadType,
		Status:        model.UploadStatusUsed,
		AccessMode:    1,
	}
	if err := db.DB(ctx).Create(&upload).Error; err != nil {
		t.Fatalf("seed upload failed: %v", err)
	}
	files := []model.PixezMirrorImageFile{{
		PixivURL: "https://i.pximg.net/img-original/img/2026/01/01/00/00/00/123_p0.jpg",
		Page:     0,
		UploadID: upload.ID,
		FileName: upload.FileName,
	}}
	if err := db.DB(ctx).Create(&model.PixezMirrorIllust{
		IllustID:        123,
		Status:          model.PixezMirrorStatusSuccess,
		ImageFilesJSON:  mustJSON(files),
		RequestURLsJSON: "[]",
		RetryURLsJSON:   "[]",
		SuccessCount:    1,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}).Error; err != nil {
		t.Fatalf("seed mirror illust failed: %v", err)
	}

	for _, pximgPath := range []string{
		"/img-original/img/2026/01/01/00/00/00/123_p0.jpg",
		"/c/600x1200_90/img-master/img/2026/01/01/00/00/00/123_p0_master1200.jpg",
		"/c/360x360_70/img-master/img/2026/01/01/00/00/00/123_p0_square1200.jpg",
	} {
		got, err := FindMirroredImageUpload(ctx, pximgPath)
		if err != nil {
			t.Fatalf("FindMirroredImageUpload(%q) error = %v", pximgPath, err)
		}
		if got.ID != upload.ID {
			t.Fatalf("FindMirroredImageUpload(%q) ID = %d, want %d", pximgPath, got.ID, upload.ID)
		}
	}
}

func seedMirrorTestUser(t *testing.T, ctx context.Context) {
	t.Helper()
	user := model.User{ID: 1001, Username: "admin", IsActive: true, IsAdmin: true}
	if err := db.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("seed user failed: %v", err)
	}
}
