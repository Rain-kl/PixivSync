// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package pixez

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	uploadapp "github.com/Rain-kl/Wavelet/internal/apps/upload"
	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/pkg/logger"
	"github.com/Rain-kl/Wavelet/internal/model"
	pixezsvc "github.com/Rain-kl/Wavelet/internal/service/pixez"
	"github.com/Rain-kl/Wavelet/internal/task"
	"github.com/Rain-kl/Wavelet/internal/common/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type batchIllustMirrorRequest struct {
	IllustIDs []int64 `json:"illust_ids"`
}

type batchNovelMirrorRequest struct {
	NovelIDs []int64 `json:"novel_ids"`
}

type listMirrorsRequest struct {
	Query  string
	Status string
}

type pixezMirroredIllustDTO struct {
	IllustID       int64     `json:"illust_id"`
	Title          string    `json:"title"`
	Type           string    `json:"type"`
	UserID         int64     `json:"user_id"`
	UserName       string    `json:"user_name"`
	CoverURL       string    `json:"cover_url"`
	PageCount      int       `json:"page_count"`
	Width          int       `json:"width"`
	Height         int       `json:"height"`
	SanityLevel    int       `json:"sanity_level"`
	XRestrict      int       `json:"x_restrict"`
	TotalBookmarks int       `json:"total_bookmarks"`
	Visible        bool      `json:"visible"`
	IsMuted        bool      `json:"is_muted"`
	TaskID         string    `json:"task_id"`
	Status         string    `json:"status"`
	StatusText     string    `json:"status_text"`
	TotalCount     int       `json:"total_count"`
	SuccessCount   int       `json:"success_count"`
	FailedCount    int       `json:"failed_count"`
	ErrorMessage   string    `json:"error_message"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type pixezMirroredNovelDTO struct {
	NovelID        int64     `json:"novel_id"`
	Title          string    `json:"title"`
	Caption        string    `json:"caption"`
	UserID         int64     `json:"user_id"`
	UserName       string    `json:"user_name"`
	CoverURL       string    `json:"cover_url"`
	TextLength     int       `json:"text_length"`
	XRestrict      int       `json:"x_restrict"`
	TotalBookmarks int       `json:"total_bookmarks"`
	IsOriginal     bool      `json:"is_original"`
	IsMuted        bool      `json:"is_muted"`
	SeriesID       *int64    `json:"series_id"`
	SeriesTitle    *string   `json:"series_title"`
	TaskID         string    `json:"task_id"`
	Status         string    `json:"status"`
	StatusText     string    `json:"status_text"`
	TotalCount     int       `json:"total_count"`
	SuccessCount   int       `json:"success_count"`
	FailedCount    int       `json:"failed_count"`
	ErrorMessage   string    `json:"error_message"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type pixezMirroredIllustDetailDTO struct {
	Item        pixezMirroredIllustDTO       `json:"item"`
	Mirror      pixezMirrorDetailDTO         `json:"mirror"`
	ImageFiles  []model.PixezMirrorImageFile `json:"image_files"`
	RequestURLs []string                     `json:"request_urls"`
	RetryURLs   []string                     `json:"retry_urls"`
	IllustJSON  json.RawMessage              `json:"illust_json"`
}

type pixezMirroredNovelDetailDTO struct {
	Item        pixezMirroredNovelDTO `json:"item"`
	Mirror      pixezMirrorDetailDTO  `json:"mirror"`
	RequestURLs []string              `json:"request_urls"`
	RetryURLs   []string              `json:"retry_urls"`
	NovelJSON   json.RawMessage       `json:"novel_json"`
}

// MirrorIllust dispatches an illustration mirror task.
func MirrorIllust(c *gin.Context) {
	illustID, ok := parsePositiveIDParam(c, "illust_id")
	if !ok {
		return
	}
	record, err := dispatchIllustMirrorIfNeeded(c, illustID)
	if err != nil {
		c.JSON(http.StatusOK, response.Err(errDispatchMirrorTaskFailed))
		return
	}
	c.JSON(http.StatusOK, response.OK(pixezsvc.MirrorIllustStatus(record)))
}

// CheckIllustMirror returns illustration mirror status.
func CheckIllustMirror(c *gin.Context) {
	illustID, ok := parsePositiveIDParam(c, "illust_id")
	if !ok {
		return
	}
	record, err := pixezsvc.GetMirrorIllust(c.Request.Context(), illustID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusOK, response.Err(errQueryMirrorStatusFailed))
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		record = model.PixezMirrorIllust{IllustID: illustID, Status: ""}
	}
	c.JSON(http.StatusOK, response.OK(pixezsvc.MirrorIllustStatus(record)))
}

