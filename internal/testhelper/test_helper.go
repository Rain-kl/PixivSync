// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package testhelper 提供测试辅助工具
package testhelper

import (
	"context"
	"testing"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// SetupTestEnvironment initializes an in-memory SQLite DB, seeds default configurations,
// starts miniredis, and overrides the global db/Redis clients. It returns a cleanup function.
func SetupTestEnvironment(t *testing.T) (*gorm.DB, *miniredis.Miniredis, func()) {
	// Initialize GORM in-memory SQLite
	sqliteDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to open in-memory SQLite db: %v", err)
	}

	// AutoMigrate all tables
	err = sqliteDB.AutoMigrate(
		&model.User{},
		&model.AuthSource{},
		&model.ExternalAccount{},
		&model.SystemConfig{},
		&model.Upload{},
		&model.AccessToken{},
		&model.TaskExecution{},
		&model.Template{},
		&model.PixezPixivUser{},
		&model.PixezBanComment{},
		&model.PixezBanIllust{},
		&model.PixezBanTag{},
		&model.PixezBanUser{},
		&model.PixezIllustHistory{},
		&model.PixezNovelHistory{},
		&model.PixezTagHistory{},
		&model.PixezMirrorIllust{},
		&model.PixezMirrorNovel{},
		&model.PixezBookmarkExportRun{},
		&model.PixezBookmarkIllust{},
		&model.PixezBookmarkNovel{},
	)
	if err != nil {
		t.Fatalf("failed to auto migrate tables: %v", err)
	}

	// Set global db
	db.SetDB(sqliteDB)

	// Start miniredis
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}

	// Hook up Redis Client to miniredis
	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	db.Redis = redisClient

	// Seed default configurations
	seedDefaultConfigs(t, sqliteDB)

	// Cleanup function
	cleanup := func() {
		_ = redisClient.Close()
		mr.Close()
		// Reset database and Redis references
		db.SetDB(nil)
		db.Redis = nil
	}

	return sqliteDB, mr, cleanup
}

