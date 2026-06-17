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
	state := loadMigrationAccessState(ctx)
	if state.loadErr != nil {
		logger.ErrorF(ctx, "读取存储维护状态失败: %v", state.loadErr)
		return true
	}
	return state.readOnly
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
	state := loadMigrationAccessState(ctx)
	if state.loadErr != nil {
		return storage.Config{}, false, state.loadErr
	}
	if state.targetErr != nil {
		return storage.Config{}, false, state.targetErr
	}
	if !state.hasTarget {
		return storage.Config{}, false, nil
	}
	return state.target, true, nil
}