// BatchCheckIllustMirror returns mirrored illustration IDs.
func BatchCheckIllustMirror(c *gin.Context) {
	var req batchIllustMirrorRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IllustIDs) == 0 {
		c.JSON(http.StatusBadRequest, response.Err(errInvalidRequestBody))
		return
	}
	respondBatchMirror(c, req.IllustIDs, mirroredIllustIDs)
}

// MirrorNovel dispatches a novel mirror task.
func MirrorNovel(c *gin.Context) {
	novelID, ok := parsePositiveIDParam(c, "novel_id")
	if !ok {
		return
	}
	record, err := dispatchNovelMirrorIfNeeded(c, novelID)
	if err != nil {
		c.JSON(http.StatusOK, response.Err(errDispatchMirrorTaskFailed))
		return
	}
	c.JSON(http.StatusOK, response.OK(pixezsvc.MirrorNovelStatus(record)))
}

// CheckNovelMirror returns novel mirror status.
func CheckNovelMirror(c *gin.Context) {
	novelID, ok := parsePositiveIDParam(c, "novel_id")
	if !ok {
		return
	}
	record, err := pixezsvc.GetMirrorNovel(c.Request.Context(), novelID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusOK, response.Err(errQueryMirrorStatusFailed))
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		record = model.PixezMirrorNovel{NovelID: novelID, Status: ""}
	}
	c.JSON(http.StatusOK, response.OK(pixezsvc.MirrorNovelStatus(record)))
}

// BatchCheckNovelMirror returns mirrored novel IDs.
func BatchCheckNovelMirror(c *gin.Context) {
	var req batchNovelMirrorRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.NovelIDs) == 0 {
		c.JSON(http.StatusBadRequest, response.Err(errInvalidRequestBody))
		return
	}
	respondBatchMirror(c, req.NovelIDs, mirroredNovelIDs)
}

func respondBatchMirror(c *gin.Context, requestedIDs []int64, query func(context.Context, []int64) ([]int64, error)) {
	if len(requestedIDs) > maxBatchMirrorIDs {
		c.JSON(http.StatusBadRequest, response.Err(errTooManyMirrorIDs))
		return
	}
	ids, err := query(c.Request.Context(), requestedIDs)
	if err != nil {
		c.JSON(http.StatusOK, response.Err(errQueryMirrorStatusFailed))
		return
	}
	c.JSON(http.StatusOK, response.OK(gin.H{"mirrored_ids": ids}))
}

func mirroredIllustIDs(ctx context.Context, requestedIDs []int64) ([]int64, error) {
	var records []model.PixezMirrorIllust
	if err := db.DB(ctx).
		Where("illust_id IN ? AND success_count > 0", requestedIDs).
		Find(&records).Error; err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.IllustID)
	}
	return ids, nil
}

func mirroredNovelIDs(ctx context.Context, requestedIDs []int64) ([]int64, error) {
	var records []model.PixezMirrorNovel
	if err := db.DB(ctx).
		Where("novel_id IN ? AND success_count > 0", requestedIDs).
		Find(&records).Error; err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.NovelID)
	}
	return ids, nil
}

// GetMirroredIllustDetail returns Pixiv-shape mirrored illustration detail.
func GetMirroredIllustDetail(c *gin.Context) {
	illustID, ok := parsePositiveQueryID(c, "illust_id")
	if !ok {
		return
	}
	record, err := pixezsvc.GetMirrorIllust(c.Request.Context(), illustID)
	if err != nil || record.DetailJSON == "" {
		c.JSON(http.StatusNotFound, gin.H{keyError: "mirror not found"})
		return
	}
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.String(http.StatusOK, pixezsvc.RewritePximgURLs(record.DetailJSON, mirrorURLPrefix(c)))
}