func seedDefaultConfigs(t *testing.T, tx *gorm.DB) {
	defaultConfigs := []model.SystemConfig{
		{
			Key:         model.ConfigKeyUploadAllowedExtensions,
			Value:       "jpg,png,webp",
			Type:        "system",
			Description: "允许上传的图片扩展名（逗号分隔）",
		},
		{
			Key:         model.ConfigKeySiteName,
			Value:       "Wavelet",
			Type:        "system",
			Description: "系统平台的展示名称",
		},
		{
			Key:         model.ConfigKeyPasswordLoginEnabled,
			Value:       "true",
			Type:        "system",
			Description: "是否允许使用账号密码登录",
		},
		{
			Key:         model.ConfigKeyRegistrationEnabled,
			Value:       "true",
			Type:        "system",
			Description: "控制普通用户是否可以自主注册（true/false）",
		},
		{
			Key:         model.ConfigKeyPasswordRegisterEnabled,
			Value:       "true",
			Type:        "system",
			Description: "是否允许通过密码创建本地账号",
		},
		{
			Key:         model.ConfigKeyOIDCLoginEnabled,
			Value:       "true",
			Type:        "system",
			Description: "是否允许使用第三方 OIDC 认证源登录",
		},
		{
			Key:         model.ConfigKeyMaxAPIKeysPerUser,
			Value:       "5",
			Type:        "business",
			Description: "限制每个普通用户可以创建的 API Key 最大数量",
		},
		{
			Key:         model.ConfigKeyCapLoginEnabled,
			Value:       "false",
			Type:        "system",
			Description: "是否启用登录人机验证（true/false）",
		},
		{
			Key:         model.ConfigKeyCapAutoSolve,
			Value:       "true",
			Type:        "system",
			Description: "打开页面后是否自动开始计算，关闭则需用户手动点击触发",
		},
		{
			Key:         model.ConfigKeyCapChallengeCount,
			Value:       "1",
			Type:        "system",
			Description: "客户端需求解的 PoW 难题总数，默认 1，推荐 1～5",
		},
		{
			Key:         model.ConfigKeyCapChallengeSize,
			Value:       "32",
			Type:        "system",
			Description: "人机验证盐值长度",
		},
		{
			Key:         model.ConfigKeyCapChallengeDifficulty,
			Value:       "4",
			Type:        "system",
			Description: "人机验证 PoW 难度（目标前缀长度）",
		},
		{
			Key:         model.ConfigKeyCapChallengeTTL,
			Value:       "600",
			Type:        "system",
			Description: "人机验证难题有效时间（秒）",
		},
		{
			Key:         model.ConfigKeyCapTokenTTL,
			Value:       "1200",
			Type:        "system",
			Description: "人机验证兑换凭证有效时间（秒）",
		},
		{
			Key:         model.ConfigKeyServerAddress,
			Value:       "",
			Type:        "system",
			Description: "服务器地址（用于跨域源控制，不设定则允许任意源）",
		},
		{
			Key:         model.ConfigKeySMTPHost,
			Value:       "",
			Type:        "system",
			Description: "SMTP 服务器地址（例如 smtp.example.com）",
		},
		{
			Key:         model.ConfigKeySMTPPort,
			Value:       "587",
			Type:        "system",
			Description: "SMTP 端口（例如 587 或 465）",
		},
		{
			Key:         model.ConfigKeySMTPUsername,
			Value:       "",
			Type:        "system",
			Description: "SMTP 账户（如 sender@example.com）",
		},
		{
			Key:         model.ConfigKeySMTPPassword,
			Value:       "",
			Type:        "system",
			Description: "SMTP 访问凭证（授权码/密码）",
		},
		{
			Key:         model.ConfigKeyEmailLoginVerificationEnabled,
			Value:       "false",
			Type:        "system",
			Description: "是否开启邮箱登录验证（true/false）",
		},
		{
			Key:         model.ConfigKeyEmailRegisterVerificationEnabled,
			Value:       "false",
			Type:        "system",
			Description: "是否开启邮箱注册验证（true/false）",
		},
		{
			Key:         model.ConfigKeyMenuDisplayConfig,
			Value:       "{}",
			Type:        "system",
			Description: "目录显示配置（JSON 字符串，格式为 {url: enabled}）",
		},
		{
			Key:         model.ConfigKeySearchEngineIndexingEnabled,
			Value:       "false",
			Type:        "system",
			Description: "是否允许搜索引擎检索",
		},
	}

	if err := tx.Create(&defaultConfigs).Error; err != nil {
		t.Fatalf("failed to seed default system configs: %v", err)
	}

	publicKeys := map[string]struct{}{
		model.ConfigKeyUploadAllowedExtensions:          {},
		model.ConfigKeySiteName:                         {},
		model.ConfigKeyPasswordLoginEnabled:             {},
		model.ConfigKeyRegistrationEnabled:              {},
		model.ConfigKeyPasswordRegisterEnabled:          {},
		model.ConfigKeyOIDCLoginEnabled:                 {},
		model.ConfigKeyMaxAPIKeysPerUser:                {},
		model.ConfigKeyCapLoginEnabled:                  {},
		model.ConfigKeyCapAutoSolve:                     {},
		model.ConfigKeyEmailLoginVerificationEnabled:    {},
		model.ConfigKeyEmailRegisterVerificationEnabled: {},
		model.ConfigKeyMenuDisplayConfig:                {},
		model.ConfigKeySearchEngineIndexingEnabled:      {},
	}
	keys := make([]string, 0, len(publicKeys))
	for key := range publicKeys {
		keys = append(keys, key)
	}
	if err := tx.Model(&model.SystemConfig{}).
		Where("key IN ?", keys).
		Update("visibility", model.ConfigVisibilityVisible).Error; err != nil {
		t.Fatalf("failed to seed public system config visibility: %v", err)
	}

	// Also seed these in miniredis context if required, but they are stored in postgres first.
	// We'll write configs to miniredis in actual handlers.
	for _, config := range defaultConfigs {
		if _, ok := publicKeys[config.Key]; ok {
			config.Visibility = model.ConfigVisibilityVisible
		}
		_ = db.HSetJSON(context.Background(), model.SystemConfigRedisHashKey, config.Key, &config)
	}
}
