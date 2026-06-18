// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package ingest

import (
	"context"

	uploadstats "github.com/Rain-kl/Wavelet/internal/apps/upload/stats"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/repository"
)

// Remove soft-deletes an upload and decrements incremental stats.
func Remove(ctx context.Context, uploadID uint64) (model.Upload, error) {
	upload, err := repository.GetActiveUploadByID(ctx, uploadID)
	if err != nil {
		return model.Upload{}, err
	}
	uploadstats.RecordUploadStatsRemove(ctx, &upload)
	if err := repository.SoftDeleteUpload(ctx, &upload); err != nil {
		return model.Upload{}, err
	}
	upload.Status = model.UploadStatusDeleted
	return upload, nil
}

// RemoveOwned soft-deletes an upload owned by userID and decrements incremental stats.
func RemoveOwned(ctx context.Context, userID, uploadID uint64) (model.Upload, error) {
	upload, err := repository.GetActiveUploadByID(ctx, uploadID)
	if err != nil {
		return model.Upload{}, err
	}
	if upload.UserID != userID {
		return model.Upload{}, ErrForbidden
	}
	uploadstats.RecordUploadStatsRemove(ctx, &upload)
	if err := repository.SoftDeleteUpload(ctx, &upload); err != nil {
		return model.Upload{}, err
	}
	upload.Status = model.UploadStatusDeleted
	return upload, nil
}