// ServeMirroredImage streams a mirrored pximg file from Upload storage.
// @Summary Get PixEz mirrored image
// @Description Returns the original mirrored image by default, or a cached WebP conversion when quality is low, medium, or high.
// @Tags pixez
// @Produce octet-stream
// @Security SessionCookie
// @Param path path string true "Pixiv image path"
// @Param quality query string false "Image quality: low, medium, high, or origin. Defaults to origin."
// @Success 200 {file} file "Mirrored image"
// @Failure 404 {object} object "Mirror file not found"
// @Router /mirror/pximg/{path} [get]
func ServeMirroredImage(c *gin.Context) {
	upload, err := pixezsvc.FindMirroredImageUpload(c.Request.Context(), c.Param("path"))
	if err == nil {
		uploadapp.ServeUpload(c, &upload)
		return
	}

	// Fallback to proxying from Pixiv on the fly
	cleanPath := strings.TrimPrefix(c.Param("path"), "/")
	if cleanPath == "" || strings.Contains(cleanPath, "..") {
		c.JSON(http.StatusNotFound, gin.H{keyError: "mirror file not found"})
		return
	}

	pixivURL := "https://i.pximg.net/" + cleanPath
	data, mimeType, err := pixezsvc.DefaultClient.DownloadFile(c.Request.Context(), pixivURL)
	if err != nil {
		// Try s.pximg.net as fallback
		pixivURL = "https://s.pximg.net/" + cleanPath
		data, mimeType, err = pixezsvc.DefaultClient.DownloadFile(c.Request.Context(), pixivURL)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{keyError: "mirror file not found"})
			return
		}
	}

	if quality := c.Query("quality"); isCompressedImageQuality(quality) {
		webpData, compressErr := uploadapp.CompressImageToWebP(bytes.NewReader(data), quality)
		if compressErr == nil {
			c.Header("Content-Type", "image/webp")
			c.Header("Content-Length", strconv.Itoa(len(webpData)))
			_, _ = c.Writer.Write(webpData)
			return
		}
		logger.WarnF(c.Request.Context(), "[PixEz] compress proxied mirror image failed path=%s quality=%s: %v", cleanPath, quality, compressErr)
	}

	c.Header("Content-Type", mimeType)
	c.Header("Content-Length", strconv.Itoa(len(data)))
	_, _ = c.Writer.Write(data)
}

func isCompressedImageQuality(quality string) bool {
	switch strings.ToLower(quality) {
	case "low", "medium", "high":
		return true
	default:
		return false
	}
}

// GetMirroredNovelDetail returns Pixiv-shape mirrored novel detail.
func GetMirroredNovelDetail(c *gin.Context) {
	novelID, ok := parsePositiveQueryID(c, "novel_id")
	if !ok {
		return
	}
	record, err := pixezsvc.GetMirrorNovel(c.Request.Context(), novelID)
	if err != nil || record.DetailJSON == "" {
		c.JSON(http.StatusNotFound, gin.H{keyError: "novel mirror not found"})
		return
	}
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.String(http.StatusOK, pixezsvc.RewritePximgURLs(record.DetailJSON, mirrorURLPrefix(c)))
}

// GetMirroredNovelText returns Pixiv webview JSON for mirrored novel text.
func GetMirroredNovelText(c *gin.Context) {
	novelID, ok := parsePositiveQueryID(c, "novel_id")
	if !ok {
		return
	}
	record, err := pixezsvc.GetMirrorNovel(c.Request.Context(), novelID)
	if err != nil || record.TextJSON == "" {
		c.JSON(http.StatusNotFound, gin.H{keyError: "novel mirror not found"})
		return
	}
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.String(http.StatusOK, pixezsvc.RewritePximgURLs(record.TextJSON, mirrorURLPrefix(c)))
}

// ListMirroredIllusts returns paginated mirrored illustration read-model rows.
// @Summary List PixEz mirrored illustrations
// @Description Returns all illustration mirror read-models, including bookmark, automatic, and manual mirror requests.
// @Tags pixez
// @Produce json
// @Security SessionCookie
// @Param q query string false "Search by illustration ID, title, or artist"
// @Param status query string false "Mirror status: success, processing, failed"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(24)
// @Success 200 {object} response.Any{data=object}
// @Router /api/pixez/mirror/illusts [get]
func ListMirroredIllusts(c *gin.Context) {
	page, pageSize := parseManagementPage(c)
	items, total, err := listMirroredIllusts(c.Request.Context(), bindMirrorListRequest(c), page, pageSize)
	if err != nil {
		logger.ErrorF(c.Request.Context(), "[PixEz] list mirrored illustrations failed: %v", err)
		c.JSON(http.StatusOK, response.Err(errQueryMirrorStatusFailed))
		return
	}
	c.JSON(http.StatusOK, response.OK(gin.H{keyItems: items, keyTotal: total, keyPage: page, keyPageSize: pageSize}))
}

