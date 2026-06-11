// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package pixez

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/logger"
	"github.com/Rain-kl/Wavelet/internal/model"
	pixezsvc "github.com/Rain-kl/Wavelet/internal/service/pixez"
	"github.com/Rain-kl/Wavelet/internal/util"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type pixezDashboardResponse struct {
	Accounts   int64               `json:"accounts"`
	Illusts    pixezMirrorProgress `json:"illusts"`
	Novels     pixezMirrorProgress `json:"novels"`
	Queue      pixezQueueStats     `json:"queue"`
	RecentRuns []pixezExportRunDTO `json:"recent_runs"`
	UpdatedAt  time.Time           `json:"updated_at"`
}

type pixezMirrorProgress struct {
	Total      int64   `json:"total"`
	Succeeded  int64   `json:"succeeded"`
	Processing int64   `json:"processing"`
	Failed     int64   `json:"failed"`
	NotQueued  int64   `json:"not_queued"`
	Percent    float64 `json:"percent"`
}

type pixezQueueStats struct {
	Running int64 `json:"running"`
	Queued  int64 `json:"queued"`
}

type pixezExportRunDTO struct {
	ID             string     `json:"id"`
	TargetType     string     `json:"target_type"`
	PixivUserID    string     `json:"pixiv_user_id"`
	Status         string     `json:"status"`
	TotalCount     int        `json:"total_count"`
	NewCount       int        `json:"new_count"`
	UpdatedCount   int        `json:"updated_count"`
	RemovedCount   int        `json:"removed_count"`
	ErrorMessage   string     `json:"error_message"`
	StartedAt      time.Time  `json:"started_at"`
	FinishedAt     *time.Time `json:"finished_at"`
	DurationMS     int64      `json:"duration_ms"`
	LastRequestURL string     `json:"last_request_url"`
}

type listPixezBookmarksRequest struct {
	Query        string `form:"q"`
	PixivUserID  string `form:"pixiv_user_id"`
	MirrorStatus string `form:"mirror_status"`
	WorkStatus   string `form:"work_status"`
}

type pixezIllustBookmarkDTO struct {
	ID               uint       `json:"id"`
	PixivUserID      string     `json:"pixiv_user_id"`
	Restrict         string     `json:"restrict"`
	IllustID         int64      `json:"illust_id"`
	Title            string     `json:"title"`
	Type             string     `json:"type"`
	UserID           int64      `json:"user_id"`
	UserName         string     `json:"user_name"`
	CoverURL         string     `json:"cover_url"`
	PageCount        int        `json:"page_count"`
	Width            int        `json:"width"`
	Height           int        `json:"height"`
	SanityLevel      int        `json:"sanity_level"`
	XRestrict        int        `json:"x_restrict"`
	TotalBookmarks   int        `json:"total_bookmarks"`
	Visible          bool       `json:"visible"`
	IsMuted          bool       `json:"is_muted"`
	MirrorStatus     int        `json:"mirror_status"`
	MirrorStatusText string     `json:"mirror_status_text"`
	MirrorRetryCount int        `json:"mirror_retry_count"`
	Removed          bool       `json:"removed"`
	RemovedAt        *time.Time `json:"removed_at"`
	LastSeenAt       time.Time  `json:"last_seen_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type pixezNovelBookmarkDTO struct {
	ID               uint       `json:"id"`
	PixivUserID      string     `json:"pixiv_user_id"`
	Restrict         string     `json:"restrict"`
	NovelID          int64      `json:"novel_id"`
	Title            string     `json:"title"`
	Caption          string     `json:"caption"`
	UserID           int64      `json:"user_id"`
	UserName         string     `json:"user_name"`
	CoverURL         string     `json:"cover_url"`
	TextLength       int        `json:"text_length"`
	XRestrict        int        `json:"x_restrict"`
	TotalBookmarks   int        `json:"total_bookmarks"`
	IsOriginal       bool       `json:"is_original"`
	Visible          bool       `json:"visible"`
	IsMuted          bool       `json:"is_muted"`
	SeriesID         *int64     `json:"series_id"`
	SeriesTitle      *string    `json:"series_title"`
	MirrorStatus     int        `json:"mirror_status"`
	MirrorStatusText string     `json:"mirror_status_text"`
	MirrorRetryCount int        `json:"mirror_retry_count"`
	Removed          bool       `json:"removed"`
	RemovedAt        *time.Time `json:"removed_at"`
	LastSeenAt       time.Time  `json:"last_seen_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type pixezIllustBookmarkDetailDTO struct {
	Item        pixezIllustBookmarkDTO       `json:"item"`
	Mirror      *pixezMirrorDetailDTO        `json:"mirror"`
	ImageFiles  []model.PixezMirrorImageFile `json:"image_files"`
	RequestURLs []string                     `json:"request_urls"`
	RetryURLs   []string                     `json:"retry_urls"`
	IllustJSON  json.RawMessage              `json:"illust_json"`
}

