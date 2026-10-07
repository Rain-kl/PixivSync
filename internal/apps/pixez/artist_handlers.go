// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package pixez

import (
	"errors"
	"net/http"

	"github.com/Rain-kl/Wavelet/internal/common/response"
	"github.com/Rain-kl/Wavelet/internal/model"
	pixezsvc "github.com/Rain-kl/Wavelet/internal/service/pixez"
	"github.com/Rain-kl/Wavelet/pkg/logger"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	artistDefaultPageSize = 24
	artistMaxPageSize     = 100
	errArtistOperation    = "作者订阅操作失败，请稍后重试"
)

// ListArtists lists shared creators discovered from archived works.
// @Summary 作者订阅目录
// @Tags pixez
// @Produce json
// @Security SessionCookie
// @Param type query string false "illust or novel"
// @Param q query string false "作者名称或 ID"
// @Param subscription query string false "enabled, disabled, error"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} response.Any
// @Failure 400 {object} response.Any
// @Failure 401 {object} response.Any
// @Failure 500 {object} response.Any
// @Router /api/pixez/artists [get]
func ListArtists(c *gin.Context) {
	req, ok := artistQuery(c)
	if !ok {
		return
	}
	if err := pixezsvc.EnsureArtistDirectory(c.Request.Context()); artistOperationError(c, err) {
		return
	}
	items, total, err := pixezsvc.ListArtists(c.Request.Context(), req)
	if artistOperationError(c, err) {
		return
	}
	c.JSON(http.StatusOK, response.OK(gin.H{"total": total, "results": items}))
}

// GetArtist returns a creator and both independent subscriptions.
// @Summary 作者订阅详情
// @Tags pixez
// @Produce json
// @Security SessionCookie
// @Param artist_id path string true "Pixiv 作者 ID"
// @Success 200 {object} response.Any
// @Failure 400 {object} response.Any
// @Failure 401 {object} response.Any
// @Failure 404 {object} response.Any
// @Failure 500 {object} response.Any
// @Router /api/pixez/artists/{artist_id} [get]
func GetArtist(c *gin.Context) {
	id, ok := parsePositiveIDParam(c, "artist_id")
	if !ok {
		return
	}
	artist, err := pixezsvc.GetArtist(c.Request.Context(), id)
	if artistOperationError(c, err) {
		return
	}
	subs, err := pixezsvc.ArtistSubscriptions(c.Request.Context(), id)
	if artistOperationError(c, err) {
		return
	}
	c.JSON(http.StatusOK, response.OK(gin.H{"artist": artist, "subscriptions": subs}))
}

// ListArtistWorks returns online-discovered and locally archived works.
// @Summary 作者作品目录
// @Tags pixez
// @Produce json
// @Security SessionCookie
// @Param artist_id path string true "作者 ID"
// @Param type query string false "illust or novel"
// @Param status query string false "complete, none, failed"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Success 200 {object} response.Any
// @Failure 400 {object} response.Any
// @Failure 401 {object} response.Any
// @Failure 404 {object} response.Any
// @Failure 500 {object} response.Any
// @Router /api/pixez/artists/{artist_id}/works [get]
func ListArtistWorks(c *gin.Context) {
	id, ok := parsePositiveIDParam(c, "artist_id")
	if !ok {
		return
	}
	req, ok := artistQuery(c)
	if !ok {
		return
	}
	if _, err := pixezsvc.GetArtist(c.Request.Context(), id); artistOperationError(c, err) {
		return
	}
	items, total, err := pixezsvc.ListArtistWorks(c.Request.Context(), id, req)
	if artistOperationError(c, err) {
		return
	}
	c.JSON(http.StatusOK, response.OK(gin.H{"total": total, "results": items}))
}

// RefreshArtistDirectory queues a directory-only scan without downloading content.
// @Summary 刷新作者作品目录
// @Tags pixez
// @Produce json
// @Security SessionCookie
// @Param artist_id path string true "作者 ID"
// @Param type query string true "illust or novel"
// @Success 200 {object} response.Any
// @Failure 400 {object} response.Any
// @Failure 401 {object} response.Any
// @Failure 404 {object} response.Any
// @Failure 500 {object} response.Any
// @Router /api/pixez/artists/{artist_id}/refresh [post]
func RefreshArtistDirectory(c *gin.Context) {
	id, ok := parsePositiveIDParam(c, "artist_id")
	if !ok {
		return
	}
	req, ok := artistQuery(c)
	if !ok {
		return
	}
	run, err := dispatchArtistScan(c.Request.Context(), id, req.TargetType, false, "refresh")
	if artistOperationError(c, err) {
		return
	}
	c.JSON(http.StatusOK, response.OK(run))
}

type artistSubscriptionRequest struct {
	Enabled *bool `json:"enabled" binding:"required"`
}

