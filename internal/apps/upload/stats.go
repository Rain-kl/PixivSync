// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package upload

import (
	"net/http"
	"strings"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/util"
	"github.com/gin-gonic/gin"
)

const (
	catImage    = "图片"
	catVideo    = "视频"
	catAudio    = "音频"
	catDocument = "文档"
	catArchive  = "压缩包"
	catOther    = "其他"
)

type trendItem struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
	Size  int64  `json:"size"`
}

type distributionItem struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
	Size  int64  `json:"size"`
}

type fileStatsResponse struct {
	TotalCount int64              `json:"total_count"`
	TotalSize  int64              `json:"total_size"`
	Trend      []trendItem        `json:"trend"`
	Categories []distributionItem `json:"categories"`
	Types      []distributionItem `json:"types"`
}

// GetFileStats 获取系统上传的文件统计数据
// @Summary 获取文件统计数据
// @Description 返回系统级的总文件数、占用大小、最近 7 天新增趋势、文件类型/格式分布等数据
// @Tags admin
// @Produce json
// @Security SessionCookie
// @Success 200 {object} util.ResponseAny{data=fileStatsResponse} "获取成功"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Failure 403 {object} util.ResponseAny "无管理员权限"
// @Failure 500 {object} util.ResponseAny "内部错误"
// @Router /api/v1/admin/uploads/stats [get]
func GetFileStats(c *gin.Context) {
	ctx := c.Request.Context()

	// 1. 获取总文件数与总文件大小
	var summary struct {
		TotalCount int64 `json:"total_count"`
		TotalSize  int64 `json:"total_size"`
	}
	err := db.DB(ctx).Model(&model.Upload{}).
		Select("COUNT(*) as total_count, COALESCE(SUM(file_size), 0) as total_size").
		Where("status != ?", model.UploadStatusDeleted).
		Scan(&summary).Error
	if err != nil {
		c.JSON(http.StatusOK, util.Err(err.Error()))
		return
	}

	// 2. 获取业务类型分布 (Group By type)
	type rawDist struct {
		Key   string `gorm:"column:key"`
		Count int64  `gorm:"column:count"`
		Size  int64  `gorm:"column:size"`
	}
	var typeRaw []rawDist
	err = db.DB(ctx).Model(&model.Upload{}).
		Select("type as key, COUNT(*) as count, COALESCE(SUM(file_size), 0) as size").
		Where("status != ?", model.UploadStatusDeleted).
		Group("type").
		Scan(&typeRaw).Error
	if err != nil {
		c.JSON(http.StatusOK, util.Err(err.Error()))
		return
	}

	types := make([]distributionItem, 0, len(typeRaw))
	for _, tr := range typeRaw {
		name := tr.Key
		if name == "" {
			name = "generic"
		}
		types = append(types, distributionItem{
			Name:  name,
			Count: tr.Count,
			Size:  tr.Size,
		})
	}

	// 3. 获取所有文件的大小、后缀与MIME，用于在 Go 中内存分类统计 (避免数据库中写复杂的 JSON/String 匹配逻辑)
	type fileCategoryRaw struct {
		Extension string `gorm:"column:extension"`
		MimeType  string `gorm:"column:mime_type"`
		FileSize  int64  `gorm:"column:file_size"`
	}
	var fileRaws []fileCategoryRaw
	err = db.DB(ctx).Model(&model.Upload{}).
		Select("extension, mime_type, file_size").
		Where("status != ?", model.UploadStatusDeleted).
		Scan(&fileRaws).Error
	if err != nil {
		c.JSON(http.StatusOK, util.Err(err.Error()))
		return
	}

	catCount := make(map[string]int64)
	catSize := make(map[string]int64)
	categoriesList := []string{catImage, catVideo, catAudio, catDocument, catArchive, catOther}
	for _, cat := range categoriesList {
		catCount[cat] = 0
		catSize[cat] = 0
	}

	for _, fr := range fileRaws {
		cat := getFileCategory(fr.MimeType, fr.Extension)
		catCount[cat]++
		catSize[cat] += fr.FileSize
	}

	categories := make([]distributionItem, 0, len(categoriesList))
	for _, cat := range categoriesList {
		categories = append(categories, distributionItem{
			Name:  cat,
			Count: catCount[cat],
			Size:  catSize[cat],
		})
	}

	// 4. 获取近 7 天的新增文件趋势 (在 Go 中补全没有新增记录的日期为 0)
	type fileTrendRaw struct {
		CreatedAt time.Time `gorm:"column:created_at"`
		FileSize  int64     `gorm:"column:file_size"`
	}
	var trendRaws []fileTrendRaw
	// 7天前 00:00:00 (即 6 天前 00:00:00 至今天)
	now := time.Now()
	startTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -6)
	err = db.DB(ctx).Model(&model.Upload{}).
		Select("created_at, file_size").
		Where("status != ? AND created_at >= ?", model.UploadStatusDeleted, startTime).
		Scan(&trendRaws).Error
	if err != nil {
		c.JSON(http.StatusOK, util.Err(err.Error()))
		return
	}

	trendCountMap := make(map[string]int64)
	trendSizeMap := make(map[string]int64)
	for i := 0; i < 7; i++ {
		dStr := now.AddDate(0, 0, -i).Format("2006-01-02")
		trendCountMap[dStr] = 0
		trendSizeMap[dStr] = 0
	}

	for _, tr := range trendRaws {
		dStr := tr.CreatedAt.Format("2006-01-02")
		if _, exists := trendCountMap[dStr]; exists {
			trendCountMap[dStr]++
			trendSizeMap[dStr] += tr.FileSize
		}
	}

	const trendDays = 7
	trend := make([]trendItem, 0, trendDays)
	for i := trendDays - 1; i >= 0; i-- {
		dStr := now.AddDate(0, 0, -i).Format("2006-01-02")
		trend = append(trend, trendItem{
			Date:  dStr,
			Count: trendCountMap[dStr],
			Size:  trendSizeMap[dStr],
		})
	}

	c.JSON(http.StatusOK, util.OK(fileStatsResponse{
		TotalCount: summary.TotalCount,
		TotalSize:  summary.TotalSize,
		Trend:      trend,
		Categories: categories,
		Types:      types,
	}))
}

func getFileCategory(mimeType, ext string) string {
	mimeType = strings.ToLower(mimeType)
	ext = strings.ToLower(ext)

	if strings.HasPrefix(mimeType, "image/") || isImageExtension(ext) {
		return catImage
	}
	if strings.HasPrefix(mimeType, "video/") {
		return catVideo
	}
	if strings.HasPrefix(mimeType, "audio/") {
		return catAudio
	}
	if isArchiveExtension(ext) || strings.Contains(mimeType, "zip") || strings.Contains(mimeType, "tar") || strings.Contains(mimeType, "gzip") {
		return catArchive
	}
	if isDocumentExtension(ext) || strings.HasPrefix(mimeType, "text/") || mimeType == "application/pdf" {
		return catDocument
	}
	return catOther
}

func isArchiveExtension(ext string) bool {
	for _, e := range []string{"zip", "rar", "7z", "tar", "gz", "tgz", "bz2", "xz"} {
		if ext == e {
			return true
		}
	}
	return false
}

func isDocumentExtension(ext string) bool {
	for _, e := range []string{"pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx", "txt", "md", "csv", "json", "yaml", "yml", "xml"} {
		if ext == e {
			return true
		}
	}
	return false
}
