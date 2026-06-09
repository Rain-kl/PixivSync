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

package model

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"gorm.io/gorm"
)

// 认证源类型
const (
	AuthSourceTypeOIDC = "oidc"
)

var authSourceNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,79}$`)

// AuthSource 认证源实体
type AuthSource struct {
	ID                     uint64    `json:"id" gorm:"primaryKey"`
	Name                   string    `json:"name" gorm:"uniqueIndex;size:80;not null"`
	Type                   string    `json:"type" gorm:"size:20;not null"`
	DisplayName            string    `json:"display_name" gorm:"size:100"`
	IsActive               bool      `json:"is_active" gorm:"index;not null;default:false"`
	ClientID               string    `json:"client_id" gorm:"size:255"`
	ClientSecret           string    `json:"-" gorm:"size:1024"`
	OpenIDDiscoveryURL     string    `json:"openid_discovery_url" gorm:"column:openid_discovery_url;size:1024"`
	Scopes                 string    `json:"scopes" gorm:"size:255"`
	IconURL                string    `json:"icon_url" gorm:"size:1024"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
	ClientSecretConfigured bool      `json:"client_secret_configured" gorm:"-"`
}

// ExternalAccount 外部账号绑定实体
type ExternalAccount struct {
	ID               uint64    `json:"id" gorm:"primaryKey"`
	AuthSourceID     uint64    `json:"auth_source_id" gorm:"uniqueIndex:idx_external_accounts_source_external,priority:1;index"`
	UserID           uint64    `json:"user_id" gorm:"index;not null"`
	ExternalID       string    `json:"external_id" gorm:"uniqueIndex:idx_external_accounts_source_external,priority:2;size:255;not null"`
	ExternalUsername string    `json:"external_username" gorm:"size:255"`
	Email            string    `json:"email" gorm:"size:255"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ExternalAccountView 外部帐号绑定视图（脱敏展示用）
type ExternalAccountView struct {
	ID               uint64    `json:"id"`
	AuthSourceID     uint64    `json:"auth_source_id"`
	AuthSourceName   string    `json:"auth_source_name"`
	AuthSourceType   string    `json:"auth_source_type"`
	AuthSourceLabel  string    `json:"auth_source_label"`
	ExternalUsername string    `json:"external_username"`
	Email            string    `json:"email"`
	CreatedAt        time.Time `json:"created_at"`
}

// Normalize 对认证源字段进行标准化处理
func (source *AuthSource) Normalize() {
	source.Type = strings.ToLower(strings.TrimSpace(source.Type))
	source.Name = strings.TrimSpace(source.Name)
	source.DisplayName = strings.TrimSpace(source.DisplayName)
	source.ClientID = strings.TrimSpace(source.ClientID)
	source.ClientSecret = strings.TrimSpace(source.ClientSecret)
	source.OpenIDDiscoveryURL = strings.TrimSpace(source.OpenIDDiscoveryURL)
	source.Scopes = strings.TrimSpace(source.Scopes)
	source.IconURL = strings.TrimSpace(source.IconURL)
	if source.DisplayName == "" {
		source.DisplayName = source.Name
	}
	if source.Type == AuthSourceTypeOIDC && source.Scopes == "" {
		source.Scopes = "openid profile email"
	}
}

// Validate 校验认证源字段合法性
func (source *AuthSource) Validate() error {
	source.Normalize()
	if source.Name == "" {
		return errors.New(errAuthSourceNameRequired)
	}
	if !authSourceNamePattern.MatchString(source.Name) {
		return errors.New(errAuthSourceNameInvalid)
	}
	if source.Type != AuthSourceTypeOIDC {
		return errors.New(errAuthSourceTypeUnsupported)
	}
	if source.OpenIDDiscoveryURL == "" {
		return errors.New(errAuthSourceDiscoveryURLRequired)
	}
	if source.IsActive && (source.ClientID == "" || source.ClientSecret == "") {
		return errors.New(errAuthSourceClientCredentialsRequired)
	}
	return nil
}

// Sanitize 脱敏处理，将 ClientSecret 清空并设置 ClientSecretConfigured 标志
func (source *AuthSource) Sanitize() {
	source.ClientSecretConfigured = source.ClientSecret != ""
	source.ClientSecret = ""
}

// GetAuthSources 获取所有认证源（已脱敏）
func GetAuthSources() ([]AuthSource, error) {
	var sources []AuthSource
	if err := db.DB(context.Background()).Order("id asc").Find(&sources).Error; err != nil {
		return nil, err
	}
	for i := range sources {
		sources[i].Sanitize()
	}
	return sources, nil
}

// GetActiveAuthSources 获取所有已启用的认证源（已脱敏）
func GetActiveAuthSources() ([]AuthSource, error) {
	var sources []AuthSource
	if err := db.DB(context.Background()).Where("is_active = ?", true).Order("id asc").Find(&sources).Error; err != nil {
		return nil, err
	}
	for i := range sources {
		sources[i].Sanitize()
	}
	return sources, nil
}

// GetAuthSourceByID 根据 ID 获取认证源
func GetAuthSourceByID(id uint64) (*AuthSource, error) {
	if id == 0 {
		return nil, errors.New(errAuthSourceIDRequired)
	}
	var source AuthSource
	if err := db.DB(context.Background()).First(&source, "id = ?", id).Error; err != nil {
		return nil, err
	}
	source.ClientSecretConfigured = source.ClientSecret != ""
	return &source, nil
}

// GetAuthSourceByName 根据名称获取认证源
func GetAuthSourceByName(name string) (*AuthSource, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New(errAuthSourceNameRequired)
	}
	var source AuthSource
	if err := db.DB(context.Background()).First(&source, "name = ?", name).Error; err != nil {
		return nil, err
	}
	source.ClientSecretConfigured = source.ClientSecret != ""
	return &source, nil
}

// CreateAuthSource 创建认证源
func CreateAuthSource(source *AuthSource) error {
	if err := source.Validate(); err != nil {
		return err
	}
	return db.DB(context.Background()).Create(source).Error
}

// UpdateAuthSource 更新认证源，keepSecret 为 true 时保留原密钥
func UpdateAuthSource(source *AuthSource, keepSecret bool) error {
	if source.ID == 0 {
		return errors.New(errAuthSourceIDRequired)
	}
	var current AuthSource
	if err := db.DB(context.Background()).First(&current, "id = ?", source.ID).Error; err != nil {
		return err
	}
	if keepSecret {
		source.ClientSecret = current.ClientSecret
	}
	if err := source.Validate(); err != nil {
		return err
	}
	return db.DB(context.Background()).Model(&current).Updates(map[string]any{
		"name":                 source.Name,
		"type":                 source.Type,
		"display_name":         source.DisplayName,
		"is_active":            source.IsActive,
		"client_id":            source.ClientID,
		"client_secret":        source.ClientSecret,
		"openid_discovery_url": source.OpenIDDiscoveryURL,
		"scopes":               source.Scopes,
		"icon_url":             source.IconURL,
	}).Error
}

// ToggleAuthSource 切换认证源启用状态
func ToggleAuthSource(id uint64, isActive bool) error {
	source, err := GetAuthSourceByID(id)
	if err != nil {
		return err
	}
	source.IsActive = isActive
	if err := source.Validate(); err != nil {
		return err
	}
	return db.DB(context.Background()).Model(&AuthSource{}).Where("id = ?", id).Update("is_active", isActive).Error
}

// DeleteAuthSource 删除认证源及其关联的外部帐号绑定
func DeleteAuthSource(id uint64) error {
	if id == 0 {
		return errors.New(errAuthSourceIDRequired)
	}
	return db.DB(context.Background()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("auth_source_id = ?", id).Delete(&ExternalAccount{}).Error; err != nil {
			return err
		}
		return tx.Delete(&AuthSource{}, "id = ?", id).Error
	})
}

// FindExternalAccount 查找外部帐号绑定记录
func FindExternalAccount(sourceID uint64, externalID string) (*ExternalAccount, error) {
	var account ExternalAccount
	if err := db.DB(context.Background()).Where("auth_source_id = ? AND external_id = ?", sourceID, externalID).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

// BindExternalAccount 绑定外部帐号（已存在时更新用户名和邮箱）
func BindExternalAccount(account *ExternalAccount) error {
	if account.UserID == 0 || strings.TrimSpace(account.ExternalID) == "" {
		return errors.New(errExternalAccountBindingIncomplete)
	}
	account.ExternalID = strings.TrimSpace(account.ExternalID)
	account.ExternalUsername = strings.TrimSpace(account.ExternalUsername)
	account.Email = strings.TrimSpace(account.Email)

	return db.DB(context.Background()).Transaction(func(tx *gorm.DB) error {
		var current ExternalAccount
		err := tx.Where("auth_source_id = ? AND external_id = ?", account.AuthSourceID, account.ExternalID).First(&current).Error
		if err == nil {
			if current.UserID != account.UserID {
				return errors.New(errExternalAccountAlreadyBoundToAnother)
			}
			return tx.Model(&current).Updates(map[string]any{
				"external_username": account.ExternalUsername,
				"email":             account.Email,
			}).Error
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return tx.Create(account).Error
	})
}

// ListExternalAccountsByUserID 获取指定用户的所有外部帐号绑定视图
func ListExternalAccountsByUserID(userID uint64) ([]ExternalAccountView, error) {
	if userID == 0 {
		return nil, errors.New(errUserIDRequired)
	}
	var accounts []ExternalAccount
	if err := db.DB(context.Background()).Where("user_id = ?", userID).Order("id asc").Find(&accounts).Error; err != nil {
		return nil, err
	}
	views := make([]ExternalAccountView, 0, len(accounts))
	for _, account := range accounts {
		var name, sourceType, label string
		if account.AuthSourceID == 0 {
			name = "default"
			sourceType = "oidc"
			label = "历史认证源"
		} else {
			source, err := GetAuthSourceByID(account.AuthSourceID)
			if err != nil {
				continue
			}
			name = source.Name
			sourceType = source.Type
			label = source.DisplayName
			if label == "" {
				label = source.Name
			}
		}
		views = append(views, ExternalAccountView{
			ID:               account.ID,
			AuthSourceID:     account.AuthSourceID,
			AuthSourceName:   name,
			AuthSourceType:   sourceType,
			AuthSourceLabel:  label,
			ExternalUsername: account.ExternalUsername,
			Email:            account.Email,
			CreatedAt:        account.CreatedAt,
		})
	}
	return views, nil
}

// DeleteExternalAccountForUser 删除指定用户的外部帐号绑定
func DeleteExternalAccountForUser(id uint64, userID uint64) error {
	if id == 0 || userID == 0 {
		return errors.New(errExternalAccountBindingIDRequired)
	}
	return db.DB(context.Background()).Where("id = ? AND user_id = ?", id, userID).Delete(&ExternalAccount{}).Error
}
