// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package model

import "time"

// PixezPixivUser stores Pixiv account credentials synced from the Flutter client.
type PixezPixivUser struct {
	PixivUserID      string    `gorm:"primaryKey;column:pixiv_user_id" json:"pixiv_user_id"`
	Name             string    `gorm:"not null" json:"name"`
	Account          string    `gorm:"not null" json:"account"`
	MailAddress      string    `json:"mail_address"`
	UserImage        string    `json:"user_image"`
	AccessToken      string    `gorm:"not null" json:"access_token"`
	RefreshToken     string    `gorm:"not null" json:"refresh_token"`
	DeviceToken      string    `json:"device_token"`
	IsPremium        int       `gorm:"default:0" json:"is_premium"`
	XRestrict        int       `gorm:"default:0" json:"x_restrict"`
	IsMailAuthorized int       `gorm:"default:0" json:"is_mail_authorized"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// TableName returns the legacy PixEz sync table name.
func (PixezPixivUser) TableName() string {
	return "pixiv_users"
}

// PixezPixivUserSafeDTO is the list representation that omits sensitive tokens.
type PixezPixivUserSafeDTO struct {
	PixivUserID      string    `json:"pixiv_user_id"`
	Name             string    `json:"name"`
	Account          string    `json:"account"`
	MailAddress      string    `json:"mail_address"`
	UserImage        string    `json:"user_image"`
	IsPremium        int       `json:"is_premium"`
	XRestrict        int       `json:"x_restrict"`
	IsMailAuthorized int       `json:"is_mail_authorized"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ToSafeDTO returns a token-free representation for account lists.
func (u PixezPixivUser) ToSafeDTO() PixezPixivUserSafeDTO {
	return PixezPixivUserSafeDTO{
		PixivUserID:      u.PixivUserID,
		Name:             u.Name,
		Account:          u.Account,
		MailAddress:      u.MailAddress,
		UserImage:        u.UserImage,
		IsPremium:        u.IsPremium,
		XRestrict:        u.XRestrict,
		IsMailAuthorized: u.IsMailAuthorized,
		CreatedAt:        u.CreatedAt,
		UpdatedAt:        u.UpdatedAt,
	}
}

// PixezBanComment stores a synced blocked comment record.
type PixezBanComment struct {
	ID          uint   `gorm:"primaryKey;autoIncrement" json:"-"`
	PixivUserID string `gorm:"column:pixiv_user_id;not null;index" json:"-"`
	CommentID   string `gorm:"column:comment_id;not null" json:"comment_id"`
	Name        string `gorm:"column:name;not null" json:"name"`
}

// TableName returns the legacy PixEz sync table name.
func (PixezBanComment) TableName() string { return "ban_comments" }

// PixezBanIllust stores a synced blocked illustration record.
type PixezBanIllust struct {
	ID          uint   `gorm:"primaryKey;autoIncrement" json:"-"`
	PixivUserID string `gorm:"column:pixiv_user_id;not null;index" json:"-"`
	IllustID    string `gorm:"column:illust_id;not null" json:"illust_id"`
	Name        string `gorm:"column:name;not null" json:"name"`
}

// TableName returns the legacy PixEz sync table name.
func (PixezBanIllust) TableName() string { return "ban_illusts" }

// PixezBanTag stores a synced blocked tag record.
type PixezBanTag struct {
	ID            uint   `gorm:"primaryKey;autoIncrement" json:"-"`
	PixivUserID   string `gorm:"column:pixiv_user_id;not null;index" json:"-"`
	Name          string `gorm:"column:name;not null" json:"name"`
	TranslateName string `gorm:"column:translate_name;not null" json:"translate_name"`
}

// TableName returns the legacy PixEz sync table name.
func (PixezBanTag) TableName() string { return "ban_tags" }

// PixezBanUser stores a synced blocked Pixiv user record.
type PixezBanUser struct {
	ID          uint   `gorm:"primaryKey;autoIncrement" json:"-"`
	PixivUserID string `gorm:"column:pixiv_user_id;not null;index" json:"-"`
	UserID      string `gorm:"column:user_id;not null" json:"user_id"`
	Name        string `gorm:"column:name;not null" json:"name"`
}

// TableName returns the legacy PixEz sync table name.
func (PixezBanUser) TableName() string { return "ban_users" }

// PixezIllustHistory stores a synced illustration browsing history record.
type PixezIllustHistory struct {
	ID          uint   `gorm:"primaryKey;autoIncrement" json:"-"`
	PixivUserID string `gorm:"column:pixiv_user_id;not null;index" json:"-"`
	IllustID    int    `gorm:"column:illust_id;not null" json:"illust_id"`
	UserID      int    `gorm:"column:user_id;not null" json:"user_id"`
	PictureURL  string `gorm:"column:picture_url;not null" json:"picture_url"`
	Title       string `gorm:"column:title" json:"title"`
	UserName    string `gorm:"column:user_name" json:"user_name"`
	Time        int64  `gorm:"column:time;not null" json:"time"`
}

// TableName returns the legacy PixEz sync table name.
func (PixezIllustHistory) TableName() string { return "illust_histories" }

// PixezNovelHistory stores a synced novel browsing history record.
type PixezNovelHistory struct {
	ID          uint   `gorm:"primaryKey;autoIncrement" json:"-"`
	PixivUserID string `gorm:"column:pixiv_user_id;not null;index" json:"-"`
	NovelID     int    `gorm:"column:novel_id;not null" json:"novel_id"`
	UserID      int    `gorm:"column:user_id;not null" json:"user_id"`
	PictureURL  string `gorm:"column:picture_url;not null" json:"picture_url"`
	Title       string `gorm:"column:title;not null" json:"title"`
	UserName    string `gorm:"column:user_name;not null" json:"user_name"`
	Time        int64  `gorm:"column:time;not null" json:"time"`
}

// TableName returns the legacy PixEz sync table name.
func (PixezNovelHistory) TableName() string { return "novel_histories" }

// PixezTagHistory stores a synced search tag history record.
type PixezTagHistory struct {
	ID             uint   `gorm:"primaryKey;autoIncrement" json:"-"`
	PixivUserID    string `gorm:"column:pixiv_user_id;not null;index" json:"-"`
	Name           string `gorm:"column:name;not null" json:"name"`
	TranslatedName string `gorm:"column:translated_name;not null" json:"translated_name"`
	Type           int    `gorm:"column:type" json:"type"`
}

// TableName returns the legacy PixEz sync table name.
func (PixezTagHistory) TableName() string { return "tag_histories" }

// PixezUserDataPayload is the client-compatible backup payload.
type PixezUserDataPayload struct {
	BanComments     *[]PixezBanComment    `json:"ban_comments,omitempty"`
	BanIllusts      *[]PixezBanIllust     `json:"ban_illusts,omitempty"`
	BanTags         *[]PixezBanTag        `json:"ban_tags,omitempty"`
	BanUsers        *[]PixezBanUser       `json:"ban_users,omitempty"`
	IllustHistories *[]PixezIllustHistory `json:"illust_histories,omitempty"`
	NovelHistories  *[]PixezNovelHistory  `json:"novel_histories,omitempty"`
	TagHistories    *[]PixezTagHistory    `json:"tag_histories,omitempty"`
}
