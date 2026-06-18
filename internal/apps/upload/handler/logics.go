// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"bytes"
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/Rain-kl/Wavelet/internal/apps/upload/shared"
	uploadstats "github.com/Rain-kl/Wavelet/internal/apps/upload/stats"
	uploadstorage "github.com/Rain-kl/Wavelet/internal/apps/upload/storage"
	"github.com/Rain-kl/Wavelet/internal/db/idgen"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/repository"
	"github.com/Rain-kl/Wavelet/internal/storage"
	"github.com/Rain-kl/Wavelet/pkg/logger"
	"gorm.io/gorm"
)

func listUploadFiles(ctx context.Context, filter repository.UploadListFilter) (int64, []model.Upload, error) {
	return repository.ListUploads(ctx, filter)
}

func listMyUploadFiles(ctx context.Context, userID uint64, filter repository.UploadListFilter) (int64, []model.Upload, error) {
	filter.UserID = userID
	return repository.ListUploads(ctx, filter)
}

func softDeleteUpload(ctx context.Context, uploadID uint64) (model.Upload, error) {
	upload, err := repository.GetActiveUploadByID(ctx, uploadID)
	if err != nil {
		return model.Upload{}, err
	}
	if err := repository.SoftDeleteUpload(ctx, &upload); err != nil {
		return model.Upload{}, err
	}
	uploadstats.RecordUploadStatsRemove(ctx, &upload)
	return upload, nil
}

func softDeleteOwnedUpload(ctx context.Context, userID, uploadID uint64) (model.Upload, error) {
	upload, err := repository.GetActiveUploadByID(ctx, uploadID)
	if err != nil {
		return model.Upload{}, err
	}
	if upload.UserID != userID {
		return model.Upload{}, errUploadForbidden
	}
	if err := repository.SoftDeleteUpload(ctx, &upload); err != nil {
		return model.Upload{}, err
	}
	uploadstats.RecordUploadStatsRemove(ctx, &upload)
	return upload, nil
}

func listDistinctUploadTypes(ctx context.Context) ([]string, error) {
	types, err := repository.ListDistinctUploadTypes(ctx)
	if err != nil {
		return nil, err
	}
	sort.Strings(types)
	return types, nil
}

type updateMyUploadInput struct {
	FileName   string
	AccessMode *int
}

func updateOwnedUpload(ctx context.Context, userID, uploadID uint64, input updateMyUploadInput) (model.Upload, error) {
	upload, err := repository.GetActiveUploadByID(ctx, uploadID)
	if err != nil {
		return model.Upload{}, err
	}
	if upload.UserID != userID {
		return model.Upload{}, errUploadForbidden
	}

	updates := make(map[string]any)
	if input.FileName != "" {
		updates["file_name"] = input.FileName
	}
	if input.AccessMode != nil {
		updates["access_mode"] = *input.AccessMode
	}
	if err := repository.UpdateUpload(ctx, &upload, updates); err != nil {
		return model.Upload{}, err
	}
	if name, ok := updates["file_name"].(string); ok {
		upload.FileName = name
	}
	if mode, ok := updates["access_mode"].(int); ok {
		upload.AccessMode = mode
	}
	return upload, nil
}

func listUploadsForBatchDownload(ctx context.Context, ids []uint64) ([]model.Upload, error) {
	return repository.ListUploadsByIDs(ctx, ids)
}

type instantUploadInput struct {
	UserID     uint64
	FileHash   string
	Size       int64
	MimeType   string
	Extension  string
	OrigName   string
	UploadType string
	AccessMode int
}

func createInstantUpload(ctx context.Context, existing model.Upload, input instantUploadInput) (model.Upload, error) {
	newUpload := model.Upload{
		ID:            idgen.NextUint64ID(),
		UserID:        input.UserID,
		FileName:      input.OrigName,
		FilePath:      existing.FilePath,
		FileSize:      input.Size,
		MimeType:      input.MimeType,
		Extension:     input.Extension,
		Hash:          input.FileHash,
		Type:          input.UploadType,
		Status:        model.UploadStatusUsed,
		AccessMode:    input.AccessMode,
		Metadata:      existing.Metadata,
	}
	if err := repository.CreateUpload(ctx, &newUpload); err != nil {
		return model.Upload{}, err
	}
	uploadstats.RecordUploadStatsAdd(ctx, &newUpload)
	logger.InfoF(ctx, "文件触发秒传成功! ID: %d, Path: %s", newUpload.ID, existing.FilePath)
	return newUpload, nil
}

func findReusableUpload(ctx context.Context, hash string, size int64) (model.Upload, error) {
	return repository.FindReusableUploadByHash(ctx, hash, size)
}

func saveNewUploadRecord(ctx context.Context, upload *model.Upload, filePath string) error {
	if err := repository.CreateUpload(ctx, upload); err != nil {
		_, backend, backendErr := storage.Active(ctx)
		if backendErr == nil {
			if deleteErr := backend.Delete(ctx, filePath); deleteErr != nil {
				logger.WarnF(ctx, "清理未写入数据库的上传对象失败: %v", deleteErr)
			}
		}
		return err
	}
	uploadstats.RecordUploadStatsAdd(ctx, upload)
	return nil
}

func loadUploadStats(ctx context.Context) ([]model.UploadStat, error) {
	return repository.ListUploadStats(ctx)
}

var errUploadForbidden = errors.New("upload forbidden")

func storeUploadObject(ctx context.Context, subPath string, size int64, mimeType string, buf *bytes.Buffer, meta *model.UploadMetadata) (string, error) {
	if uploadstorage.ReadOnly(ctx) {
		return "", errors.New(shared.ErrStorageReadOnly)
	}
	driver, backend, err := storage.Active(ctx)
	if err != nil {
		logger.ErrorF(ctx, "初始化活动存储失败: %v", err)
		return "", errors.New(shared.ErrSaveFileFailed)
	}
	result, err := backend.Put(ctx, subPath, bytes.NewReader(buf.Bytes()), size, mimeType)
	if err != nil {
		logger.ErrorF(ctx, "写入 %s 存储失败: %v", driver, err)
		return "", errors.New(shared.ErrSaveFileFailed)
	}
	meta.Bucket = result.Bucket
	return result.Key, nil
}

func validateUploadAllowedExtension(ctx context.Context, ext string) string {
	sc, err := repository.GetSystemConfigByKey(ctx, model.ConfigKeyUploadAllowedExtensions)
	if err != nil || sc.Value == "" {
		return ""
	}
	allowedExts := strings.Split(strings.ToLower(sc.Value), ",")
	for _, allowedExt := range allowedExts {
		if strings.TrimSpace(allowedExt) == ext {
			return ""
		}
	}
	return shared.ErrUnsupportedFormat
}

func isRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}