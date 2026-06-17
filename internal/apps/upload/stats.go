// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package upload

import (
	"net/http"
	"strings"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/gin-gonic/gin"

	"github.com/Rain-kl/Wavelet/internal/common/response"
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
// @Success 200 {object} response.Any{data=fileStatsResponse} "获取成功"
// @Failure 401 {object} response.Any "未登录"
// @Failure 403 {object} response.Any "无管理员权限"
// @Failure 500 {object} response.Any "内部错误"
// @Router /api/v1/admin/uploads/stats [get]
func GetFileStats(c *gin.Context) {
	ctx := c.Request.Context()

	var stats []model.UploadStat
	if err := db.DB(ctx).Find(&stats).Error; err != nil {
		c.JSON(http.StatusOK, response.Err(err.Error()))
		return
	}

	now := time.Now()
	trendDates := make([]string, 0, fileStatsTrendDays)
	trendCountMap := make(map[string]int64, fileStatsTrendDays)
	trendSizeMap := make(map[string]int64, fileStatsTrendDays)
	for i := fileStatsTrendDays - 1; i >= 0; i-- {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		trendDates = append(trendDates, date)
		trendCountMap[date] = 0
		trendSizeMap[date] = 0
	}

	var (
		totalCount int64
		totalSize  int64
		types      []distributionItem
		categories []distributionItem
	)

	categoriesList := []string{catImage, catVideo, catAudio, catDocument, catArchive, catOther}
	categoryMap := make(map[string]distributionItem, len(categoriesList))
	for _, cat := range categoriesList {
		categoryMap[cat] = distributionItem{Name: cat}
	}

	for _, stat := range stats {
		switch stat.Dimension {
		case model.UploadStatDimensionTotal:
			totalCount = stat.FileCount
			totalSize = stat.FileSize
		case model.UploadStatDimensionType:
			types = append(types, distributionItem{
				Name:  stat.StatKey,
				Count: stat.FileCount,
				Size:  stat.FileSize,
			})
		case model.UploadStatDimensionCategory:
			if item, ok := categoryMap[stat.StatKey]; ok {
				item.Count = stat.FileCount
				item.Size = stat.FileSize
				categoryMap[stat.StatKey] = item
			}
		case model.UploadStatDimensionTrend:
			if _, ok := trendCountMap[stat.StatKey]; ok {
				trendCountMap[stat.StatKey] = stat.FileCount
				trendSizeMap[stat.StatKey] = stat.FileSize
			}
		}
	}

	categories = make([]distributionItem, 0, len(categoriesList))
	for _, cat := range categoriesList {
		categories = append(categories, categoryMap[cat])
	}

	trend := make([]trendItem, 0, len(trendDates))
	for _, date := range trendDates {
		trend = append(trend, trendItem{
			Date:  date,
			Count: trendCountMap[date],
			Size:  trendSizeMap[date],
		})
	}

	c.JSON(http.StatusOK, response.OK(fileStatsResponse{
		TotalCount: totalCount,
		TotalSize:  totalSize,
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
