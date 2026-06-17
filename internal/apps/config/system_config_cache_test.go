// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"errors"
	"testing"

	"github.com/redis/go-redis/v9"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/testhelper"
)

func TestSystemConfigRAMCacheServesUntilInvalidated(t *testing.T) {
	dbConn, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	model.ResetSystemConfigRAMCacheForTest()
	if err := model.InvalidateAllSystemConfigCaches(ctx); err != nil {
		t.Fatalf("InvalidateAllSystemConfigCaches() error = %v", err)
	}

	var warm model.SystemConfig
	if err := warm.GetByKey(ctx, model.ConfigKeySiteName); err != nil {
		t.Fatalf("GetByKey(site_name) warm error = %v", err)
	}
	if warm.Value != "Wavelet" {
		t.Fatalf("GetByKey(site_name).Value = %q, want %q", warm.Value, "Wavelet")
	}

	if err := dbConn.Model(&model.SystemConfig{}).
		Where("key = ?", model.ConfigKeySiteName).
		Update("value", "ram_probe_value").Error; err != nil {
		t.Fatalf("Update(site_name) error = %v", err)
	}
	if err := db.HDel(ctx, model.SystemConfigRedisHashKey, model.ConfigKeySiteName); err != nil {
		t.Fatalf("HDel(site_name) error = %v", err)
	}

	var cached model.SystemConfig
	if err := cached.GetByKey(ctx, model.ConfigKeySiteName); err != nil {
		t.Fatalf("GetByKey(site_name) cached error = %v", err)
	}
	if cached.Value != "Wavelet" {
		t.Fatalf("GetByKey(site_name).Value = %q, want stale RAM value %q", cached.Value, "Wavelet")
	}

	if err := model.InvalidateSystemConfigCache(ctx, model.ConfigKeySiteName); err != nil {
		t.Fatalf("InvalidateSystemConfigCache(site_name) error = %v", err)
	}

	var refreshed model.SystemConfig
	if err := refreshed.GetByKey(ctx, model.ConfigKeySiteName); err != nil {
		t.Fatalf("GetByKey(site_name) refreshed error = %v", err)
	}
	if refreshed.Value != "ram_probe_value" {
		t.Fatalf("GetByKey(site_name).Value = %q, want %q", refreshed.Value, "ram_probe_value")
	}

	exists, err := db.Redis.HExists(ctx, db.PrefixedKey(model.SystemConfigRedisHashKey), model.ConfigKeySiteName).Result()
	if err != nil {
		t.Fatalf("HExists(site_name) error = %v", err)
	}
	if !exists {
		t.Fatal("expected redis hash field to be repopulated after refresh")
	}
}

func TestInvalidateSystemConfigCacheClearsRedisField(t *testing.T) {
	dbConn, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	var sc model.SystemConfig
	if err := sc.GetByKey(ctx, model.ConfigKeySiteName); err != nil {
		t.Fatalf("GetByKey(site_name) error = %v", err)
	}

	if err := model.InvalidateSystemConfigCache(ctx, model.ConfigKeySiteName); err != nil {
		t.Fatalf("InvalidateSystemConfigCache(site_name) error = %v", err)
	}

	_, err := db.Redis.HGet(ctx, db.PrefixedKey(model.SystemConfigRedisHashKey), model.ConfigKeySiteName).Result()
	if !errors.Is(err, redis.Nil) {
		t.Fatalf("HGet(site_name) error = %v, want redis.Nil", err)
	}

	if err := dbConn.Model(&model.SystemConfig{}).
		Where("key = ?", model.ConfigKeySiteName).
		Update("value", "after_invalidate").Error; err != nil {
		t.Fatalf("Update(site_name) error = %v", err)
	}

	var refreshed model.SystemConfig
	if err := refreshed.GetByKey(ctx, model.ConfigKeySiteName); err != nil {
		t.Fatalf("GetByKey(site_name) refreshed error = %v", err)
	}
	if refreshed.Value != "after_invalidate" {
		t.Fatalf("GetByKey(site_name).Value = %q, want %q", refreshed.Value, "after_invalidate")
	}
}
