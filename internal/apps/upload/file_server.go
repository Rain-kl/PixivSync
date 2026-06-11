// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package upload

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Rain-kl/Wavelet/internal/apps/oauth"
	"github.com/Rain-kl/Wavelet/internal/common"
	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/diskcache"
	"github.com/Rain-kl/Wavelet/internal/logger"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/storage"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ServeFileByID 根据 ID 获取并提供已上传的文件
// @Summary 获取已上传文件
// @Description 根据文件 ID 获取并提供已上传的临时或正式文件，若配置了缓存则优先走本地缓存，否则从 S3 等后端存储读取并流式返回
// @Tags upload
// @Produce octet-stream
// @Param id path string true "文件 ID"
// @Param quality query string false "图片质量 (low, medium, high, origin)，默认为 origin"
// @Success 200 {file} file "成功获取文件内容"
// @Failure 400 {object} util.ResponseAny "文件 ID 格式错误"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Failure 404 {object} util.ResponseAny "文件未找到"
// @Failure 500 {object} util.ResponseAny "服务内部错误"
// @Router /f/{id} [get]
func ServeFileByID(c *gin.Context) {
	upload, err := getUploadRecordByID(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		if _, ok := err.(*strconv.NumError); ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid upload ID"})
			return
		}
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// 校验业务白名单与访问权限
	if err := checkFileAccessPermission(c, upload.Type); err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error_msg": common.UnAuthorized, "data": nil})
		return
	}

	ServeUpload(c, upload)
}

// getUploadRecordByID 从请求路径参数中解析文件 ID 并从数据库中检索处于 Pending 或 Used 状态的上传记录。
// 同时会自动设置通用的安全响应头。
func getUploadRecordByID(c *gin.Context) (*model.Upload, error) {
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Security-Policy", "sandbox")

	idStr := c.Param("id")
	uploadID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return nil, err
	}

	var upload model.Upload
	if err := db.DB(c.Request.Context()).
		Where("id = ? AND status IN (?, ?)", uploadID, model.UploadStatusPending, model.UploadStatusUsed).
		First(&upload).Error; err != nil {
		return nil, err
	}

	return &upload, nil
}

// ServeUpload 将已存在的文件内容读取并流式响应给客户端，支持本地和 S3/CDN 驱动，并可选支持 WebP 图片压缩与本地缓存。
func ServeUpload(c *gin.Context, upload *model.Upload) {
	quality := normalizeImageQuality(c.Query("quality"))
	isImage := strings.HasPrefix(strings.ToLower(upload.MimeType), "image/") || isImageExtension(strings.ToLower(upload.Extension))

	if quality == imageQualityOrigin || !isImage {
		serveOriginal(c, upload)
		return
	}

	cache := diskcache.GetGlobalCache()
	cacheKey := imageCompressionCacheKey(upload, quality)
	if webpBytes, err := cache.Get(cacheKey); err == nil {
		c.Data(http.StatusOK, "image/webp", webpBytes)
		return
	} else if !errors.Is(err, diskcache.ErrCacheMiss) {
		logger.WarnF(c.Request.Context(), "failed to read compressed image cache: %v", err)
	}

	// Cache miss: retrieve original file content
	origBytes, err := getOriginalFileBytes(c.Request.Context(), upload)
	if err != nil {
		logger.ErrorF(c.Request.Context(), "failed to retrieve original file bytes for compression: %v", err)
		serveOriginal(c, upload)
		return
	}

	// Compress to WebP
	webpBytes, err := CompressImageToWebP(bytes.NewReader(origBytes), quality)
	if err != nil {
		logger.ErrorF(c.Request.Context(), "failed to compress image to WebP: %v", err)
		serveOriginal(c, upload)
		return
	}

	if err := cache.Set(cacheKey, webpBytes, 0); err != nil {
		logger.WarnF(c.Request.Context(), "failed to cache compressed image: %v", err)
	}

	// Serve compressed WebP
	c.Data(http.StatusOK, "image/webp", webpBytes)
}

func imageCompressionCacheKey(upload *model.Upload, quality string) string {
	return fmt.Sprintf(
		"upload_webp_v1_%d_%d_%d_%s_%s",
		upload.ID,
		upload.UpdatedAt.UnixNano(),
		upload.FileSize,
		upload.Hash,
		quality,
	)
}

func normalizeImageQuality(quality string) string {
	switch strings.ToLower(quality) {
	case imageQualityLow, imageQualityMedium, imageQualityHigh:
		return strings.ToLower(quality)
	default:
		return imageQualityOrigin
	}
}

// serveOriginal 原始文件的流式响应逻辑
func serveOriginal(c *gin.Context, upload *model.Upload) {
	if upload.StorageDriver == storageDriverLocal || (upload.StorageDriver == "" && !storage.IsEnabled()) {
		c.File(upload.FilePath)
		return
	}

	// Retrieve file from S3 (via CDN if configured)
	obj, err := storage.GetObjectViaCache(c.Request.Context(), upload.FilePath)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// Cachefile
	if obj.CachePath != "" {
		c.File(obj.CachePath)
		return
	}

	// Stream from CDN/S3
	defer func() { _ = obj.Body.Close() }()

	// Respond with the file content
	c.DataFromReader(http.StatusOK, obj.ContentLength, obj.ContentType, obj.Body, nil)
}

// getOriginalFileBytes 获取原始文件所有字节
func getOriginalFileBytes(ctx context.Context, upload *model.Upload) ([]byte, error) {
	if upload.StorageDriver == storageDriverLocal || (upload.StorageDriver == "" && !storage.IsEnabled()) {
		return os.ReadFile(upload.FilePath)
	}

	// Retrieve file from S3 (via CDN if configured)
	obj, err := storage.GetObjectViaCache(ctx, upload.FilePath)
	if err != nil {
		return nil, err
	}

	// Cachefile
	if obj.CachePath != "" {
		return os.ReadFile(obj.CachePath)
	}

	defer func() { _ = obj.Body.Close() }()
	return io.ReadAll(obj.Body)
}

// checkFileAccessPermission 校验文件是否可以被当前请求访问
func checkFileAccessPermission(c *gin.Context, uploadType string) error {
	var sc model.SystemConfig
	var whitelist []string
	if err := sc.GetByKey(c.Request.Context(), model.ConfigKeyFileAccessWhitelist); err == nil && sc.Value != "" {
		if err := json.Unmarshal([]byte(sc.Value), &whitelist); err != nil {
			// 降级使用逗号分隔解析
			parts := strings.Split(sc.Value, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					whitelist = append(whitelist, p)
				}
			}
		}
	} else {
		// 默认兜底白名单为 avatar
		whitelist = []string{"avatar"}
	}

	inWhitelist := false
	for _, w := range whitelist {
		if strings.EqualFold(w, uploadType) {
			inWhitelist = true
			break
		}
	}

	if !inWhitelist {
		// 必须进行鉴权
		if _, err := oauth.GetUserFromRequest(c); err != nil {
			return err
		}
	}
	return nil
}
