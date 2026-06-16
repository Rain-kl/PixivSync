// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package model

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"gorm.io/gorm"
)

const (
	// TypeCustom 自定义消息通道类型
	TypeCustom = "custom"
	// TypeEmail 邮件推送消息通道类型
	TypeEmail = "email"
	// TypeTelegram 电报机器人推送消息通道类型
	TypeTelegram = "telegram"
)

// PushChannel 消息通道模型
type PushChannel struct {
	ID          uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string    `json:"name" gorm:"uniqueIndex;size:80;not null"`      // 通道名称，仅英文字母和下划线，唯一
	Description string    `json:"description" gorm:"size:255"`                   // 备注
	Type        string    `json:"type" gorm:"size:50;not null;default:'custom'"` // 通道类型：custom, lark, email
	Token       string    `json:"token" gorm:"size:100"`                         // 鉴权令牌或发信用户名等
	URL         string    `json:"url" gorm:"type:text;not null"`                 // 请求地址，HTTPS 协议或 SMTP 地址
	Other       string    `json:"other" gorm:"type:text;not null"`               // 请求体/SMTP 密码等
	Enabled     bool      `json:"enabled" gorm:"index;not null;default:true"`    // 通道是否启用
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime;index"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime;index"`
}

// TableName 指定 GORM 表名
func (PushChannel) TableName() string {
	return "w_push_channels"
}

var nameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// Validate 参数合法性与 JSON 格式校验
func (pc *PushChannel) Validate() error {
	pc.Name = strings.TrimSpace(pc.Name)
	pc.URL = strings.TrimSpace(pc.URL)
	pc.Other = strings.TrimSpace(pc.Other)
	pc.Type = strings.TrimSpace(pc.Type)

	if pc.Type == "" {
		pc.Type = TypeCustom
	}

	if pc.Type == TypeTelegram && pc.URL == "" {
		pc.URL = "https://api.telegram.org"
	}

	if pc.Name == "" {
		return errors.New("channel name is required")
	}
	if !nameRegex.MatchString(pc.Name) {
		return errors.New("channel name can only contain letters, numbers, and underscores")
	}
	if pc.Type != TypeEmail && pc.URL == "" {
		return errors.New("request URL/address is required")
	}

	// For custom and lark, we must enforce https:// URL prefix for security.
	// For email, it is an SMTP host:port, so no need for https:// prefix.
	if pc.Type != TypeEmail && !strings.HasPrefix(pc.URL, "https://") {
		return errors.New("request URL must use HTTPS protocol for security reasons")
	}

	switch pc.Type {
	case TypeCustom:
		if pc.Other == "" {
			return errors.New("payload schema (request body) is required")
		}
		return validateJSON(pc.Other)
	case TypeEmail:
		// Email channel SMTP configs fall back to global settings, so they are not required to be filled.
	case TypeTelegram:
		if pc.Token == "" {
			return errors.New("telegram bot token is required")
		}
	}
	return nil
}

func validateJSON(s string) error {
	var jsonTest map[string]any
	if err := json.Unmarshal([]byte(s), &jsonTest); err == nil {
		return nil
	}
	var jsonArr []any
	if err := json.Unmarshal([]byte(s), &jsonArr); err == nil {
		return nil
	}
	return errors.New("payload schema must be a valid JSON format")
}

// GetPushChannelByName 根据名称获取消息通道
func GetPushChannelByName(ctx context.Context, name string) (*PushChannel, error) {
	var channel PushChannel
	err := db.DB(ctx).Where("name = ?", name).First(&channel).Error
	if err != nil {
		return nil, err
	}
	return &channel, nil
}

const activePushChannelCacheTTL = 24 * time.Hour

// GetActivePushChannelByName 根据名称获取启用的消息通道 (优先从 Redis 缓存获取)
func GetActivePushChannelByName(ctx context.Context, name string) (*PushChannel, error) {
	cacheKey := "push:channel:active:" + name
	var channel PushChannel
	if db.Redis != nil {
		if err := db.GetJSON(ctx, cacheKey, &channel); err == nil {
			return &channel, nil
		}
	}

	err := db.DB(ctx).Where("name = ? AND enabled = ?", name, true).First(&channel).Error
	if err != nil {
		return nil, err
	}

	if db.Redis != nil {
		// 缓存有效时间设置为 24 小时
		_ = db.SetJSON(ctx, cacheKey, channel, activePushChannelCacheTTL)
	}

	return &channel, nil
}

// DeleteActivePushChannelCache 清理启用消息通道的缓存
func DeleteActivePushChannelCache(ctx context.Context, name string) {
	if db.Redis != nil {
		_ = db.Redis.Del(ctx, db.PrefixedKey("push:channel:active:"+name)).Err()
	}
}

// AfterSave GORM 保存后钩子，用于自动清理缓存
func (pc *PushChannel) AfterSave(tx *gorm.DB) error {
	DeleteActivePushChannelCache(tx.Statement.Context, pc.Name)
	return nil
}

// AfterDelete GORM 删除后钩子，用于自动清理缓存
func (pc *PushChannel) AfterDelete(tx *gorm.DB) error {
	DeleteActivePushChannelCache(tx.Statement.Context, pc.Name)
	return nil
}
