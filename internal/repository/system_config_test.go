// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package repository

import (
	"context"
	"testing"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/maintnotifications"
	"gorm.io/gorm"
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

func TestListSystemConfigsByKeys_LoadsFromRedisBeforeDB(t *testing.T) {
	dbConn, cleanup := setupSystemConfigTest(t)
	defer cleanup()
	ctx := context.Background()

	ResetSystemConfigRAMCacheForTest()
	if err := InvalidateAllSystemConfigCaches(ctx); err != nil {
		t.Fatalf("InvalidateAllSystemConfigCaches() error = %v", err)
	}

	warm, err := GetSystemConfigByKey(ctx, model.ConfigKeySiteName)
	if err != nil {
		t.Fatalf("GetSystemConfigByKey(site_name) warm error = %v", err)
	}
	if warm.Value != "Wavelet" {
		t.Fatalf("GetSystemConfigByKey(site_name).Value = %q, want %q", warm.Value, "Wavelet")
	}

	if err := dbConn.Model(&model.SystemConfig{}).
		Where("key = ?", model.ConfigKeySiteName).
		Update("value", "db_only_value").Error; err != nil {
		t.Fatalf("Update(site_name) error = %v", err)
	}

	ResetSystemConfigRAMCacheForTest()

	configs, err := ListSystemConfigsByKeys(ctx, []string{model.ConfigKeySiteName})
	if err != nil {
		t.Fatalf("ListSystemConfigsByKeys(site_name) error = %v", err)
	}

	sc, ok := configs[model.ConfigKeySiteName]
	if !ok {
		t.Fatal("ListSystemConfigsByKeys(site_name) missing site_name entry")
	}
	if sc.Value != "Wavelet" {
		t.Fatalf("ListSystemConfigsByKeys(site_name).Value = %q, want redis value %q", sc.Value, "Wavelet")
	}
}

func TestListSystemConfigsByKeys_PopulatesRAMFromRedis(t *testing.T) {
	_, cleanup := setupSystemConfigTest(t)
	defer cleanup()
	ctx := context.Background()

	ResetSystemConfigRAMCacheForTest()
	if err := InvalidateAllSystemConfigCaches(ctx); err != nil {
		t.Fatalf("InvalidateAllSystemConfigCaches() error = %v", err)
	}

	if _, err := GetSystemConfigByKey(ctx, model.ConfigKeySiteName); err != nil {
		t.Fatalf("GetSystemConfigByKey(site_name) warm error = %v", err)
	}

	ResetSystemConfigRAMCacheForTest()

	if _, err := ListSystemConfigsByKeys(ctx, []string{model.ConfigKeySiteName}); err != nil {
		t.Fatalf("ListSystemConfigsByKeys(site_name) error = %v", err)
	}

	cached, ok := systemConfigRAMCache.GetIfPresent(model.ConfigKeySiteName)
	if !ok {
		t.Fatal("expected RAM cache to be populated after redis hit")
	}
	if cached.Value != "Wavelet" {
		t.Fatalf("RAM cache value = %q, want %q", cached.Value, "Wavelet")
	}
}