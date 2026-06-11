// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package pixez

import (
	"context"
	//nolint:gosec // MD5 is used only for non-cryptographic checksums of sync data
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/logger"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/util"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Ping verifies the PixEz sync API is reachable under Wavelet AccessToken auth.
// @Summary PixEz sync ping
// @Description Verifies that the PixEz sync API is reachable with Wavelet authentication.
// @Tags pixez
// @Produce json
// @Security SessionCookie
// @Success 200 {object} util.ResponseAny
// @Failure 401 {object} util.ResponseAny
// @Router /api/pixez/ping [get]
func Ping(c *gin.Context) {
	c.JSON(http.StatusOK, util.OK(gin.H{"status": "ok"}))
}

// ListUsers retrieves all saved Pixiv users as safe DTOs.
// @Summary List Pixiv users
// @Description Retrieves all saved Pixiv users without sensitive token fields.
// @Tags pixez
// @Produce json
// @Security SessionCookie
// @Success 200 {object} util.ResponseAny
// @Router /api/pixez/users [get]
func ListUsers(c *gin.Context) {
	ctx := c.Request.Context()
	var users []model.PixezPixivUser
	if err := db.DB(ctx).Order("updated_at desc").Find(&users).Error; err != nil {
		logger.ErrorF(ctx, "[PixEz] list users failed: %v", err)
		c.JSON(http.StatusOK, util.Err(errFetchUsersFailed))
		return
	}

	dtos := make([]model.PixezPixivUserSafeDTO, len(users))
	for i, u := range users {
		dtos[i] = u.ToSafeDTO()
	}
	c.JSON(http.StatusOK, util.OK(dtos))
}

// GetUser retrieves a Pixiv user's full credentials.
// @Summary Get Pixiv user credentials
// @Description Retrieves a Pixiv user's stored credentials, including access and refresh tokens.
// @Tags pixez
// @Produce json
// @Security SessionCookie
// @Param pixiv_user_id path string true "Pixiv user ID"
// @Success 200 {object} util.ResponseAny
// @Failure 404 {object} util.ResponseAny
// @Router /api/pixez/users/{pixiv_user_id} [get]
func GetUser(c *gin.Context) {
	userID, ok := pixivUserIDParam(c)
	if !ok {
		return
	}

	var user model.PixezPixivUser
	if err := db.DB(c.Request.Context()).Where("pixiv_user_id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, util.Err(errUserNotFound))
			return
		}
		logger.ErrorF(c.Request.Context(), "[PixEz] get user failed pixiv_user_id=%s: %v", userID, err)
		c.JSON(http.StatusOK, util.Err(errFetchUserFailed))
		return
	}
	c.JSON(http.StatusOK, util.OK(user))
}

// UpsertUser inserts or updates a Pixiv user's credentials.
// @Summary Upsert Pixiv user credentials
// @Description Inserts or updates a Pixiv user's stored credentials. The path Pixiv user ID is authoritative.
// @Tags pixez
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param pixiv_user_id path string true "Pixiv user ID"
// @Param payload body model.PixezPixivUser true "Pixiv user credentials"
// @Success 200 {object} util.ResponseAny
// @Router /api/pixez/users/{pixiv_user_id} [put]
func UpsertUser(c *gin.Context) {
	userID, ok := pixivUserIDParam(c)
	if !ok {
		return
	}

	var input model.PixezPixivUser
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(errInvalidRequestBody))
		return
	}
	input.PixivUserID = userID

	if err := db.DB(c.Request.Context()).Save(&input).Error; err != nil {
		logger.ErrorF(c.Request.Context(), "[PixEz] save user failed pixiv_user_id=%s: %v", userID, err)
		c.JSON(http.StatusOK, util.Err(errSaveUserFailed))
		return
	}
	c.JSON(http.StatusOK, util.OKNil())
}

