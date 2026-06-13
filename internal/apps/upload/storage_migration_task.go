// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package upload

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/storage"
	"github.com/Rain-kl/Wavelet/internal/task"
	"gorm.io/gorm"
)

const (
	// StorageMigrationTask is the Asynq task name for storage migration.
	StorageMigrationTask = "storage:migrate"
	// TaskTypeStorageMigration is the task metadata type for storage migration.
	TaskTypeStorageMigration = "storage_migration"
)

// StorageMigrationMeta describes the manually dispatchable migration task.
var StorageMigrationMeta = task.TaskMeta{
	Type:         TaskTypeStorageMigration,
	AsynqTask:    StorageMigrationTask,
	Name:         "迁移文件存储",
	Description:  "将活动存储中的文件迁移到待切换的目标存储，迁移期间文件系统保持只读",
	SupportsTime: false,
	MaxRetry:     task.DefaultMaxRetry,
	Queue:        task.QueueDefault,
	Retryable:    true,
}

// MigrationHandler copies stored objects and activates the target backend.
type MigrationHandler struct{}

type storageMigrationPayload struct {
	Target storage.Config `json:"target"`
}

// ValidatePayload rejects duplicate active migrations through the task framework.
func (h *MigrationHandler) ValidatePayload(payload []byte) ([]byte, error) {
	normalized, _, err := normalizeStorageMigrationPayload(context.Background(), payload)
	if err != nil {
		return payload, err
	}
	active, err := hasUnresolvedMigrationTask(context.Background())
	if err != nil {
		return payload, err
	}
	if active {
		return payload, fmt.Errorf("storage migration task is already unresolved")
	}
	return normalized, nil
}

// Execute migrates all unique active-storage objects to the pending backend.
func (h *MigrationHandler) Execute(ctx context.Context, payload []byte) (*task.TaskResult, error) {
	active, err := storage.LoadConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load active storage config: %w", err)
	}
	target, err := parseMigrationTargetConfig(ctx, payload)
	if err != nil {
		return nil, err
	}
	if target.Driver == active.Driver {
		if err := storage.SaveActiveConfig(ctx, target); err != nil {
			return nil, fmt.Errorf("activate same-driver storage config: %w", err)
		}
		message := fmt.Sprintf("存储配置已更新，活动存储保持为 %s", target.Driver)
		task.AppendLog(ctx, "%s", message)
		return &task.TaskResult{Message: message}, nil
	}

	total, err := countStorageObjects(ctx, active.Driver)
	if err != nil {
		return nil, fmt.Errorf("count source objects: %w", err)
	}
	if total == 0 {
		if err := storage.SaveActiveConfig(ctx, target); err != nil {
			return nil, fmt.Errorf("activate empty storage config: %w", err)
		}
		message := fmt.Sprintf("当前存储没有需要迁移的对象，活动存储已切换为 %s", target.Driver)
		task.AppendLog(ctx, "%s", message)
		return &task.TaskResult{Message: message}, nil
	}

	sourceBackend, err := storage.NewBackend(ctx, active, active.Driver)
	if err != nil {
		return nil, fmt.Errorf("create source storage: %w", err)
	}
	targetBackend, err := storage.NewBackend(ctx, target, target.Driver)
	if err != nil {
		return nil, fmt.Errorf("create target storage: %w", err)
	}

	task.AppendLog(ctx, "开始存储迁移: %s -> %s，总对象数: %d", active.Driver, target.Driver, total)
	migrated, err := migrateObjects(ctx, sourceBackend, targetBackend, active.Driver, target.Driver, total)
	if err != nil {
		return nil, err
	}

	if err := storage.SaveActiveConfig(ctx, target); err != nil {
		return nil, fmt.Errorf("activate target storage: %w", err)
	}
	message := fmt.Sprintf("存储迁移完成，共迁移 %d 个对象，活动存储已切换为 %s", migrated, target.Driver)
	task.AppendLog(ctx, "%s", message)
	return &task.TaskResult{Message: message}, nil
}

func normalizeStorageMigrationPayload(ctx context.Context, payload []byte) ([]byte, storage.Config, error) {
	target, err := parseMigrationTargetConfig(ctx, payload)
	if err != nil {
		return nil, storage.Config{}, err
	}
	normalized, err := json.Marshal(storageMigrationPayload{Target: target})
	if err != nil {
		return nil, storage.Config{}, fmt.Errorf("marshal storage migration payload: %w", err)
	}
	return normalized, target, nil
}

