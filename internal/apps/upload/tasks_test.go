/*
Copyright 2025-2026 linux.do
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

package upload

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/storage"
	"github.com/Rain-kl/Wavelet/internal/task"
	"github.com/Rain-kl/Wavelet/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanupUnusedUploadsHandler_Execute(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	// Mock S3 存储（让 DeleteObject 总是成功）
	storageMock := storage.MockStorage(
		func(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
			return nil
		},
		func(ctx context.Context, key string) (*storage.ObjectInfo, error) { return nil, nil },
		func(ctx context.Context, key string) error { return nil },
	)
	defer storageMock()

	ctx := context.Background()

	// 准备测试数据：创建一些上传记录
	now := time.Now()
	twoHoursAgo := now.Add(-2 * time.Hour)

	records := []*model.Upload{
		// 超过1小时且状态为 pending 的记录 —— 应被清理
		{
			UserID: 1001, FileName: "old_file_1.jpg", FilePath: "uploads/old_1.jpg",
			FileSize: 1024, MimeType: "image/jpeg", Extension: "jpg", Hash: "hash1",
			StorageDriver: "s3", Type: "attachment", Status: model.UploadStatusPending,
			CreatedAt: twoHoursAgo,
		},
		{
			UserID: 1001, FileName: "old_file_2.png", FilePath: "uploads/old_2.png",
			FileSize: 2048, MimeType: "image/png", Extension: "png", Hash: "hash2",
			StorageDriver: "s3", Type: "attachment", Status: model.UploadStatusPending,
			CreatedAt: twoHoursAgo,
		},
		// 状态为 used 的记录 —— 不应被清理
		{
			UserID: 1001, FileName: "used_file.jpg", FilePath: "uploads/used.jpg",
			FileSize: 512, MimeType: "image/jpeg", Extension: "jpg", Hash: "hash3",
			StorageDriver: "s3", Type: "attachment", Status: model.UploadStatusUsed,
			CreatedAt: twoHoursAgo,
		},
		// 不到1小时的 pending 记录 —— 不应被清理
		{
			UserID: 1001, FileName: "recent_file.jpg", FilePath: "uploads/recent.jpg",
			FileSize: 256, MimeType: "image/jpeg", Extension: "jpg", Hash: "hash4",
			StorageDriver: "s3", Type: "attachment", Status: model.UploadStatusPending,
			CreatedAt: now.Add(-10 * time.Minute),
		},
	}
	for _, r := range records {
		err := db.DB(ctx).Create(r).Error
		require.NoError(t, err)
	}

	// 执行 handler
	handler := &CleanupUnusedUploadsHandler{}
	result, err := handler.Execute(ctx, nil)

	// 验证结果
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, result.Message, "共处理 2 个文件，成功删除 2 个")

	// 验证数据库状态：pending 且超过1小时的应被标记为 deleted
	var pendingCount int64
	db.DB(ctx).Model(&model.Upload{}).Where("status = ?", model.UploadStatusPending).Count(&pendingCount)
	assert.Equal(t, int64(1), pendingCount, "应只剩1条 pending 记录（最近的文件）")

	var deletedCount int64
	db.DB(ctx).Model(&model.Upload{}).Where("status = ?", model.UploadStatusDeleted).Count(&deletedCount)
	assert.Equal(t, int64(2), deletedCount, "应有2条被标记为 deleted")

	var usedCount int64
	db.DB(ctx).Model(&model.Upload{}).Where("status = ?", model.UploadStatusUsed).Count(&usedCount)
	assert.Equal(t, int64(1), usedCount, "used 状态的文件不应受影响")
}

func TestCleanupUnusedUploadsHandler_ExecuteNoFiles(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	// Mock S3 存储
	storageMock := storage.MockStorage(
		func(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
			return nil
		},
		func(ctx context.Context, key string) (*storage.ObjectInfo, error) { return nil, nil },
		func(ctx context.Context, key string) error { return nil },
	)
	defer storageMock()

	ctx := context.Background()

	// 没有任何上传记录
	handler := &CleanupUnusedUploadsHandler{}
	result, err := handler.Execute(ctx, nil)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, result.Message, "共处理 0 个文件，成功删除 0 个")
}

func TestCleanupUnusedUploadsHandler_ImplementsTaskHandler(t *testing.T) {
	// 编译期验证 CleanupUnusedUploadsHandler 实现了 TaskHandler 接口
	var _ task.TaskHandler = (*CleanupUnusedUploadsHandler)(nil)
}
