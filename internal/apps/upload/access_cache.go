// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package upload

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/storage"
)

const accessCacheTTL = 5 * time.Second

const fileAccessInvalidationChannel = "upload:file_access_invalidation"

type migrationAccessState struct {
	readOnly  bool
	target    storage.Config
	hasTarget bool
	targetErr error
	loadErr   error
}

var (
	accessCacheOnce sync.Once

	migrationAccessMu        sync.RWMutex
	migrationAccessCached    migrationAccessState
	migrationAccessValid     bool
	migrationAccessCheckedAt time.Time

	fileAccessWhitelistMu        sync.RWMutex
	fileAccessWhitelistTypes     map[string]struct{}
	fileAccessWhitelistValid     bool
	fileAccessWhitelistCheckedAt time.Time
)

// ResetAccessCaches clears in-process upload access caches.
func ResetAccessCaches() {
	migrationAccessMu.Lock()
	migrationAccessValid = false
	migrationAccessMu.Unlock()

	fileAccessWhitelistMu.Lock()
	fileAccessWhitelistValid = false
	fileAccessWhitelistTypes = nil
	fileAccessWhitelistMu.Unlock()
}

// PublishAccessCacheInvalidation broadcasts upload access cache eviction to all nodes.
func PublishAccessCacheInvalidation(ctx context.Context) {
	if db.Redis != nil {
		_ = db.Redis.Publish(ctx, fileAccessInvalidationChannel, "reset").Err()
	}
}

func ensureAccessCacheListener() {
	accessCacheOnce.Do(startAccessCacheInvalidationListener)
}

func startAccessCacheInvalidationListener() {
	if db.Redis == nil {
		return
	}

	go func() {
		pubsub := db.Redis.Subscribe(
			context.Background(),
			storage.ConfigInvalidationChannel,
			fileAccessInvalidationChannel,
		)
		defer func() {
			_ = pubsub.Close()
		}()

		for range pubsub.Channel() {
			ResetAccessCaches()
		}
	}()
}

func loadMigrationAccessState(ctx context.Context) migrationAccessState {
	ensureAccessCacheListener()

	migrationAccessMu.RLock()
	if migrationAccessValid && time.Since(migrationAccessCheckedAt) < accessCacheTTL {
		state := migrationAccessCached
		migrationAccessMu.RUnlock()
		return state
	}
	migrationAccessMu.RUnlock()

	migrationAccessMu.Lock()
	defer migrationAccessMu.Unlock()

	if migrationAccessValid && time.Since(migrationAccessCheckedAt) < accessCacheTTL {
		return migrationAccessCached
	}

	migrationAccessCached = buildMigrationAccessState(ctx)
	migrationAccessValid = true
	migrationAccessCheckedAt = time.Now()
	return migrationAccessCached
}

func buildMigrationAccessState(ctx context.Context) migrationAccessState {
	execution, ok, err := latestStorageMigrationExecution(ctx)
	if err != nil {
		return migrationAccessState{loadErr: err, readOnly: true}
	}
	if !ok {
		return migrationAccessState{}
	}

	state := migrationAccessState{
		readOnly: execution.Status != model.TaskExecutionStatusSucceeded,
	}
	if execution.Status == model.TaskExecutionStatusSucceeded {
		return state
	}

	target, err := parseMigrationTargetConfig(ctx, []byte(execution.Payload))
	if err != nil {
		state.targetErr = err
		return state
	}

	state.target = target
	state.hasTarget = true
	return state
}

func loadFileAccessWhitelist(ctx context.Context) map[string]struct{} {
	ensureAccessCacheListener()

	fileAccessWhitelistMu.RLock()
	if fileAccessWhitelistValid && time.Since(fileAccessWhitelistCheckedAt) < accessCacheTTL {
		types := fileAccessWhitelistTypes
		fileAccessWhitelistMu.RUnlock()
		return types
	}
	fileAccessWhitelistMu.RUnlock()

	fileAccessWhitelistMu.Lock()
	defer fileAccessWhitelistMu.Unlock()

	if fileAccessWhitelistValid && time.Since(fileAccessWhitelistCheckedAt) < accessCacheTTL {
		return fileAccessWhitelistTypes
	}

	fileAccessWhitelistTypes = fetchFileAccessWhitelist(ctx)
	fileAccessWhitelistValid = true
	fileAccessWhitelistCheckedAt = time.Now()
	return fileAccessWhitelistTypes
}

func fetchFileAccessWhitelist(ctx context.Context) map[string]struct{} {
	whitelist := parseFileAccessWhitelist(ctx)
	types := make(map[string]struct{}, len(whitelist))
	for _, item := range whitelist {
		types[strings.ToLower(item)] = struct{}{}
	}
	return types
}

func parseFileAccessWhitelist(ctx context.Context) []string {
	var sc model.SystemConfig
	if err := sc.GetByKey(ctx, model.ConfigKeyFileAccessWhitelist); err != nil || sc.Value == "" {
		return []string{defaultPublicUploadType}
	}

	var whitelist []string
	if err := json.Unmarshal([]byte(sc.Value), &whitelist); err == nil && len(whitelist) > 0 {
		return whitelist
	}

	whitelist = parseCommaSeparatedWhitelist(sc.Value)
	if len(whitelist) == 0 {
		return []string{defaultPublicUploadType}
	}
	return whitelist
}

func parseCommaSeparatedWhitelist(value string) []string {
	parts := strings.Split(value, ",")
	whitelist := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			whitelist = append(whitelist, part)
		}
	}
	return whitelist
}