// DeleteUser removes a Pixiv user and synced backup data.
// @Summary Delete Pixiv user
// @Description Deletes a Pixiv user and the migrated backup sync data associated with that user.
// @Tags pixez
// @Produce json
// @Security SessionCookie
// @Param pixiv_user_id path string true "Pixiv user ID"
// @Success 200 {object} util.ResponseAny
// @Router /api/pixez/users/{pixiv_user_id} [delete]
func DeleteUser(c *gin.Context) {
	userID, ok := pixivUserIDParam(c)
	if !ok {
		return
	}

	ctx := c.Request.Context()
	if err := db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := deleteUserSyncData(tx, userID, allSyncTables()); err != nil {
			return err
		}
		if err := tx.Where("pixiv_user_id = ?", userID).Delete(&model.PixezBookmarkIllust{}).Error; err != nil {
			return err
		}
		if err := tx.Where("pixiv_user_id = ?", userID).Delete(&model.PixezBookmarkNovel{}).Error; err != nil {
			return err
		}
		if err := tx.Where("pixiv_user_id = ?", userID).Delete(&model.PixezBookmarkExportRun{}).Error; err != nil {
			return err
		}
		return tx.Where("pixiv_user_id = ?", userID).Delete(&model.PixezPixivUser{}).Error
	}); err != nil {
		logger.ErrorF(ctx, "[PixEz] delete user failed pixiv_user_id=%s: %v", userID, err)
		c.JSON(http.StatusOK, util.Err(errDeleteUserFailed))
		return
	}
	c.JSON(http.StatusOK, util.OKNil())
}

// GetUserData fetches all or selected synced user data tables.
// @Summary Get synced user data
// @Description Fetches all or selected synced data tables for a Pixiv user.
// @Tags pixez
// @Produce json
// @Security SessionCookie
// @Param pixiv_user_id path string true "Pixiv user ID"
// @Param tables query string false "Comma-separated table names"
// @Success 200 {object} util.ResponseAny
// @Router /api/pixez/users/{pixiv_user_id}/sync-data [get]
func GetUserData(c *gin.Context) {
	userID, ok := pixivUserIDParam(c)
	if !ok {
		return
	}

	tables := requestedTables(c.Query("tables"))
	payload, err := fetchUserData(c.Request.Context(), userID, tables)
	if err != nil {
		logger.ErrorF(c.Request.Context(), "[PixEz] fetch sync data failed pixiv_user_id=%s: %v", userID, err)
		c.JSON(http.StatusOK, util.Err(errFetchSyncDataFailed))
		return
	}
	c.JSON(http.StatusOK, util.OK(payload))
}

// PostUserData replaces submitted synced user data tables.
// @Summary Replace synced user data
// @Description Replaces only the submitted synced data tables for a Pixiv user in a single transaction.
// @Tags pixez
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param pixiv_user_id path string true "Pixiv user ID"
// @Param payload body model.PixezUserDataPayload true "Synced data payload"
// @Success 200 {object} util.ResponseAny
// @Router /api/pixez/users/{pixiv_user_id}/sync-data [post]
func PostUserData(c *gin.Context) {
	userID, ok := pixivUserIDParam(c)
	if !ok {
		return
	}

	var payload model.PixezUserDataPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(errInvalidRequestBody))
		return
	}

	tables := presentTables(payload)
	if err := db.DB(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := deleteUserSyncData(tx, userID, tables); err != nil {
			return err
		}
		return createUserSyncData(tx, userID, payload)
	}); err != nil {
		logger.ErrorF(c.Request.Context(), "[PixEz] save sync data failed pixiv_user_id=%s: %v", userID, err)
		c.JSON(http.StatusOK, util.Err(errSaveSyncDataFailed))
		return
	}

	c.JSON(http.StatusOK, util.OKNil())
}

