// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"

	"github.com/Rain-kl/Wavelet/internal/db"
)

// 配置键常量 - 所有系统配置的 key 定义
const (
	ConfigKeyUploadAllowedExtensions          = "upload_allowed_extensions"              // 允许上传的文件扩展名，逗号分隔
	ConfigKeySiteName                         = "site_name"                              // 站点名称
	ConfigKeyPasswordLoginEnabled             = "password_login_enabled"                 // 是否允许密码登录
	ConfigKeyRegistrationEnabled              = "registration_enabled"                   // 是否允许注册
	ConfigKeyPasswordRegisterEnabled          = "password_register_enabled"              // 是否允许密码注册
	ConfigKeyOIDCLoginEnabled                 = "oidc_login_enabled"                     // 是否允许 OIDC 登录
	ConfigKeyMaxAPIKeysPerUser                = "max_api_keys_per_user"                  //nolint:gosec // false positive: config key name. 每个用户最大 API Key 数量
	ConfigKeyCapLoginEnabled                  = "cap_login_enabled"                      // 是否启用登录人机验证
	ConfigKeyCapAutoSolve                     = "cap_auto_solve"                         // 打开页面后是否自动开始计算（false 则需用户手动点击）
	ConfigKeyCapChallengeCount                = "cap_challenge_count"                    // 客户端需求解的 PoW 难题总数，默认 1，推荐 1～5
	ConfigKeyCapChallengeSize                 = "cap_challenge_size"                     // 人机验证盐值长度
	ConfigKeyCapChallengeDifficulty           = "cap_challenge_difficulty"               // 人机验证 PoW 难度（目标前缀长度）
	ConfigKeyCapChallengeTTL                  = "cap_challenge_ttl_seconds"              // 人机验证难题有效时间（秒）
	ConfigKeyCapTokenTTL                      = "cap_token_ttl_seconds"                  //nolint:gosec // false positive: config key name. 人机验证兑换凭证有效时间（秒）
	ConfigKeyServerAddress                    = "server_address"                         // 服务器地址
	ConfigKeySMTPHost                         = "smtp_host"                              // SMTP 服务器地址
	ConfigKeySMTPPort                         = "smtp_port"                              // SMTP 端口
	ConfigKeySMTPUsername                     = "smtp_username"                          // SMTP 账户
	ConfigKeySMTPPassword                     = "smtp_password"                          // SMTP 访问凭证
	ConfigKeyEmailLoginVerificationEnabled    = "email_login_verification_enabled"       // 是否启用邮箱登录验证
	ConfigKeyEmailRegisterVerificationEnabled = "email_register_verification_enabled"    // 是否启用邮箱注册验证
	ConfigKeyFileAccessWhitelist              = "file_access_whitelist"                  // 免登录访问的文件业务类型白名单 (JSON 数组格式)
	ConfigKeyDiskCacheMaxSizeMB               = "disk_cache_max_size_mb"                 // 磁盘缓存最大空间大小 (MB)
	ConfigKeyDiskCacheTTLMinutes              = "disk_cache_ttl_minutes"                 // 磁盘缓存默认有效期 (分钟)
	ConfigKeyDiskCacheLRUEnabled              = "disk_cache_lru_enabled"                 // 是否启用 LRU 淘汰机制
	ConfigKeyMenuDisplayConfig                = "menu_display_config"                    // 目录显示配置 (JSON 字符串)
	ConfigKeySearchEngineIndexingEnabled      = "search_engine_indexing_enabled"         // 是否允许搜索引擎检索
	ConfigKeyPixezMirrorDownloadInterval      = "pixez_mirror_download_interval_seconds" // Pixiv插画图片下载间隔（秒）
	ConfigKeyPixezMirrorIllustConcurrency     = "pixez_mirror_illust_concurrency"        // Pixiv插画并发镜像限制数
	ConfigKeyPixezMirrorNovelConcurrency      = "pixez_mirror_novel_concurrency"         // Pixiv小说并发镜像限制数
	ConfigKeyLoginSessionTTLHours             = "login_session_ttl_hours"                // 登录会话过期时间 (小时，0表示浏览器关闭后自动退出登录，-1表示永不过期)

)

const (
	// SystemConfigRedisHashKey Redis Hash key，存储所有系统配置
	SystemConfigRedisHashKey = "system:system_configs"
)