type pixezNovelBookmarkDetailDTO struct {
	Item        pixezNovelBookmarkDTO `json:"item"`
	Mirror      *pixezMirrorDetailDTO `json:"mirror"`
	RequestURLs []string              `json:"request_urls"`
	RetryURLs   []string              `json:"retry_urls"`
	NovelJSON   json.RawMessage       `json:"novel_json"`
}

type pixezMirrorDetailDTO struct {
	TaskID       string    `json:"task_id"`
	Status       string    `json:"status"`
	TotalCount   int       `json:"total_count"`
	SuccessCount int       `json:"success_count"`
	FailedCount  int       `json:"failed_count"`
	ErrorMessage string    `json:"error_message"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// GetDashboard returns PixEz management dashboard metrics.
// @Summary PixEz management dashboard
// @Description Returns account count, bookmark mirror progress, queue status, and recent bookmark export runs.
// @Tags pixez
// @Produce json
// @Security SessionCookie
// @Success 200 {object} util.ResponseAny{data=object}
// @Router /api/pixez/dashboard [get]
func GetDashboard(c *gin.Context) {
	payload, err := buildDashboard(c.Request.Context())
	if err != nil {
		logger.ErrorF(c.Request.Context(), "[PixEz] fetch dashboard failed: %v", err)
		c.JSON(http.StatusOK, util.Err(errFetchDashboardFailed))
		return
	}
	c.JSON(http.StatusOK, util.OK(payload))
}

// RefreshUserToken refreshes stored Pixiv credentials for one account.
// @Summary Refresh PixEz Pixiv token
// @Description Refreshes the stored Pixiv access token for a PixEz account.
// @Tags pixez
// @Produce json
// @Security SessionCookie
// @Param pixiv_user_id path string true "Pixiv user ID"
// @Success 200 {object} util.ResponseAny
// @Router /api/pixez/users/{pixiv_user_id}/refresh-token [post]
func RefreshUserToken(c *gin.Context) {
	userID, ok := pixivUserIDParam(c)
	if !ok {
		return
	}

	var user model.PixezPixivUser
	if err := db.DB(c.Request.Context()).Where("pixiv_user_id = ?", userID).First(&user).Error; err != nil {
		logger.ErrorF(c.Request.Context(), "[PixEz] fetch user for token refresh failed pixiv_user_id=%s: %v", userID, err)
		c.JSON(http.StatusOK, util.Err(errFetchUserFailed))
		return
	}
	if _, err := pixezsvc.DefaultClient.RefreshPixivToken(c.Request.Context(), userID, user.RefreshToken); err != nil {
		logger.ErrorF(c.Request.Context(), "[PixEz] refresh token failed pixiv_user_id=%s: %v", userID, err)
		c.JSON(http.StatusOK, util.Err(errRefreshPixivAccountFailed))
		return
	}

	c.JSON(http.StatusOK, util.OKNil())
}

// ListBookmarkExportRuns returns recent PixEz bookmark export batches.
// @Summary List PixEz bookmark export runs
// @Description Returns paginated bookmark export run records for PixEz management.
// @Tags pixez
// @Produce json
// @Security SessionCookie
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} util.ResponseAny{data=object}
// @Router /api/pixez/bookmark-export-runs [get]
func ListBookmarkExportRuns(c *gin.Context) {
	page, pageSize := parsePage(c)
	if pageSize > maxExportRunPageSize {
		pageSize = maxExportRunPageSize
	}
	items, total, err := listBookmarkExportRuns(c.Request.Context(), page, pageSize)
	if err != nil {
		logger.ErrorF(c.Request.Context(), "[PixEz] list bookmark export runs failed: %v", err)
		c.JSON(http.StatusOK, util.Err(errFetchExportRunsFailed))
		return
	}
	c.JSON(http.StatusOK, util.OK(gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}))
}

// ListBookmarkIllusts returns bookmarked illustrations with mirror state.
// @Summary List PixEz bookmark illustrations
// @Description Returns paginated Pixiv bookmark illustration records with mirror state and cover URLs.
// @Tags pixez
// @Produce json
// @Security SessionCookie
// @Param q query string false "Search by ID, title, artist, or Pixiv account"
// @Param pixiv_user_id query string false "Pixiv account filter"
// @Param mirror_status query string false "Mirror status: success, processing, failed, none"
// @Param work_status query string false "Work status: visible, muted, unavailable, removed, all"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(24)
// @Success 200 {object} util.ResponseAny{data=object}
// @Router /api/pixez/bookmarks/illusts [get]
func ListBookmarkIllusts(c *gin.Context) {
	req := bindBookmarkListRequest(c)
	page, pageSize := parseManagementPage(c)
	items, total, err := listBookmarkIllusts(c.Request.Context(), req, page, pageSize)
	if err != nil {
		logger.ErrorF(c.Request.Context(), "[PixEz] list bookmark illusts failed: %v", err)
		c.JSON(http.StatusOK, util.Err(errFetchBookmarksFailed))
		return
	}
	c.JSON(http.StatusOK, util.OK(gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}))
}

// ListBookmarkNovels returns bookmarked novels with mirror state.
// @Summary List PixEz bookmark novels
// @Description Returns paginated Pixiv bookmark novel records with mirror state and cover URLs.
// @Tags pixez
// @Produce json
// @Security SessionCookie
// @Param q query string false "Search by ID, title, author, or Pixiv account"
// @Param pixiv_user_id query string false "Pixiv account filter"
// @Param mirror_status query string false "Mirror status: success, processing, failed, none"
// @Param work_status query string false "Work status: visible, muted, unavailable, removed, all"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(24)
// @Success 200 {object} util.ResponseAny{data=object}
// @Router /api/pixez/bookmarks/novels [get]
func ListBookmarkNovels(c *gin.Context) {
	req := bindBookmarkListRequest(c)
	page, pageSize := parseManagementPage(c)
	items, total, err := listBookmarkNovels(c.Request.Context(), req, page, pageSize)
	if err != nil {
		logger.ErrorF(c.Request.Context(), "[PixEz] list bookmark novels failed: %v", err)
		c.JSON(http.StatusOK, util.Err(errFetchBookmarksFailed))
		return
	}
	c.JSON(http.StatusOK, util.OK(gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}))
}

// GetBookmarkIllustDetail returns one illustration bookmark and mirror diagnostics.
// @Summary Get PixEz bookmark illustration detail
// @Description Returns one bookmarked illustration with mirror read-model diagnostics.
// @Tags pixez
// @Produce json
// @Security SessionCookie
// @Param illust_id path int true "Pixiv illustration ID"
// @Success 200 {object} util.ResponseAny{data=object}
// @Router /api/pixez/bookmarks/illusts/{illust_id}/detail [get]
func GetBookmarkIllustDetail(c *gin.Context) {
	illustID, ok := parsePositiveIDParam(c, "illust_id")
	if !ok {
		return
	}
	detail, err := getBookmarkIllustDetail(c.Request.Context(), illustID)
	if err != nil {
		logger.ErrorF(c.Request.Context(), "[PixEz] get bookmark illust detail failed illust_id=%d: %v", illustID, err)
		c.JSON(http.StatusOK, util.Err(errFetchBookmarkDetailFailed))
		return
	}
	c.JSON(http.StatusOK, util.OK(detail))
}

// GetBookmarkNovelDetail returns one novel bookmark and mirror diagnostics.
// @Summary Get PixEz bookmark novel detail
// @Description Returns one bookmarked novel with mirror read-model diagnostics.
// @Tags pixez
// @Produce json
// @Security SessionCookie
// @Param novel_id path int true "Pixiv novel ID"
// @Success 200 {object} util.ResponseAny{data=object}
// @Router /api/pixez/bookmarks/novels/{novel_id}/detail [get]
func GetBookmarkNovelDetail(c *gin.Context) {
	novelID, ok := parsePositiveIDParam(c, "novel_id")
	if !ok {
		return
	}
	detail, err := getBookmarkNovelDetail(c.Request.Context(), novelID)
	if err != nil {
		logger.ErrorF(c.Request.Context(), "[PixEz] get bookmark novel detail failed novel_id=%d: %v", novelID, err)
		c.JSON(http.StatusOK, util.Err(errFetchBookmarkDetailFailed))
		return
	}
	c.JSON(http.StatusOK, util.OK(detail))
}

func buildDashboard(ctx context.Context) (pixezDashboardResponse, error) {
	var accounts int64
	if err := db.DB(ctx).Model(&model.PixezPixivUser{}).Count(&accounts).Error; err != nil {
		return pixezDashboardResponse{}, fmt.Errorf("count Pixiv users: %w", err)
	}
	illusts, err := bookmarkMirrorProgress(ctx, &model.PixezBookmarkIllust{})
	if err != nil {
		return pixezDashboardResponse{}, fmt.Errorf("count illust progress: %w", err)
	}
	novels, err := bookmarkMirrorProgress(ctx, &model.PixezBookmarkNovel{})
	if err != nil {
		return pixezDashboardResponse{}, fmt.Errorf("count novel progress: %w", err)
	}
	queue, err := pixezQueueStatsForTasks(ctx)
	if err != nil {
		return pixezDashboardResponse{}, fmt.Errorf("count PixEz task queue: %w", err)
	}
	runs, _, err := listBookmarkExportRuns(ctx, 1, dashboardRecentRunLimit)
	if err != nil {
		return pixezDashboardResponse{}, fmt.Errorf("list recent export runs: %w", err)
	}
	return pixezDashboardResponse{
		Accounts:   accounts,
		Illusts:    illusts,
		Novels:     novels,
		Queue:      queue,
		RecentRuns: runs,
		UpdatedAt:  time.Now(),
	}, nil
}

func bookmarkMirrorProgress(ctx context.Context, modelObj any) (pixezMirrorProgress, error) {
	tx := db.DB(ctx).Model(modelObj).Where("removed = ?", false)
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return pixezMirrorProgress{}, err
	}
	countStatus := func(status int) (int64, error) {
		var count int64
		err := db.DB(ctx).Model(modelObj).
			Where("removed = ? AND mirror_status = ?", false, status).
			Count(&count).Error
		return count, err
	}
	succeeded, err := countStatus(model.PixezBookmarkMirrorDone)
	if err != nil {
		return pixezMirrorProgress{}, err
	}
	processing, err := countStatus(model.PixezBookmarkMirrorProcessing)
	if err != nil {
		return pixezMirrorProgress{}, err
	}
	failed, err := countStatus(model.PixezBookmarkMirrorFailed)
	if err != nil {
		return pixezMirrorProgress{}, err
	}
	notQueued, err := countStatus(model.PixezBookmarkMirrorNone)
	if err != nil {
		return pixezMirrorProgress{}, err
	}
	return pixezMirrorProgress{
		Total:      total,
		Succeeded:  succeeded,
		Processing: processing,
		Failed:     failed,
		NotQueued:  notQueued,
		Percent:    percent(succeeded, total),
	}, nil
}

func pixezQueueStatsForTasks(ctx context.Context) (pixezQueueStats, error) {
	taskTypes := pixezTaskTypes()
	countStatus := func(status model.TaskExecutionStatus) (int64, error) {
		var count int64
		err := db.DB(ctx).Model(&model.TaskExecution{}).
			Where("task_type IN ? AND status = ?", taskTypes, status).
			Count(&count).Error
		return count, err
	}
	running, err := countStatus(model.TaskExecutionStatusRunning)
	if err != nil {
		return pixezQueueStats{}, err
	}
	queued, err := countStatus(model.TaskExecutionStatusPending)
	if err != nil {
		return pixezQueueStats{}, err
	}
	return pixezQueueStats{Running: running, Queued: queued}, nil
}

func pixezTaskTypes() []string {
	return []string{
		TaskTypePixezMirror,
		TaskTypePixezExportBookmarks,
		TaskTypePixezAutoEnqueueBookmarkMirrors,
		TaskTypePixezImportLegacyServer,
	}
}

func listBookmarkExportRuns(ctx context.Context, page int, pageSize int) ([]pixezExportRunDTO, int64, error) {
	query := db.DB(ctx).Model(&model.PixezBookmarkExportRun{})
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var runs []model.PixezBookmarkExportRun
	if err := query.Order("started_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&runs).Error; err != nil {
		return nil, 0, err
	}
	items := make([]pixezExportRunDTO, 0, len(runs))
	for _, run := range runs {
		items = append(items, exportRunDTO(run))
	}
	return items, total, nil
}

func exportRunDTO(run model.PixezBookmarkExportRun) pixezExportRunDTO {
	return pixezExportRunDTO{
		ID:             run.ID,
		TargetType:     run.TargetType,
		PixivUserID:    run.PixivUserID,
		Status:         run.Status,
		TotalCount:     run.TotalCount,
		NewCount:       run.NewCount,
		UpdatedCount:   run.UpdatedCount,
		RemovedCount:   run.RemovedCount,
		ErrorMessage:   run.ErrorMessage,
		StartedAt:      run.StartedAt,
		FinishedAt:     run.FinishedAt,
		DurationMS:     exportRunDurationMS(run),
		LastRequestURL: run.LastRequestURL,
	}
}

func exportRunDurationMS(run model.PixezBookmarkExportRun) int64 {
	end := time.Now()
	if run.FinishedAt != nil {
		end = *run.FinishedAt
	}
	if run.StartedAt.IsZero() || end.Before(run.StartedAt) {
		return 0
	}
	return end.Sub(run.StartedAt).Milliseconds()
}

//nolint:dupl // Illustration and novel list DTOs intentionally stay parallel for explicit API shapes.
func listBookmarkIllusts(
	ctx context.Context,
	req listPixezBookmarksRequest,
	page int,
	pageSize int,
) ([]pixezIllustBookmarkDTO, int64, error) {
	query := applyBookmarkFilters(db.DB(ctx).Model(&model.PixezBookmarkIllust{}), req, "illust")
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.PixezBookmarkIllust
	if err := query.Order("created_at desc, id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]pixezIllustBookmarkDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, illustBookmarkDTO(row))
	}
	return items, total, nil
}

//nolint:dupl // Illustration and novel list DTOs intentionally stay parallel for explicit API shapes.
func listBookmarkNovels(
	ctx context.Context,
	req listPixezBookmarksRequest,
	page int,
	pageSize int,
) ([]pixezNovelBookmarkDTO, int64, error) {
	query := applyBookmarkFilters(db.DB(ctx).Model(&model.PixezBookmarkNovel{}), req, "novel")
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.PixezBookmarkNovel
	if err := query.Order("created_at desc, id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]pixezNovelBookmarkDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, novelBookmarkDTO(row))
	}
	return items, total, nil
}

func getBookmarkIllustDetail(ctx context.Context, illustID int64) (pixezIllustBookmarkDetailDTO, error) {
	var row model.PixezBookmarkIllust
	if err := db.DB(ctx).Where("illust_id = ?", illustID).First(&row).Error; err != nil {
		return pixezIllustBookmarkDetailDTO{}, err
	}
	var mirror model.PixezMirrorIllust
	err := db.DB(ctx).Where("illust_id = ?", illustID).First(&mirror).Error
	if err != nil && !isRecordNotFound(err) {
		return pixezIllustBookmarkDetailDTO{}, err
	}
	detail := pixezIllustBookmarkDetailDTO{
		Item:        illustBookmarkDTO(row),
		ImageFiles:  []model.PixezMirrorImageFile{},
		RequestURLs: []string{},
		RetryURLs:   []string{},
		IllustJSON:  json.RawMessage(row.IllustJSON),
	}
	if err == nil {
		imageFiles, err := decodeImageFiles(mirror.ImageFilesJSON)
		if err != nil {
			return pixezIllustBookmarkDetailDTO{}, fmt.Errorf("decode image files: %w", err)
		}
		requestURLs, err := decodeStringSlice(mirror.RequestURLsJSON)
		if err != nil {
			return pixezIllustBookmarkDetailDTO{}, fmt.Errorf("decode request URLs: %w", err)
		}
		retryURLs, err := decodeStringSlice(mirror.RetryURLsJSON)
		if err != nil {
			return pixezIllustBookmarkDetailDTO{}, fmt.Errorf("decode retry URLs: %w", err)
		}
		mirrorDetail := illustMirrorDetailDTO(mirror)
		detail.Mirror = &mirrorDetail
		detail.ImageFiles = imageFiles
		detail.RequestURLs = requestURLs
		detail.RetryURLs = retryURLs
	}
	return detail, nil
}

func getBookmarkNovelDetail(ctx context.Context, novelID int64) (pixezNovelBookmarkDetailDTO, error) {
	var row model.PixezBookmarkNovel
	if err := db.DB(ctx).Where("novel_id = ?", novelID).First(&row).Error; err != nil {
		return pixezNovelBookmarkDetailDTO{}, err
	}
	var mirror model.PixezMirrorNovel
	err := db.DB(ctx).Where("novel_id = ?", novelID).First(&mirror).Error
	if err != nil && !isRecordNotFound(err) {
		return pixezNovelBookmarkDetailDTO{}, err
	}
	detail := pixezNovelBookmarkDetailDTO{
		Item:        novelBookmarkDTO(row),
		RequestURLs: []string{},
		RetryURLs:   []string{},
		NovelJSON:   json.RawMessage(row.NovelJSON),
	}
	if err == nil {
		requestURLs, err := decodeStringSlice(mirror.RequestURLsJSON)
		if err != nil {
			return pixezNovelBookmarkDetailDTO{}, fmt.Errorf("decode request URLs: %w", err)
		}
		retryURLs, err := decodeStringSlice(mirror.RetryURLsJSON)
		if err != nil {
			return pixezNovelBookmarkDetailDTO{}, fmt.Errorf("decode retry URLs: %w", err)
		}
		mirrorDetail := novelMirrorDetailDTO(mirror)
		detail.Mirror = &mirrorDetail
		detail.RequestURLs = requestURLs
		detail.RetryURLs = retryURLs
	}
	return detail, nil
}

func applyBookmarkFilters(query *gorm.DB, req listPixezBookmarksRequest, targetType string) *gorm.DB {
	if strings.TrimSpace(req.WorkStatus) != "all" {
		query = applyWorkStatusFilter(query, req.WorkStatus)
	}
	if req.PixivUserID != "" {
		query = query.Where("pixiv_user_id = ?", req.PixivUserID)
	}
	if status, ok := bookmarkMirrorStatusFromQuery(req.MirrorStatus); ok {
		query = query.Where("mirror_status = ?", status)
	}
	queryText := strings.TrimSpace(req.Query)
	if queryText == "" {
		return query
	}
	like := "%" + queryText + "%"
	idColumn := "illust_id"
	if targetType == "novel" {
		idColumn = "novel_id"
	}
	if id, err := strconv.ParseInt(queryText, 10, 64); err == nil && id > 0 {
		return query.Where(
			"(title LIKE ? OR user_name LIKE ? OR pixiv_user_id LIKE ? OR "+idColumn+" = ?)",
			like,
			like,
			like,
			id,
		)
	}
	return query.Where("(title LIKE ? OR user_name LIKE ? OR pixiv_user_id LIKE ?)", like, like, like)
}

func applyWorkStatusFilter(query *gorm.DB, status string) *gorm.DB {
	switch status {
	case "visible":
		return query.Where("removed = ? AND visible = ? AND is_muted = ?", false, true, false)
	case "muted":
		return query.Where("removed = ? AND is_muted = ?", false, true)
	case "unavailable":
		return query.Where("removed = ? AND visible = ?", false, false)
	case "removed":
		return query.Where("removed = ?", true)
	default:
		return query.Where("removed = ?", false)
	}
}

func illustBookmarkDTO(row model.PixezBookmarkIllust) pixezIllustBookmarkDTO {
	return pixezIllustBookmarkDTO{
		ID:               row.ID,
		PixivUserID:      row.PixivUserID,
		Restrict:         row.Restrict,
		IllustID:         row.IllustID,
		Title:            row.Title,
		Type:             row.Type,
		UserID:           row.UserID,
		UserName:         row.UserName,
		CoverURL:         illustCoverURL(row.IllustJSON),
		PageCount:        row.PageCount,
		Width:            row.Width,
		Height:           row.Height,
		SanityLevel:      row.SanityLevel,
		XRestrict:        row.XRestrict,
		TotalBookmarks:   row.TotalBookmarks,
		Visible:          row.Visible,
		IsMuted:          row.IsMuted,
		MirrorStatus:     row.MirrorStatus,
		MirrorStatusText: bookmarkMirrorStatusText(row.MirrorStatus),
		MirrorRetryCount: row.MirrorRetryCount,
		Removed:          row.Removed,
		RemovedAt:        row.RemovedAt,
		LastSeenAt:       row.LastSeenAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

func novelBookmarkDTO(row model.PixezBookmarkNovel) pixezNovelBookmarkDTO {
	return pixezNovelBookmarkDTO{
		ID:               row.ID,
		PixivUserID:      row.PixivUserID,
		Restrict:         row.Restrict,
		NovelID:          row.NovelID,
		Title:            row.Title,
		Caption:          row.Caption,
		UserID:           row.UserID,
		UserName:         row.UserName,
		CoverURL:         firstNonEmpty(row.CoverURL, novelCoverURL(row.NovelJSON)),
		TextLength:       row.TextLength,
		XRestrict:        row.XRestrict,
		TotalBookmarks:   row.TotalBookmarks,
		IsOriginal:       row.IsOriginal,
		Visible:          row.Visible,
		IsMuted:          row.IsMuted,
		SeriesID:         row.SeriesID,
		SeriesTitle:      row.SeriesTitle,
		MirrorStatus:     row.MirrorStatus,
		MirrorStatusText: bookmarkMirrorStatusText(row.MirrorStatus),
		MirrorRetryCount: row.MirrorRetryCount,
		Removed:          row.Removed,
		RemovedAt:        row.RemovedAt,
		LastSeenAt:       row.LastSeenAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

func illustMirrorDetailDTO(row model.PixezMirrorIllust) pixezMirrorDetailDTO {
	return pixezMirrorDetailDTO{
		TaskID:       row.TaskID,
		Status:       row.Status,
		TotalCount:   row.TotalCount,
		SuccessCount: row.SuccessCount,
		FailedCount:  row.FailedCount,
		ErrorMessage: row.ErrorMessage,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}

func novelMirrorDetailDTO(row model.PixezMirrorNovel) pixezMirrorDetailDTO {
	return pixezMirrorDetailDTO{
		TaskID:       row.TaskID,
		Status:       row.Status,
		TotalCount:   row.TotalCount,
		SuccessCount: row.SuccessCount,
		FailedCount:  row.FailedCount,
		ErrorMessage: row.ErrorMessage,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}

func bindBookmarkListRequest(c *gin.Context) listPixezBookmarksRequest {
	return listPixezBookmarksRequest{
		Query:        strings.TrimSpace(c.Query("q")),
		PixivUserID:  strings.TrimSpace(c.Query("pixiv_user_id")),
		MirrorStatus: strings.TrimSpace(c.Query("mirror_status")),
		WorkStatus:   strings.TrimSpace(c.DefaultQuery("work_status", "active")),
	}
}

func parseManagementPage(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", strconv.Itoa(defaultManagementPageSize)))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultManagementPageSize
	}
	if pageSize > maxManagementPageSize {
		pageSize = maxManagementPageSize
	}
	return page, pageSize
}

func bookmarkMirrorStatusFromQuery(value string) (int, bool) {
	switch value {
	case "success", "succeeded", "done":
		return model.PixezBookmarkMirrorDone, true
	case "processing", "downloading":
		return model.PixezBookmarkMirrorProcessing, true
	case "failed":
		return model.PixezBookmarkMirrorFailed, true
	case "none", "not_queued":
		return model.PixezBookmarkMirrorNone, true
	default:
		return 0, false
	}
}

func bookmarkMirrorStatusText(status int) string {
	switch status {
	case model.PixezBookmarkMirrorDone:
		return "success"
	case model.PixezBookmarkMirrorProcessing:
		return "processing"
	case model.PixezBookmarkMirrorFailed:
		return "failed"
	default:
		return "none"
	}
}

func percent(value int64, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(value) / float64(total) * percentScale
}

func decodeStringSlice(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return []string{}, nil
	}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, err
	}
	if values == nil {
		return []string{}, nil
	}
	return values, nil
}

func decodeImageFiles(raw string) ([]model.PixezMirrorImageFile, error) {
	if strings.TrimSpace(raw) == "" {
		return []model.PixezMirrorImageFile{}, nil
	}
	var files []model.PixezMirrorImageFile
	if err := json.Unmarshal([]byte(raw), &files); err != nil {
		return nil, err
	}
	if files == nil {
		return []model.PixezMirrorImageFile{}, nil
	}
	return files, nil
}

func illustCoverURL(raw string) string {
	var payload struct {
		ImageURLs struct {
			SquareMedium string `json:"square_medium"`
			Medium       string `json:"medium"`
			Large        string `json:"large"`
		} `json:"image_urls"`
	}
	// Cover extraction is best-effort; DB title/artist fields remain authoritative.
	_ = json.Unmarshal([]byte(raw), &payload)
	return firstNonEmpty(payload.ImageURLs.SquareMedium, payload.ImageURLs.Medium, payload.ImageURLs.Large)
}

func novelCoverURL(raw string) string {
	var payload struct {
		ImageURLs struct {
			Medium string `json:"medium"`
			Large  string `json:"large"`
		} `json:"image_urls"`
	}
	// Cover extraction is best-effort; DB title/artist fields remain authoritative.
	_ = json.Unmarshal([]byte(raw), &payload)
	return firstNonEmpty(payload.ImageURLs.Medium, payload.ImageURLs.Large)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func isRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