// ListMirroredNovels returns paginated mirrored novel read-model rows.
// @Summary List PixEz mirrored novels
// @Description Returns all novel mirror read-models, including bookmark, automatic, and manual mirror requests.
// @Tags pixez
// @Produce json
// @Security SessionCookie
// @Param q query string false "Search by novel ID, title, or author"
// @Param status query string false "Mirror status: success, processing, failed"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(24)
// @Success 200 {object} response.Any{data=object}
// @Router /api/pixez/mirror/novels [get]
func ListMirroredNovels(c *gin.Context) {
	page, pageSize := parseManagementPage(c)
	items, total, err := listMirroredNovels(c.Request.Context(), bindMirrorListRequest(c), page, pageSize)
	if err != nil {
		logger.ErrorF(c.Request.Context(), "[PixEz] list mirrored novels failed: %v", err)
		c.JSON(http.StatusOK, response.Err(errQueryMirrorStatusFailed))
		return
	}
	c.JSON(http.StatusOK, response.OK(gin.H{keyItems: items, keyTotal: total, keyPage: page, keyPageSize: pageSize}))
}

// GetMirroredIllustManagementDetail returns one illustration mirror read-model.
// @Summary Get PixEz mirrored illustration detail
// @Description Returns illustration metadata and mirror diagnostics without requiring a bookmark record.
// @Tags pixez
// @Produce json
// @Security SessionCookie
// @Param illust_id path int true "Pixiv illustration ID"
// @Success 200 {object} response.Any{data=object}
// @Router /api/pixez/mirror/illusts/{illust_id}/detail [get]
func GetMirroredIllustManagementDetail(c *gin.Context) {
	illustID, ok := parsePositiveIDParam(c, "illust_id")
	if !ok {
		return
	}
	detail, err := getMirroredIllustManagementDetail(c.Request.Context(), illustID)
	if err != nil {
		logger.ErrorF(c.Request.Context(), "[PixEz] get mirrored illustration detail failed illust_id=%d: %v", illustID, err)
		c.JSON(http.StatusOK, response.Err(errFetchMirrorDetailFailed))
		return
	}
	c.JSON(http.StatusOK, response.OK(detail))
}

// GetMirroredNovelManagementDetail returns one novel mirror read-model.
// @Summary Get PixEz mirrored novel detail
// @Description Returns novel metadata and mirror diagnostics without requiring a bookmark record.
// @Tags pixez
// @Produce json
// @Security SessionCookie
// @Param novel_id path int true "Pixiv novel ID"
// @Success 200 {object} response.Any{data=object}
// @Router /api/pixez/mirror/novels/{novel_id}/detail [get]
func GetMirroredNovelManagementDetail(c *gin.Context) {
	novelID, ok := parsePositiveIDParam(c, "novel_id")
	if !ok {
		return
	}
	detail, err := getMirroredNovelManagementDetail(c.Request.Context(), novelID)
	if err != nil {
		logger.ErrorF(c.Request.Context(), "[PixEz] get mirrored novel detail failed novel_id=%d: %v", novelID, err)
		c.JSON(http.StatusOK, response.Err(errFetchMirrorDetailFailed))
		return
	}
	c.JSON(http.StatusOK, response.OK(detail))
}

//nolint:dupl // Illustration and novel mirror lists intentionally keep explicit DTO and model types.
func listMirroredIllusts(
	ctx context.Context,
	req listMirrorsRequest,
	page int,
	pageSize int,
) ([]pixezMirroredIllustDTO, int64, error) {
	query := applyMirrorFilters(db.DB(ctx).Model(&model.PixezMirrorIllust{}), req, "illust_id")
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var records []model.PixezMirrorIllust
	if err := query.Order("updated_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, err
	}
	items := make([]pixezMirroredIllustDTO, 0, len(records))
	for _, record := range records {
		items = append(items, mirroredIllustDTO(record))
	}
	return items, total, nil
}

