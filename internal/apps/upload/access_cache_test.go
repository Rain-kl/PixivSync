// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package upload

import (
	"context"
	"testing"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/testhelper"
)

func TestLoadMigrationAccessStateCachesResult(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ResetAccessCaches()

	ctx := context.Background()
	first := loadMigrationAccessState(ctx)
	second := loadMigrationAccessState(ctx)

	if first.readOnly != second.readOnly {
		t.Fatalf("readOnly mismatch: first=%v second=%v", first.readOnly, second.readOnly)
	}
	if first.hasTarget != second.hasTarget {
		t.Fatalf("hasTarget mismatch: first=%v second=%v", first.hasTarget, second.hasTarget)
	}
}

func TestIsFilePublicUsesCachedWhitelist(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ResetAccessCaches()

	ctx := context.Background()
	if !isFilePublic(ctx, "avatar") {
		t.Fatal("expected avatar to be public by default")
	}
	if isFilePublic(ctx, "attachment") {
		t.Fatal("expected attachment to be private by default")
	}
	if !isFilePublic(ctx, "AVATAR") {
		t.Fatal("expected whitelist lookup to be case-insensitive")
	}
}

func TestResetAccessCachesRefreshesWhitelist(t *testing.T) {
	dbConn, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ResetAccessCaches()

	ctx := context.Background()
	if !isFilePublic(ctx, "avatar") {
		t.Fatal("expected seeded avatar whitelist before reset")
	}

	var sc model.SystemConfig
	if err := dbConn.Where("key = ?", model.ConfigKeyFileAccessWhitelist).First(&sc).Error; err != nil {
		t.Fatalf("load whitelist config: %v", err)
	}
	sc.Value = `["attachment"]`
	if err := dbConn.Save(&sc).Error; err != nil {
		t.Fatalf("save whitelist config: %v", err)
	}
	if err := db.HSetJSON(ctx, model.SystemConfigRedisHashKey, model.ConfigKeyFileAccessWhitelist, &sc); err != nil {
		t.Fatalf("refresh whitelist redis cache: %v", err)
	}

	ResetAccessCaches()
	if !isFilePublic(ctx, "attachment") {
		t.Fatal("expected attachment to be public after whitelist refresh")
	}
	if isFilePublic(ctx, "avatar") {
		t.Fatal("expected avatar to be private after whitelist refresh")
	}
}

func TestAccessCacheTTLExpires(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ResetAccessCaches()

	ctx := context.Background()
	_ = loadFileAccessWhitelist(ctx)

	fileAccessWhitelistMu.Lock()
	fileAccessWhitelistCheckedAt = time.Now().Add(-accessCacheTTL - time.Second)
	fileAccessWhitelistMu.Unlock()

	// Should still work after TTL by reloading from config.
	if !isFilePublic(ctx, "avatar") {
		t.Fatal("expected whitelist reload after TTL expiration")
	}
}