// UpdateArtistSubscription independently enables or cancels one work category.
// @Summary 开启或取消作者订阅
// @Tags pixez
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param artist_id path string true "作者 ID"
// @Param type path string true "illust or novel"
// @Param payload body artistSubscriptionRequest true "订阅状态"
// @Success 200 {object} response.Any
// @Failure 400 {object} response.Any
// @Failure 401 {object} response.Any
// @Failure 404 {object} response.Any
// @Failure 500 {object} response.Any
// @Router /api/pixez/artists/{artist_id}/subscriptions/{type} [put]
func UpdateArtistSubscription(c *gin.Context) {
	id, target, ok := artistSubscriptionParams(c)
	if !ok {
		return
	}
	var req artistSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortBadRequest(c, "必须指定 enabled")
		return
	}
	if _, err := pixezsvc.GetArtist(c.Request.Context(), id); artistOperationError(c, err) {
		return
	}
	newlyEnabled, err := pixezsvc.SetArtistSubscription(c.Request.Context(), id, target, *req.Enabled)
	if artistOperationError(c, err) {
		return
	}
	if newlyEnabled {
		if _, err = dispatchArtistScan(c.Request.Context(), id, target, true, "initial"); err != nil {
			logger.ErrorF(c.Request.Context(), "[PixEz] subscription saved but initial dispatch failed: %v", err)
			response.AbortInternal(c, "订阅已保存，首次同步下发失败，可稍后点击立即同步")
			return
		}
	}
	subs, err := pixezsvc.ArtistSubscriptions(c.Request.Context(), id)
	if artistOperationError(c, err) {
		return
	}
	c.JSON(http.StatusOK, response.OK(subs))
}

// SyncArtistSubscription manually checks an enabled subscription.
// @Summary 立即同步作者订阅
// @Tags pixez
// @Produce json
// @Security SessionCookie
// @Param artist_id path string true "作者 ID"
// @Param type path string true "illust or novel"
// @Success 200 {object} response.Any
// @Failure 400 {object} response.Any
// @Failure 401 {object} response.Any
// @Failure 404 {object} response.Any
// @Failure 500 {object} response.Any
// @Router /api/pixez/artists/{artist_id}/subscriptions/{type}/sync [post]
func SyncArtistSubscription(c *gin.Context) {
	id, target, ok := artistSubscriptionParams(c)
	if !ok {
		return
	}
	subs, err := pixezsvc.ArtistSubscriptions(c.Request.Context(), id)
	if artistOperationError(c, err) {
		return
	}
	enabled := false
	for _, sub := range subs {
		if sub.TargetType == target && sub.Enabled {
			enabled = true
		}
	}
	if !enabled {
		response.AbortBadRequest(c, "请先开启该类型订阅")
		return
	}
	run, err := dispatchArtistScan(c.Request.Context(), id, target, true, "manual")
	if artistOperationError(c, err) {
		return
	}
	c.JSON(http.StatusOK, response.OK(run))
}

// ListArtistSyncRuns returns scan history, which is independent of download completion.
// @Summary 作者扫描记录
// @Tags pixez
// @Produce json
// @Security SessionCookie
// @Param artist_id path string true "作者 ID"
// @Success 200 {object} response.Any
// @Failure 400 {object} response.Any
// @Failure 401 {object} response.Any
// @Failure 500 {object} response.Any
// @Router /api/pixez/artists/{artist_id}/sync-runs [get]
func ListArtistSyncRuns(c *gin.Context) {
	id, ok := parsePositiveIDParam(c, "artist_id")
	if !ok {
		return
	}
	runs, err := pixezsvc.ListArtistRuns(c.Request.Context(), id)
	if artistOperationError(c, err) {
		return
	}
	c.JSON(http.StatusOK, response.OK(runs))
}

func artistQuery(c *gin.Context) (pixezsvc.ArtistQuery, bool) {
	var req pixezsvc.ArtistQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		response.AbortBadRequest(c, "查询参数无效")
		return req, false
	}
	if req.TargetType == "" {
		req.TargetType = model.PixezMirrorTargetNovel
	}
	if !validArtistType(req.TargetType) {
		response.AbortBadRequest(c, "作品类型必须为 illust 或 novel")
		return req, false
	}
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = artistDefaultPageSize
	}
	if req.PageSize > artistMaxPageSize {
		req.PageSize = artistMaxPageSize
	}
	return req, true
}
func artistSubscriptionParams(c *gin.Context) (int64, string, bool) {
	id, ok := parsePositiveIDParam(c, "artist_id")
	if !ok {
		return 0, "", false
	}
	target := c.Param("type")
	if !validArtistType(target) {
		response.AbortBadRequest(c, "作品类型必须为 illust 或 novel")
		return 0, "", false
	}
	return id, target, true
}
func validArtistType(target string) bool {
	return target == model.PixezMirrorTargetIllust || target == model.PixezMirrorTargetNovel
}
func artistOperationError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.AbortNotFound(c, "作者或扫描记录不存在")
		return true
	}
	logger.ErrorF(c.Request.Context(), "[PixEz] creator operation: %v", err)
	response.AbortInternal(c, errArtistOperation)
	return true
}