const (
	// ConfigVisibilityHidden 表示配置不通过公共配置接口暴露
	ConfigVisibilityHidden = 0
	// ConfigVisibilityVisible 表示配置通过公共配置接口暴露
	ConfigVisibilityVisible = 1
)

// SystemConfig 系统配置实体
type SystemConfig struct {
	Key         string    `json:"key" gorm:"primaryKey;size:64;not null"`
	Value       string    `json:"value" gorm:"size:255;not null"`
	Type        string    `json:"type" gorm:"size:32;not null;default:'system'"`
	Visibility  int       `json:"visibility" gorm:"not null;default:0"`
	Description string    `json:"description" gorm:"size:255"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName 表名
func (SystemConfig) TableName() string {
	return "w_system_configs"
}

// GetByKey 通过 key 查询配置（带 Redis 缓存）
func (sc *SystemConfig) GetByKey(ctx context.Context, key string) error {
	if db.Redis != nil {
		if err := db.HGetJSON(ctx, SystemConfigRedisHashKey, key, sc); err == nil {
			return nil
		} else if !errors.Is(err, redis.Nil) {
			// Redis 服务错误，返回错误
			return err
		}
	}

	// 查数据库
	database := db.DB(ctx)
	if database == nil {
		return errors.New(errDatabaseNotInitialized)
	}

	if err := database.Where("key = ?", key).First(sc).Error; err != nil {
		return err
	}

	// 更新 Redis Hash 缓存
	if db.Redis != nil {
		_ = db.HSetJSON(ctx, SystemConfigRedisHashKey, key, sc)
	}

	return nil
}

// ListVisibleSystemConfigs 查询所有可通过公共配置接口暴露的配置
func ListVisibleSystemConfigs(ctx context.Context) ([]SystemConfig, error) {
	database := db.DB(ctx)
	if database == nil {
		return nil, errors.New(errDatabaseNotInitialized)
	}

	var configs []SystemConfig
	if err := database.Where("visibility = ?", ConfigVisibilityVisible).Find(&configs).Error; err != nil {
		return nil, err
	}

	return configs, nil
}

// GetIntByKey 通过 key 查询配置并转换为 int 类型
func GetIntByKey(ctx context.Context, key string) (int, error) {
	var sc SystemConfig
	if err := sc.GetByKey(ctx, key); err != nil {
		return 0, err
	}

	value, err := strconv.Atoi(sc.Value)
	if err != nil {
		return 0, fmt.Errorf(errConfigIntParseFailed, key, sc.Value, err)
	}

	return value, nil
}

// GetDecimalByKey 通过 key 查询配置并转换为 decimal.Decimal 类型
// precision 指定保留的小数位数，多余的小数会被裁剪
func GetDecimalByKey(ctx context.Context, key string, precision int32) (decimal.Decimal, error) {
	var sc SystemConfig
	if err := sc.GetByKey(ctx, key); err != nil {
		return decimal.Zero, err
	}

	value, err := decimal.NewFromString(sc.Value)
	if err != nil {
		return decimal.Zero, fmt.Errorf(errConfigDecimalParseFailed, key, sc.Value, err)
	}

	// 裁剪到指定小数位数
	return value.Truncate(precision), nil
}

// GetBoolByKey 通过 key 查询配置并转换为 bool 类型
func GetBoolByKey(ctx context.Context, key string) (bool, error) {
	var sc SystemConfig
	if err := sc.GetByKey(ctx, key); err != nil {
		return false, err
	}

	value, err := strconv.ParseBool(sc.Value)
	if err != nil {
		return false, fmt.Errorf(errConfigBoolParseFailed, key, sc.Value, err)
	}

	return value, nil
}

// GetMenuDisplayConfig 获取目录显示配置，解析为 map[string]bool
func GetMenuDisplayConfig(ctx context.Context) (map[string]bool, error) {
	var sc SystemConfig
	if err := sc.GetByKey(ctx, ConfigKeyMenuDisplayConfig); err != nil {
		return nil, err
	}

	config := make(map[string]bool)
	if sc.Value == "" || sc.Value == "{}" {
		return config, nil
	}

	if err := json.Unmarshal([]byte(sc.Value), &config); err != nil {
		return nil, fmt.Errorf(errParseMenuDisplayConfigFailed, err)
	}

	return config, nil
}
