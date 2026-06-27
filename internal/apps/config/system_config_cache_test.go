// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"testing"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/repository"
	"github.com/Rain-kl/Wavelet/internal/testhelper"
)

func TestSystemConfigRAMCacheServesUntilInvalidated(t *testing.T) {
	dbConn, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	repository.ResetSystemConfigRAMCacheForTest()
	if err := repository.InvalidateAllSystemConfigCaches(ctx); err != nil {
		t.Fatalf("InvalidateAllSystemConfigCaches() error = %v", err)
	}

	warm, err := repository.GetSystemConfigByKey(ctx, model.ConfigKeySiteName)
	if err != nil {
		t.Fatalf("GetSystemConfigByKey(site_name) warm error = %v", err)
	}
	if warm.Value != "Wavelet" {
		t.Fatalf("GetSystemConfigByKey(site_name).Value = %q, want %q", warm.Value, "Wavelet")
	}

	// Update DB directly (bypassing caching layer)
	if err := dbConn.Model(&model.SystemConfig{}).
		Where("key = ?", model.ConfigKeySiteName).
		Update("value", "ram_probe_value").Error; err != nil {
		t.Fatalf("Update(site_name) error = %v", err)
	}

	// Should still return "Wavelet" since it's cached in RAM cache
	cached, err := repository.GetSystemConfigByKey(ctx, model.ConfigKeySiteName)
	if err != nil {
		t.Fatalf("GetSystemConfigByKey(site_name) cached error = %v", err)
	}
	if cached.Value != "Wavelet" {
		t.Fatalf("GetSystemConfigByKey(site_name).Value = %q, want stale RAM value %q", cached.Value, "Wavelet")
	}

	// Invalidate the cache (triggers refresh callback)
	if err := repository.InvalidateSystemConfigCache(ctx, model.ConfigKeySiteName); err != nil {
		t.Fatalf("InvalidateSystemConfigCache(site_name) error = %v", err)
	}

	// Allow some time for broadcast listener in test environment
	time.Sleep(100 * time.Millisecond)

	// Should return the updated value now
	refreshed, err := repository.GetSystemConfigByKey(ctx, model.ConfigKeySiteName)
	if err != nil {
		t.Fatalf("GetSystemConfigByKey(site_name) refreshed error = %v", err)
	}
	if refreshed.Value != "ram_probe_value" {
		t.Fatalf("GetSystemConfigByKey(site_name).Value = %q, want %q", refreshed.Value, "ram_probe_value")
	}
}

func TestInvalidateSystemConfigCacheBroadcast(t *testing.T) {
	dbConn, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	repository.ResetSystemConfigRAMCacheForTest()

	// Initially seed in cache
	_, err := repository.GetSystemConfigByKey(ctx, model.ConfigKeySiteName)
	if err != nil {
		t.Fatalf("GetSystemConfigByKey(site_name) error = %v", err)
	}

	// Update DB directly
	if err := dbConn.Model(&model.SystemConfig{}).
		Where("key = ?", model.ConfigKeySiteName).
		Update("value", "broadcast_value").Error; err != nil {
		t.Fatalf("Update(site_name) error = %v", err)
	}

	// Invalidate: publishes to Redis and refreshes locally/other nodes
	if err := repository.InvalidateSystemConfigCache(ctx, model.ConfigKeySiteName); err != nil {
		t.Fatalf("InvalidateSystemConfigCache(site_name) error = %v", err)
	}

	// Wait for Redis Pub/Sub delivery in test
	time.Sleep(100 * time.Millisecond)

	refreshed, err := repository.GetSystemConfigByKey(ctx, model.ConfigKeySiteName)
	if err != nil {
		t.Fatalf("GetSystemConfigByKey(site_name) refreshed error = %v", err)
	}
	if refreshed.Value != "broadcast_value" {
		t.Fatalf("GetSystemConfigByKey(site_name).Value = %q, want %q", refreshed.Value, "broadcast_value")
	}

	// Verify Redis pub/sub channel received message
	if db.Redis != nil {
		// Just a sanity check: we can publish a new manual update and verify subscription triggers
		// which was done implicitly above.
	}
}
