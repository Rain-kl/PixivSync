/*
Copyright 2026 Arctel.net

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package migrator

import (
	"context"
	"testing"

	"github.com/Rain-kl/Wavelet/internal/config"
	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func TestMigrateInitializesSQLiteDatabase(t *testing.T) {
	sqliteDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("gorm.Open(sqlite) error = %v", err)
	}

	previousDBEnabled := config.Config.Database.Enabled
	config.Config.Database.Enabled = false
	db.SetDB(sqliteDB)
	t.Cleanup(func() {
		config.Config.Database.Enabled = previousDBEnabled
		db.SetDB(nil)
	})

	Migrate()

	var systemConfigCount int64
	if err := sqliteDB.Table("system_configs").Count(&systemConfigCount).Error; err != nil {
		t.Fatalf("Migrate() count system_configs error = %v", err)
	}
	if systemConfigCount != 23 {
		t.Errorf("Migrate() system_configs count = %d, want %d", systemConfigCount, 23)
	}

	var adminCount int64
	if err := sqliteDB.Table("users").Where("username = ?", "admin").Count(&adminCount).Error; err != nil {
		t.Fatalf("Migrate() count admin user error = %v", err)
	}
	if adminCount != 1 {
		t.Errorf("Migrate() admin user count = %d, want %d", adminCount, 1)
	}

	var templateCount int64
	if err := sqliteDB.Table("templates").Count(&templateCount).Error; err != nil {
		t.Fatalf("Migrate() count templates error = %v", err)
	}
	if templateCount != 2 {
		t.Errorf("Migrate() templates count = %d, want %d", templateCount, 2)
	}
}

func TestMigrateClearsStaleSystemConfigCache(t *testing.T) {
	sqliteDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("gorm.Open(sqlite) error = %v", err)
	}

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	previousDBEnabled := config.Config.Database.Enabled
	previousRedis := db.Redis
	config.Config.Database.Enabled = false
	db.SetDB(sqliteDB)
	db.Redis = redisClient
	t.Cleanup(func() {
		config.Config.Database.Enabled = previousDBEnabled
		db.SetDB(nil)
		db.Redis = previousRedis
		_ = redisClient.Close()
		mr.Close()
	})

	staleConfig := model.SystemConfig{
		Key:   model.ConfigKeyCapLoginEnabled,
		Value: "true",
		Type:  "system",
	}
	if err := db.HSetJSON(context.Background(), model.SystemConfigRedisHashKey, model.ConfigKeyCapLoginEnabled, &staleConfig); err != nil {
		t.Fatalf("HSetJSON() error = %v", err)
	}

	Migrate()

	exists, err := db.Redis.Exists(context.Background(), db.PrefixedKey(model.SystemConfigRedisHashKey)).Result()
	if err != nil {
		t.Fatalf("Redis.Exists() error = %v", err)
	}
	if exists != 0 {
		t.Fatalf("system config cache exists = %d, want 0", exists)
	}

	enabled, err := model.GetBoolByKey(context.Background(), model.ConfigKeyCapLoginEnabled)
	if err != nil {
		t.Fatalf("GetBoolByKey(%s) error = %v", model.ConfigKeyCapLoginEnabled, err)
	}
	if enabled {
		t.Fatalf("GetBoolByKey(%s) = true, want false", model.ConfigKeyCapLoginEnabled)
	}
}