func parseMigrationTargetConfig(ctx context.Context, payload []byte) (storage.Config, error) {
	if strings.TrimSpace(string(payload)) == "" {
		return storage.Config{}, errors.New("storage migration target payload is required")
	}
	var req storageMigrationPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		return storage.Config{}, fmt.Errorf("parse storage migration payload: %w", err)
	}
	current, err := storage.LoadConfig(ctx)
	if err != nil {
		return storage.Config{}, fmt.Errorf("load active storage config: %w", err)
	}
	target := storage.MergeMaskedSecrets(req.Target, current)
	if err := storage.ValidateConfig(target); err != nil {
		return storage.Config{}, fmt.Errorf("validate target storage config: %w", err)
	}
	return target, nil
}

func countStorageObjects(ctx context.Context, driver storage.Driver) (int64, error) {
	var count int64
	err := db.DB(ctx).Model(&model.Upload{}).
		Where("storage_driver = ? AND status != ?", driver, model.UploadStatusDeleted).
		Distinct("file_path").
		Count(&count).Error
	return count, err
}

func hasUnresolvedMigrationTask(ctx context.Context) (bool, error) {
	execution, ok, err := latestStorageMigrationExecution(ctx)
	if err != nil || !ok {
		return false, err
	}
	return execution.Status == model.TaskExecutionStatusPending || execution.Status == model.TaskExecutionStatusRunning, nil
}

func latestStorageMigrationExecution(ctx context.Context) (*model.TaskExecution, bool, error) {
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

func migrateObjects(
	ctx context.Context,
	sourceBackend storage.Backend,
	targetBackend storage.Backend,
	sourceDriver storage.Driver,
	targetDriver storage.Driver,
	total int64,
) (int64, error) {
	const batchSize = 50
	var migrated int64
	for {
		if err := ctx.Err(); err != nil {
			return migrated, fmt.Errorf("storage migration canceled: %w", err)
		}

		var objects []struct {
			FilePath string `gorm:"column:file_path"`
			FileSize int64  `gorm:"column:file_size"`
			MimeType string `gorm:"column:mime_type"`
		}
		if err := db.DB(ctx).Model(&model.Upload{}).
			Select("file_path, MAX(file_size) AS file_size, MAX(mime_type) AS mime_type").
			Where("storage_driver = ? AND status != ?", sourceDriver, model.UploadStatusDeleted).
			Group("file_path").
			Order("file_path ASC").
			Limit(batchSize).
			Scan(&objects).Error; err != nil {
			return migrated, fmt.Errorf("query source objects: %w", err)
		}
		if len(objects) == 0 {
			break
		}

		for _, object := range objects {
			source, err := sourceBackend.Get(ctx, object.FilePath)
			if err != nil {
				if isNotFoundError(err) {
					task.AppendLog(ctx, "警告: 源存储中物理文件不存在，标记为已删除并跳过: %s (错误: %v)", object.FilePath, err)
					if updateErr := db.DB(ctx).Model(&model.Upload{}).
						Where("storage_driver = ? AND file_path = ?", sourceDriver, object.FilePath).
						Updates(map[string]any{
							"status":         model.UploadStatusDeleted,
							"storage_driver": targetDriver,
						}).Error; updateErr != nil {
						return migrated, fmt.Errorf("update missing object %q: %w", object.FilePath, updateErr)
					}
					continue
				}
				return migrated, fmt.Errorf("open source object %q: %w", object.FilePath, err)
			}
			targetPath, putErr := targetBackend.Put(ctx, object.FilePath, source.Body, object.FileSize, object.MimeType)
			closeErr := source.Body.Close()
			if putErr != nil {
				return migrated, fmt.Errorf("copy object %q: %w", object.FilePath, putErr)
			}
			if closeErr != nil {
				return migrated, fmt.Errorf("close source object %q: %w", object.FilePath, closeErr)
			}
			if err := db.DB(ctx).Model(&model.Upload{}).
				Where("storage_driver = ? AND file_path = ?", sourceDriver, object.FilePath).
				Updates(map[string]any{
					"storage_driver": targetDriver,
					"file_path":      targetPath,
				}).Error; err != nil {
				return migrated, fmt.Errorf("update migrated object %q: %w", object.FilePath, err)
			}
			migrated++
		}
		task.AppendLog(ctx, "迁移进度: %d/%d", migrated, total)
	}
	return migrated, nil
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	errStr := strings.ToLower(err.Error())
	for _, sub := range []string{"not found", "nosuchkey", "nosuchbucket", "404", "does not exist"} {
		if strings.Contains(errStr, sub) {
			return true
		}
	}
	return false
}