// GetUserDataHashes returns MD5 checksums for each synced table.
// @Summary Get synced data hashes
// @Description Returns MD5 checksums for each synced table of a Pixiv user.
// @Tags pixez
// @Produce json
// @Security SessionCookie
// @Param pixiv_user_id path string true "Pixiv user ID"
// @Success 200 {object} util.ResponseAny
// @Router /api/pixez/users/{pixiv_user_id}/sync-data/hashes [get]
func GetUserDataHashes(c *gin.Context) {
	userID, ok := pixivUserIDParam(c)
	if !ok {
		return
	}

	hashes, err := computeUserDataHashes(c.Request.Context(), userID)
	if err != nil {
		logger.ErrorF(c.Request.Context(), "[PixEz] fetch sync data hashes failed pixiv_user_id=%s: %v", userID, err)
		c.JSON(http.StatusOK, util.Err(errFetchHashesFailed))
		return
	}
	c.JSON(http.StatusOK, util.OK(hashes))
}

func pixivUserIDParam(c *gin.Context) (string, bool) {
	userID := strings.TrimSpace(c.Param("pixiv_user_id"))
	if userID == "" {
		c.JSON(http.StatusBadRequest, util.Err(errPixivUserIDRequired))
		return "", false
	}
	return userID, true
}

func requestedTables(raw string) map[string]bool {
	if raw == "" {
		return allSyncTables()
	}
	tables := make(map[string]bool)
	for _, item := range strings.Split(raw, ",") {
		name := strings.TrimSpace(item)
		if isKnownSyncTable(name) {
			tables[name] = true
		}
	}
	return tables
}

func allSyncTables() map[string]bool {
	return map[string]bool{
		"ban_comments":     true,
		"ban_illusts":      true,
		"ban_tags":         true,
		"ban_users":        true,
		"illust_histories": true,
		"novel_histories":  true,
		"tag_histories":    true,
	}
}

func isKnownSyncTable(name string) bool {
	return allSyncTables()[name]
}

func presentTables(payload model.PixezUserDataPayload) map[string]bool {
	tables := make(map[string]bool)
	if payload.BanComments != nil {
		tables["ban_comments"] = true
	}
	if payload.BanIllusts != nil {
		tables["ban_illusts"] = true
	}
	if payload.BanTags != nil {
		tables["ban_tags"] = true
	}
	if payload.BanUsers != nil {
		tables["ban_users"] = true
	}
	if payload.IllustHistories != nil {
		tables["illust_histories"] = true
	}
	if payload.NovelHistories != nil {
		tables["novel_histories"] = true
	}
	if payload.TagHistories != nil {
		tables["tag_histories"] = true
	}
	return tables
}

func fetchUserData(ctx context.Context, userID string, tables map[string]bool) (model.PixezUserDataPayload, error) {
	tx := db.DB(ctx)
	var payload model.PixezUserDataPayload

	if tables["ban_comments"] {
		var rows []model.PixezBanComment
		if err := tx.Where("pixiv_user_id = ?", userID).Find(&rows).Error; err != nil {
			return payload, err
		}
		payload.BanComments = &rows
	}
	if tables["ban_illusts"] {
		var rows []model.PixezBanIllust
		if err := tx.Where("pixiv_user_id = ?", userID).Find(&rows).Error; err != nil {
			return payload, err
		}
		payload.BanIllusts = &rows
	}
	if tables["ban_tags"] {
		var rows []model.PixezBanTag
		if err := tx.Where("pixiv_user_id = ?", userID).Find(&rows).Error; err != nil {
			return payload, err
		}
		payload.BanTags = &rows
	}
	if tables["ban_users"] {
		var rows []model.PixezBanUser
		if err := tx.Where("pixiv_user_id = ?", userID).Find(&rows).Error; err != nil {
			return payload, err
		}
		payload.BanUsers = &rows
	}
	if tables["illust_histories"] {
		var rows []model.PixezIllustHistory
		if err := tx.Where("pixiv_user_id = ?", userID).Find(&rows).Error; err != nil {
			return payload, err
		}
		payload.IllustHistories = &rows
	}
	if tables["novel_histories"] {
		var rows []model.PixezNovelHistory
		if err := tx.Where("pixiv_user_id = ?", userID).Find(&rows).Error; err != nil {
			return payload, err
		}
		payload.NovelHistories = &rows
	}
	if tables["tag_histories"] {
		var rows []model.PixezTagHistory
		if err := tx.Where("pixiv_user_id = ?", userID).Find(&rows).Error; err != nil {
			return payload, err
		}
		payload.TagHistories = &rows
	}

	return payload, nil
}

