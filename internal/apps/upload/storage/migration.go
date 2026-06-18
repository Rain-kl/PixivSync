// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/storage"
	"gorm.io/gorm"
)

// StorageMigrationTask is the Asynq task name for storage migration.
const StorageMigrationTask = "storage:migrate"

// LatestMigrationExecution returns the most recent storage migration task execution.
func LatestMigrationExecution(ctx context.Context) (*model.TaskExecution, bool, error) {
	var execution model.TaskExecution
	err := db.DB(ctx).
		Where("task_type = ?", StorageMigrationTask).
		Order("id DESC").
		First(&execution).Error
	if err == nil {
		return &execution, true, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}
	return nil, false, nil
}

// ParseMigrationTargetConfig parses and validates a storage migration target payload.
func ParseMigrationTargetConfig(ctx context.Context, payload []byte) (storage.Config, error) {
	if strings.TrimSpace(string(payload)) == "" {
		return storage.Config{}, errors.New("storage migration target payload is required")
	}

	var raw struct {
		Target json.RawMessage `json:"target"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return storage.Config{}, fmt.Errorf("parse storage migration payload envelope: %w", err)
	}

	if len(raw.Target) == 0 {
		return storage.Config{}, errors.New("storage migration target payload is required")
	}

	var targetBytes []byte
	var targetStr string
	if err := json.Unmarshal(raw.Target, &targetStr); err == nil {
		targetBytes = []byte(targetStr)
	} else {
		targetBytes = raw.Target
	}

	var target storage.Config
	if err := json.Unmarshal(targetBytes, &target); err != nil {
		return storage.Config{}, fmt.Errorf("parse target storage config: %w", err)
	}

	current, err := storage.LoadConfig(ctx)
	if err != nil {
		return storage.Config{}, fmt.Errorf("load active storage config: %w", err)
	}
	target = storage.MergeMaskedSecrets(target, current)
	if err := storage.ValidateConfig(target); err != nil {
		return storage.Config{}, fmt.Errorf("validate target storage config: %w", err)
	}
	return target, nil
}

// NormalizeMigrationPayload validates and normalizes a storage migration payload.
func NormalizeMigrationPayload(ctx context.Context, payload []byte) ([]byte, storage.Config, error) {
	target, err := ParseMigrationTargetConfig(ctx, payload)
	if err != nil {
		return nil, storage.Config{}, err
	}
	type storageMigrationPayload struct {
		Target storage.Config `json:"target"`
	}
	normalized, err := json.Marshal(storageMigrationPayload{Target: target})
	if err != nil {
		return nil, storage.Config{}, fmt.Errorf("marshal storage migration payload: %w", err)
	}
	return normalized, target, nil
}