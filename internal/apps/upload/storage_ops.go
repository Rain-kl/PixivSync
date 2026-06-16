// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package upload

import (
	"context"
	"fmt"

	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/storage"
	"github.com/Rain-kl/Wavelet/pkg/logger"
)

// StorageReadOnly checks if the storage system is in read-only maintenance mode.
func StorageReadOnly(ctx context.Context) bool {
	execution, ok, err := latestStorageMigrationExecution(ctx)
	if err != nil {
		logger.ErrorF(ctx, "读取存储维护状态失败: %v", err)
		return true
	}
	if !ok {
		return false
	}
	return execution.Status != model.TaskExecutionStatusSucceeded
}

func openStoredObject(ctx context.Context, upload *model.Upload) (*storage.Object, error) {
	driver := storage.Driver(upload.StorageDriver)
	if driver == "" {
		driver = storage.DriverLocal
	}
	backend, err := backendForStoredDriver(ctx, driver)
	if err != nil {
		return nil, err
	}
	return backend.Get(ctx, upload.FilePath)
}

func backendForStoredDriver(ctx context.Context, driver storage.Driver) (storage.Backend, error) {
	backend, err := storage.ForDriver(ctx, driver)
	if err == nil {
		return backend, nil
	}

	target, ok, targetErr := currentMigrationTargetConfig(ctx)
	if targetErr != nil {
		return nil, targetErr
	}
	if ok && target.Driver == driver {
		return storage.NewBackend(ctx, target, driver)
	}
	return nil, fmt.Errorf("storage configuration for driver %q is unavailable", driver)
}

func currentMigrationTargetConfig(ctx context.Context) (storage.Config, bool, error) {
	execution, ok, err := latestStorageMigrationExecution(ctx)
	if err != nil || !ok {
		return storage.Config{}, false, err
	}
	if execution.Status == model.TaskExecutionStatusSucceeded {
		return storage.Config{}, false, nil
	}
	target, err := parseMigrationTargetConfig(ctx, []byte(execution.Payload))
	if err != nil {
		return storage.Config{}, false, err
	}
	return target, true, nil
}