func deleteUserSyncData(tx *gorm.DB, userID string, tables map[string]bool) error {
	if tables["ban_comments"] {
		if err := tx.Where("pixiv_user_id = ?", userID).Delete(&model.PixezBanComment{}).Error; err != nil {
			return err
		}
	}
	if tables["ban_illusts"] {
		if err := tx.Where("pixiv_user_id = ?", userID).Delete(&model.PixezBanIllust{}).Error; err != nil {
			return err
		}
	}
	if tables["ban_tags"] {
		if err := tx.Where("pixiv_user_id = ?", userID).Delete(&model.PixezBanTag{}).Error; err != nil {
			return err
		}
	}
	if tables["ban_users"] {
		if err := tx.Where("pixiv_user_id = ?", userID).Delete(&model.PixezBanUser{}).Error; err != nil {
			return err
		}
	}
	if tables["illust_histories"] {
		if err := tx.Where("pixiv_user_id = ?", userID).Delete(&model.PixezIllustHistory{}).Error; err != nil {
			return err
		}
	}
	if tables["novel_histories"] {
		if err := tx.Where("pixiv_user_id = ?", userID).Delete(&model.PixezNovelHistory{}).Error; err != nil {
			return err
		}
	}
	if tables["tag_histories"] {
		if err := tx.Where("pixiv_user_id = ?", userID).Delete(&model.PixezTagHistory{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func createUserSyncData(tx *gorm.DB, userID string, payload model.PixezUserDataPayload) error {
	creators := []func() error{
		func() error { return createBanComments(tx, userID, payload.BanComments) },
		func() error { return createBanIllusts(tx, userID, payload.BanIllusts) },
		func() error { return createBanTags(tx, userID, payload.BanTags) },
		func() error { return createBanUsers(tx, userID, payload.BanUsers) },
		func() error { return createIllustHistories(tx, userID, payload.IllustHistories) },
		func() error { return createNovelHistories(tx, userID, payload.NovelHistories) },
		func() error { return createTagHistories(tx, userID, payload.TagHistories) },
	}
	for _, create := range creators {
		if err := create(); err != nil {
			return err
		}
	}
	return nil
}

func createBanComments(tx *gorm.DB, userID string, input *[]model.PixezBanComment) error {
	if input == nil {
		return nil
	}
	rows := *input
	for i := range rows {
		rows[i].ID = 0
		rows[i].PixivUserID = userID
	}
	return createRows(tx, rows)
}

func createBanIllusts(tx *gorm.DB, userID string, input *[]model.PixezBanIllust) error {
	if input == nil {
		return nil
	}
	rows := *input
	for i := range rows {
		rows[i].ID = 0
		rows[i].PixivUserID = userID
	}
	return createRows(tx, rows)
}

func createBanTags(tx *gorm.DB, userID string, input *[]model.PixezBanTag) error {
	if input == nil {
		return nil
	}
	rows := *input
	for i := range rows {
		rows[i].ID = 0
		rows[i].PixivUserID = userID
	}
	return createRows(tx, rows)
}

func createBanUsers(tx *gorm.DB, userID string, input *[]model.PixezBanUser) error {
	if input == nil {
		return nil
	}
	rows := *input
	for i := range rows {
		rows[i].ID = 0
		rows[i].PixivUserID = userID
	}
	return createRows(tx, rows)
}

func createIllustHistories(tx *gorm.DB, userID string, input *[]model.PixezIllustHistory) error {
	if input == nil {
		return nil
	}
	rows := *input
	for i := range rows {
		rows[i].ID = 0
		rows[i].PixivUserID = userID
	}
	return createRows(tx, rows)
}

func createNovelHistories(tx *gorm.DB, userID string, input *[]model.PixezNovelHistory) error {
	if input == nil {
		return nil
	}
	rows := *input
	for i := range rows {
		rows[i].ID = 0
		rows[i].PixivUserID = userID
	}
	return createRows(tx, rows)
}

func createTagHistories(tx *gorm.DB, userID string, input *[]model.PixezTagHistory) error {
	if input == nil {
		return nil
	}
	rows := *input
	for i := range rows {
		rows[i].ID = 0
		rows[i].PixivUserID = userID
	}
	return createRows(tx, rows)
}

func createRows[T any](tx *gorm.DB, rows []T) error {
	if len(rows) == 0 {
		return nil
	}
	return tx.Create(&rows).Error
}

func computeUserDataHashes(ctx context.Context, userID string) (map[string]string, error) {
	hashes := make(map[string]string)
	tx := db.DB(ctx)

	var banComments []model.PixezBanComment
	if err := tx.Where("pixiv_user_id = ?", userID).Find(&banComments).Error; err != nil {
		return nil, err
	}
	lines := make([]string, len(banComments))
	for i, r := range banComments {
		lines[i] = fmt.Sprintf("%s:%s", r.CommentID, r.Name)
	}
	hashes["ban_comments"] = computeHash(lines)

	var banIllusts []model.PixezBanIllust
	if err := tx.Where("pixiv_user_id = ?", userID).Find(&banIllusts).Error; err != nil {
		return nil, err
	}
	lines = make([]string, len(banIllusts))
	for i, r := range banIllusts {
		lines[i] = fmt.Sprintf("%s:%s", r.IllustID, r.Name)
	}
	hashes["ban_illusts"] = computeHash(lines)

	var banTags []model.PixezBanTag
	if err := tx.Where("pixiv_user_id = ?", userID).Find(&banTags).Error; err != nil {
		return nil, err
	}
	lines = make([]string, len(banTags))
	for i, r := range banTags {
		lines[i] = fmt.Sprintf("%s:%s", r.Name, r.TranslateName)
	}
	hashes["ban_tags"] = computeHash(lines)

	var banUsers []model.PixezBanUser
	if err := tx.Where("pixiv_user_id = ?", userID).Find(&banUsers).Error; err != nil {
		return nil, err
	}
	lines = make([]string, len(banUsers))
	for i, r := range banUsers {
		lines[i] = fmt.Sprintf("%s:%s", r.UserID, r.Name)
	}
	hashes["ban_users"] = computeHash(lines)

	var illustHistories []model.PixezIllustHistory
	if err := tx.Where("pixiv_user_id = ?", userID).Find(&illustHistories).Error; err != nil {
		return nil, err
	}
	lines = make([]string, len(illustHistories))
	for i, r := range illustHistories {
		lines[i] = fmt.Sprintf("%d:%d:%d", r.IllustID, r.UserID, r.Time)
	}
	hashes["illust_histories"] = computeHash(lines)

	var novelHistories []model.PixezNovelHistory
	if err := tx.Where("pixiv_user_id = ?", userID).Find(&novelHistories).Error; err != nil {
		return nil, err
	}
	lines = make([]string, len(novelHistories))
	for i, r := range novelHistories {
		lines[i] = fmt.Sprintf("%d:%d:%d", r.NovelID, r.UserID, r.Time)
	}
	hashes["novel_histories"] = computeHash(lines)

	var tagHistories []model.PixezTagHistory
	if err := tx.Where("pixiv_user_id = ?", userID).Find(&tagHistories).Error; err != nil {
		return nil, err
	}
	lines = make([]string, len(tagHistories))
	for i, r := range tagHistories {
		lines[i] = fmt.Sprintf("%s:%s:%d", r.Name, r.TranslatedName, r.Type)
	}
	hashes["tag_histories"] = computeHash(lines)

	return hashes, nil
}

func computeHash(lines []string) string {
	if len(lines) == 0 {
		return "empty"
	}
	sort.Strings(lines)
	//nolint:gosec // MD5 is used only for non-cryptographic checksums of sync data
	sum := md5.Sum([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:])
}
