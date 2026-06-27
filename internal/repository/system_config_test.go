// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/maintnotifications"
	"gorm.io/gorm"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
)

func setupSystemConfigTest(t *testing.T) (*gorm.DB, func()) {
	t.Helper()

	sqliteDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("gorm.Open(sqlite) error = %v", err)
	}
	if err := sqliteDB.AutoMigrate(&model.SystemConfig{}); err != nil {
		t.Fatalf("AutoMigrate(SystemConfig) error = %v", err)
	}

	siteConfig := model.SystemConfig{
		Key:         model.ConfigKeySiteName,
		Value:       "Wavelet",
		Type:        "system",
		Description: "系统平台的展示名称",
	}
	if err := sqliteDB.Create(&siteConfig).Error; err != nil {
		t.Fatalf("Create(site_name) error = %v", err)
	}

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
		MaintNotificationsConfig: &maintnotifications.Config{
			Mode: maintnotifications.ModeDisabled,
		},
	})

	previousRedis := db.Redis
	db.SetDB(sqliteDB)
	db.Redis = redisClient

	cleanup := func() {
		StopSystemConfigCacheListener()
		ResetSystemConfigRAMCacheForTest()
		db.SetDB(nil)
		db.Redis = previousRedis
		_ = redisClient.Close()
		mr.Close()
	}

	return sqliteDB, cleanup
}

func TestListSystemConfigsByKeys_EmptyKeys(t *testing.T) {
	result, err := ListSystemConfigsByKeys(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListSystemConfigsByKeys(nil) error = %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("ListSystemConfigsByKeys(nil) = %#v, want empty map", result)
	}
}

func TestListSystemConfigsByKeys_LoadsFromRAMCache(t *testing.T) {
	dbConn, cleanup := setupSystemConfigTest(t)
	defer cleanup()
	ctx := context.Background()

	ResetSystemConfigRAMCacheForTest()

	// Initial load
	warm, err := GetSystemConfigByKey(ctx, model.ConfigKeySiteName)
	if err != nil {
		t.Fatalf("GetSystemConfigByKey(site_name) warm error = %v", err)
	}
	if warm.Value != "Wavelet" {
		t.Fatalf("GetSystemConfigByKey(site_name).Value = %q, want %q", warm.Value, "Wavelet")
	}

	// Update DB directly
	if err := dbConn.Model(&model.SystemConfig{}).
		Where("key = ?", model.ConfigKeySiteName).
		Update("value", "db_only_value").Error; err != nil {
		t.Fatalf("Update(site_name) error = %v", err)
	}

	// Fetch via ListSystemConfigsByKeys should serve from local store (meaning the old value "Wavelet")
	configs, err := ListSystemConfigsByKeys(ctx, []string{model.ConfigKeySiteName})
	if err != nil {
		t.Fatalf("ListSystemConfigsByKeys(site_name) error = %v", err)
	}

	sc, ok := configs[model.ConfigKeySiteName]
	if !ok {
		t.Fatal("ListSystemConfigsByKeys(site_name) missing site_name entry")
	}
	if sc.Value != "Wavelet" {
		t.Fatalf("ListSystemConfigsByKeys(site_name).Value = %q, want cached value %q", sc.Value, "Wavelet")
	}
}

func TestGetSystemConfigByGroupAndInvalidation(t *testing.T) {
	dbConn, cleanup := setupSystemConfigTest(t)
	defer cleanup()
	ctx := context.Background()

	ResetSystemConfigRAMCacheForTest()

	// Get via specific group/type
	cfg, err := GetSystemConfigByGroup(ctx, ConfigCacheType, model.ConfigKeySiteName)
	if err != nil {
		t.Fatalf("GetSystemConfigByGroup error = %v", err)
	}
	if cfg.Value != "Wavelet" {
		t.Fatalf("value = %q, want %q", cfg.Value, "Wavelet")
	}

	// Direct DB update
	if err := dbConn.Model(&model.SystemConfig{}).
		Where("key = ?", model.ConfigKeySiteName).
		Update("value", "new_site_name").Error; err != nil {
		t.Fatalf("DB Update error = %v", err)
	}

	// Invalidate
	if err := InvalidateSystemConfigCache(ctx, model.ConfigKeySiteName); err != nil {
		t.Fatalf("InvalidateSystemConfigCache error = %v", err)
	}

	// Wait for broadcast execution
	time.Sleep(100 * time.Millisecond)

	// Fetch again
	updated, err := GetSystemConfigByKey(ctx, model.ConfigKeySiteName)
	if err != nil {
		t.Fatalf("GetSystemConfigByKey error = %v", err)
	}
	if updated.Value != "new_site_name" {
		t.Fatalf("value = %q, want %q", updated.Value, "new_site_name")
	}
}
