// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package upload

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/storage"
	"github.com/Rain-kl/Wavelet/internal/task"
	"golang.org/x/sync/errgroup"
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
	if db.Redis != nil {
		const (
			cleanupTimeout  = 5 * time.Second
			renewalInterval = 10 * time.Minute
		)

		lockKey := db.PrefixedKey("lock:storage:migrate")
		ok, err := db.Redis.SetNX(ctx, lockKey, "locked", time.Hour).Result()
		if err != nil {
			return nil, fmt.Errorf("acquire migration lock: %w", err)
		}
		if !ok {
			return nil, errors.New("另一个存储迁移任务正在运行中")
		}

		// 任务结束时清理锁，使用 Background context 避免受任务 context 取消的影响
		stopRenewal := make(chan struct{})
		//nolint:contextcheck
		defer func() {
			close(stopRenewal)
			cleanupCtx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
			defer cancel()
			_ = db.Redis.Del(cleanupCtx, lockKey)
		}()

		// 启动看门狗续租协程，每 10 分钟将锁的 TTL 自动延长为 1 小时
		//nolint:contextcheck,gosec
		go func() {
			ticker := time.NewTicker(renewalInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					renewCtx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
					_ = db.Redis.Expire(renewCtx, lockKey, time.Hour).Err()
					cancel()
				case <-stopRenewal:
					return
				case <-ctx.Done():
					return
				}
			}
		}()
	}

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
	const migrationConcurrency = 10
	const sha256HexLength = 64
	var migrated int64
	for {
		if err := ctx.Err(); err != nil {
			return atomic.LoadInt64(&migrated), fmt.Errorf("storage migration canceled: %w", err)
		}

		var objects []struct {
			FilePath string `gorm:"column:file_path"`
			FileSize int64  `gorm:"column:file_size"`
			MimeType string `gorm:"column:mime_type"`
			Hash     string `gorm:"column:hash"`
		}
		if err := db.DB(ctx).Model(&model.Upload{}).
			Select("file_path, MAX(file_size) AS file_size, MAX(mime_type) AS mime_type, MAX(hash) AS hash").
			Where("storage_driver = ? AND status != ?", sourceDriver, model.UploadStatusDeleted).
			Group("file_path").
			Order("file_path ASC").
			Limit(batchSize).
			Scan(&objects).Error; err != nil {
			return atomic.LoadInt64(&migrated), fmt.Errorf("query source objects: %w", err)
		}
		if len(objects) == 0 {
			break
		}

		var g errgroup.Group
		g.SetLimit(migrationConcurrency)

		for _, object := range objects {
			obj := object // Capture range variable
			g.Go(func() error {
				source, err := sourceBackend.Get(ctx, obj.FilePath)
				if err != nil {
					if isNotFoundError(err) {
						task.AppendLog(ctx, "警告: 源存储中物理文件不存在，标记为已删除并跳过: %s (错误: %v)", obj.FilePath, err)
						if updateErr := db.DB(ctx).Model(&model.Upload{}).
							Where("storage_driver = ? AND file_path = ?", sourceDriver, obj.FilePath).
							Updates(map[string]any{
								"status":         model.UploadStatusDeleted,
								"storage_driver": targetDriver,
							}).Error; updateErr != nil {
							return fmt.Errorf("update missing object %q: %w", obj.FilePath, updateErr)
						}
						return nil
					}
					return fmt.Errorf("open source object %q: %w", obj.FilePath, err)
				}
				targetPath, putErr := targetBackend.Put(ctx, obj.FilePath, source.Body, obj.FileSize, obj.MimeType)
				closeErr := source.Body.Close()
				if putErr != nil {
					return fmt.Errorf("copy object %q: %w", obj.FilePath, putErr)
				}
				if closeErr != nil {
					return fmt.Errorf("close source object %q: %w", obj.FilePath, closeErr)
				}

				// Data integrity check (SHA-256 hash verification)
				if len(obj.Hash) == sha256HexLength {
					targetObj, getErr := targetBackend.Get(ctx, targetPath)
					if getErr != nil {
						return fmt.Errorf("retrieve target object for verification %q: %w", obj.FilePath, getErr)
					}
					h := sha256.New()
					if _, copyErr := io.Copy(h, targetObj.Body); copyErr != nil {
						_ = targetObj.Body.Close()
						return fmt.Errorf("read target object for verification %q: %w", obj.FilePath, copyErr)
					}
					_ = targetObj.Body.Close()
					computedHash := hex.EncodeToString(h.Sum(nil))
					if computedHash != obj.Hash {
						return fmt.Errorf("integrity check failed for %q: got hash %s, want %s", obj.FilePath, computedHash, obj.Hash)
					}
				}

				if err := db.DB(ctx).Model(&model.Upload{}).
					Where("storage_driver = ? AND file_path = ?", sourceDriver, obj.FilePath).
					Updates(map[string]any{
						"storage_driver": targetDriver,
						"file_path":      targetPath,
					}).Error; err != nil {
					return fmt.Errorf("update migrated object %q: %w", obj.FilePath, err)
				}
				atomic.AddInt64(&migrated, 1)
				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return atomic.LoadInt64(&migrated), err
		}

		task.AppendLog(ctx, "迁移进度: %d/%d", atomic.LoadInt64(&migrated), total)
	}
	return atomic.LoadInt64(&migrated), nil
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