//nolint:dupl // Illustration and novel mirror lists intentionally keep explicit DTO and model types.
func listMirroredNovels(
	ctx context.Context,
	req listMirrorsRequest,
	page int,
	pageSize int,
) ([]pixezMirroredNovelDTO, int64, error) {
	query := applyMirrorFilters(db.DB(ctx).Model(&model.PixezMirrorNovel{}), req, "novel_id")
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var records []model.PixezMirrorNovel
	if err := query.Order("updated_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, err
	}
	items := make([]pixezMirroredNovelDTO, 0, len(records))
	for _, record := range records {
		items = append(items, mirroredNovelDTO(record))
	}
	return items, total, nil
}

func getMirroredIllustManagementDetail(ctx context.Context, illustID int64) (pixezMirroredIllustDetailDTO, error) {
	var record model.PixezMirrorIllust
	if err := db.DB(ctx).Where("illust_id = ?", illustID).First(&record).Error; err != nil {
		return pixezMirroredIllustDetailDTO{}, err
	}
	imageFiles, err := decodeImageFiles(record.ImageFilesJSON)
	if err != nil {
		return pixezMirroredIllustDetailDTO{}, fmt.Errorf("decode image files: %w", err)
	}
	requestURLs, err := decodeStringSlice(record.RequestURLsJSON)
	if err != nil {
		return pixezMirroredIllustDetailDTO{}, fmt.Errorf("decode request URLs: %w", err)
	}
	retryURLs, err := decodeStringSlice(record.RetryURLsJSON)
	if err != nil {
		return pixezMirroredIllustDetailDTO{}, fmt.Errorf("decode retry URLs: %w", err)
	}
	return pixezMirroredIllustDetailDTO{
		Item:        mirroredIllustDTO(record),
		Mirror:      illustMirrorDetailDTO(record),
		ImageFiles:  imageFiles,
		RequestURLs: requestURLs,
		RetryURLs:   retryURLs,
		IllustJSON:  rawJSON(record.DetailJSON),
	}, nil
}

func getMirroredNovelManagementDetail(ctx context.Context, novelID int64) (pixezMirroredNovelDetailDTO, error) {
	var record model.PixezMirrorNovel
	if err := db.DB(ctx).Where("novel_id = ?", novelID).First(&record).Error; err != nil {
		return pixezMirroredNovelDetailDTO{}, err
	}
	requestURLs, err := decodeStringSlice(record.RequestURLsJSON)
	if err != nil {
		return pixezMirroredNovelDetailDTO{}, fmt.Errorf("decode request URLs: %w", err)
	}
	retryURLs, err := decodeStringSlice(record.RetryURLsJSON)
	if err != nil {
		return pixezMirroredNovelDetailDTO{}, fmt.Errorf("decode retry URLs: %w", err)
	}
	return pixezMirroredNovelDetailDTO{
		Item:        mirroredNovelDTO(record),
		Mirror:      novelMirrorDetailDTO(record),
		RequestURLs: requestURLs,
		RetryURLs:   retryURLs,
		NovelJSON:   rawJSON(record.DetailJSON),
	}, nil
}

func applyMirrorFilters(query *gorm.DB, req listMirrorsRequest, idColumn string) *gorm.DB {
	switch req.Status {
	case statusSuccess:
		query = query.Where("status = ?", model.PixezMirrorStatusSuccess)
	case statusProcessing:
		query = query.Where("status IN ?", []string{model.PixezMirrorStatusQueued, model.PixezMirrorStatusProcessing})
	case statusFailed:
		query = query.Where("status = ?", model.PixezMirrorStatusFailed)
	}
	queryText := strings.TrimSpace(req.Query)
	if queryText == "" {
		return query
	}
	like := "%" + queryText + "%"
	if id, err := strconv.ParseInt(queryText, 10, 64); err == nil && id > 0 {
		return query.Where("("+idColumn+" = ? OR detail_json LIKE ?)", id, like)
	}
	return query.Where("detail_json LIKE ?", like)
}

func bindMirrorListRequest(c *gin.Context) listMirrorsRequest {
	return listMirrorsRequest{
		Query:  strings.TrimSpace(c.Query("q")),
		Status: strings.TrimSpace(c.Query("status")),
	}
}

