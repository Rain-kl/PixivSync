// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package pixez

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	pixezsvc "github.com/Rain-kl/Wavelet/internal/service/pixez"
	"github.com/Rain-kl/Wavelet/internal/common/response"
	"github.com/gin-gonic/gin"
)

// ListRemovedBookmarkIllusts returns exported illustrations that disappeared from Pixiv bookmarks.
func ListRemovedBookmarkIllusts(c *gin.Context) {
	userID, ok := pixivUserIDParam(c)
	if !ok {
		return
	}
	offset := parseNonNegativeInt(c.Query("offset"), 0)
	limit := parseNonNegativeInt(c.Query("limit"), defaultRemovedBookmarkLimit)
	if limit <= 0 {
		limit = defaultRemovedBookmarkLimit
	}
	if limit > maxRemovedBookmarkLimit {
		limit = maxRemovedBookmarkLimit
	}

	query := db.DB(c.Request.Context()).
		Where("pixiv_user_id = ? AND removed = ?", userID, true).
		Order("removed_at desc, updated_at desc, id desc")
	if restrict := c.Query("restrict"); restrict == "public" || restrict == "private" {
		query = query.Where("restrict = ?", restrict)
	}

	var total int64
	if err := query.Model(&model.PixezBookmarkIllust{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusOK, response.Err(errFetchRemovedBookmarksFailed))
		return
	}
	var records []model.PixezBookmarkIllust
	if err := query.Limit(limit).Offset(offset).Find(&records).Error; err != nil {
		c.JSON(http.StatusOK, response.Err(errFetchRemovedBookmarksFailed))
		return
	}

	nextURL := ""
	nextOffset := offset + limit
	if int64(nextOffset) < total {
		nextURL = removedBookmarkNextURL(userID, c.Query("restrict"), nextOffset, limit)
	}
	c.JSON(http.StatusOK, response.OK(gin.H{
		"illusts":  pixezsvc.RemovedBookmarkIllustPayload(records),
		"next_url": nextURL,
	}))
}

func parseNonNegativeInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func removedBookmarkNextURL(userID string, restrict string, offset int, limit int) string {
	values := url.Values{}
	values.Set("offset", strconv.Itoa(offset))
	values.Set("limit", strconv.Itoa(limit))
	if restrict == "public" || restrict == "private" {
		values.Set("restrict", restrict)
	}
	return "/api/pixez/users/" + url.PathEscape(userID) + "/bookmarks/illust/removed?" + values.Encode()
}