func mirroredIllustDTO(record model.PixezMirrorIllust) pixezMirroredIllustDTO {
	var detail pixezsvc.IllustDetail
	_ = json.Unmarshal([]byte(record.DetailJSON), &detail)
	illust := detail.Illust
	return pixezMirroredIllustDTO{
		IllustID:       record.IllustID,
		Title:          illust.Title,
		Type:           illust.Type,
		UserID:         illust.User.ID,
		UserName:       illust.User.Name,
		CoverURL:       firstNonEmpty(illust.ImageUrls.SquareMedium, illust.ImageUrls.Medium, illust.ImageUrls.Large),
		PageCount:      illust.PageCount,
		Width:          illust.Width,
		Height:         illust.Height,
		SanityLevel:    illust.SanityLevel,
		XRestrict:      illust.XRestrict,
		TotalBookmarks: illust.TotalBookmarks,
		Visible:        illust.Visible,
		IsMuted:        illust.IsMuted,
		TaskID:         record.TaskID,
		Status:         record.Status,
		StatusText:     mirrorStatusText(record.Status),
		TotalCount:     record.TotalCount,
		SuccessCount:   record.SuccessCount,
		FailedCount:    record.FailedCount,
		ErrorMessage:   record.ErrorMessage,
		CreatedAt:      record.CreatedAt,
		UpdatedAt:      record.UpdatedAt,
	}
}

func mirroredNovelDTO(record model.PixezMirrorNovel) pixezMirroredNovelDTO {
	var detail pixezsvc.NovelDetail
	_ = json.Unmarshal([]byte(record.DetailJSON), &detail)
	novel := detail.Novel
	dto := pixezMirroredNovelDTO{
		NovelID:        record.NovelID,
		Title:          novel.Title,
		Caption:        novel.Caption,
		UserID:         novel.User.ID,
		UserName:       novel.User.Name,
		CoverURL:       firstNonEmpty(novel.ImageUrls.SquareMedium, novel.ImageUrls.Medium, novel.ImageUrls.Large),
		TextLength:     novel.TextLength,
		XRestrict:      novel.XRestrict,
		TotalBookmarks: novel.TotalBookmarks,
		IsOriginal:     novel.IsOriginal,
		IsMuted:        novel.IsMuted,
		TaskID:         record.TaskID,
		Status:         record.Status,
		StatusText:     mirrorStatusText(record.Status),
		TotalCount:     record.TotalCount,
		SuccessCount:   record.SuccessCount,
		FailedCount:    record.FailedCount,
		ErrorMessage:   record.ErrorMessage,
		CreatedAt:      record.CreatedAt,
		UpdatedAt:      record.UpdatedAt,
	}
	if novel.Series != nil {
		dto.SeriesID = &novel.Series.ID
		dto.SeriesTitle = &novel.Series.Title
	}
	return dto
}

func mirrorStatusText(status string) string {
	switch status {
	case model.PixezMirrorStatusSuccess:
		return statusSuccess
	case model.PixezMirrorStatusFailed:
		return statusFailed
	default:
		return statusProcessing
	}
}

func rawJSON(value string) json.RawMessage {
	if json.Valid([]byte(value)) {
		return json.RawMessage(value)
	}
	return json.RawMessage("null")
}

// DeleteMirroredIllust deletes one mirrored illustration read-model and marks its uploads deleted.
func DeleteMirroredIllust(c *gin.Context) {
	illustID, ok := parsePositiveIDParam(c, "illust_id")
	if !ok {
		return
	}
	deleted, err := deleteMirroredIllust(c.Request.Context(), illustID)
	if err != nil {
		c.JSON(http.StatusOK, response.Err(errDeleteMirrorFailed))
		return
	}
	c.JSON(http.StatusOK, response.OK(gin.H{"deleted": deleted, "illust_id": illustID}))
}

// DeleteMirroredNovel deletes one mirrored novel read-model.
func DeleteMirroredNovel(c *gin.Context) {
	novelID, ok := parsePositiveIDParam(c, "novel_id")
	if !ok {
		return
	}
	if err := db.DB(c.Request.Context()).Where("novel_id = ?", novelID).Delete(&model.PixezMirrorNovel{}).Error; err != nil {
		c.JSON(http.StatusOK, response.Err(errDeleteMirrorFailed))
		return
	}
	c.JSON(http.StatusOK, response.OK(gin.H{"deleted": true, "novel_id": novelID}))
}

// BatchDeleteMirroredItems deletes mirrored items by target type.
func BatchDeleteMirroredItems(c *gin.Context) {
	var req struct {
		TargetType string  `json:"target_type"`
		IDs        []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, response.Err(errInvalidRequestBody))
		return
	}
	deleted := 0
	switch req.TargetType {
	case model.PixezMirrorTargetIllust:
		for _, id := range req.IDs {
			ok, err := deleteMirroredIllust(c.Request.Context(), id)
			if err != nil {
				c.JSON(http.StatusOK, response.Err(errDeleteMirrorFailed))
				return
			}
			if ok {
				deleted++
			}
		}
	case model.PixezMirrorTargetNovel:
		result := db.DB(c.Request.Context()).Where("novel_id IN ?", req.IDs).Delete(&model.PixezMirrorNovel{})
		if result.Error != nil {
			c.JSON(http.StatusOK, response.Err(errDeleteMirrorFailed))
			return
		}
		deleted = int(result.RowsAffected)
	default:
		c.JSON(http.StatusBadRequest, response.Err(errInvalidRequestBody))
		return
	}
	c.JSON(http.StatusOK, response.OK(gin.H{"deleted_count": deleted}))
}

//nolint:dupl // Illust and novel mirror dispatch flows keep parallel structures for distinct model types
func dispatchIllustMirrorIfNeeded(c *gin.Context, illustID int64) (model.PixezMirrorIllust, error) {
	record, err := pixezsvc.GetMirrorIllust(c.Request.Context(), illustID)
	if err == nil && record.Status != model.PixezMirrorStatusFailed {
		return record, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return record, err
	}
	payload, _ := json.Marshal(mirrorPayload{TargetType: TargetTypeIllust, TargetID: illustID})
	taskID, err := task.DispatchTask(c.Request.Context(), TaskTypePixezMirror, payload, "api")
	if err != nil {
		return record, err
	}
	return pixezsvc.EnsureMirrorIllustQueued(c.Request.Context(), illustID, taskID)
}

//nolint:dupl // Illust and novel mirror dispatch flows keep parallel structures for distinct model types
func dispatchNovelMirrorIfNeeded(c *gin.Context, novelID int64) (model.PixezMirrorNovel, error) {
	record, err := pixezsvc.GetMirrorNovel(c.Request.Context(), novelID)
	if err == nil && record.Status != model.PixezMirrorStatusFailed {
		return record, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return record, err
	}
	payload, _ := json.Marshal(mirrorPayload{TargetType: TargetTypeNovel, TargetID: novelID})
	taskID, err := task.DispatchTask(c.Request.Context(), TaskTypePixezMirror, payload, "api")
	if err != nil {
		return record, err
	}
	return pixezsvc.EnsureMirrorNovelQueued(c.Request.Context(), novelID, taskID)
}

func parsePositiveIDParam(c *gin.Context, key string) (int64, bool) {
	raw := c.Param(key)
	id, err := strconv.ParseInt(raw, 10, 64)
	if raw == "" || err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, response.Err(fmt.Sprintf("%s is required", key)))
		return 0, false
	}
	return id, true
}

func parsePositiveQueryID(c *gin.Context, key string) (int64, bool) {
	raw := c.Query(key)
	id, err := strconv.ParseInt(raw, 10, 64)
	if raw == "" || err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{keyError: fmt.Sprintf("%s is required", key)})
		return 0, false
	}
	return id, true
}

func parsePage(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}

func mirrorURLPrefix(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil || c.Request.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + c.Request.Host + "/mirror/pximg"
}

func deleteMirroredIllust(ctx context.Context, illustID int64) (bool, error) {
	var record model.PixezMirrorIllust
	err := db.DB(ctx).Where("illust_id = ?", illustID).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var files []model.PixezMirrorImageFile
	_ = json.Unmarshal([]byte(record.ImageFilesJSON), &files)
	for _, file := range files {
		if _, err := uploadapp.Remove(ctx, file.UploadID); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.WarnF(ctx, "failed to remove mirrored upload %d: %v", file.UploadID, err)
		}
	}
	if err := db.DB(ctx).Where("illust_id = ?", illustID).Delete(&model.PixezMirrorIllust{}).Error; err != nil {
		return false, err
	}
	return true, nil
}
