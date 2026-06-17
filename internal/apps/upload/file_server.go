// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package upload

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Rain-kl/Wavelet/internal/apps/oauth"
	"github.com/Rain-kl/Wavelet/internal/common"
	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/diskcache"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/util"
	"github.com/Rain-kl/Wavelet/pkg/logger"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

var compressedImageFlight singleflight.Group

type compressedImageCacheResult struct {
	bytes  []byte
	cached bool
	err    error
}

// ServeFileByID 根据 ID 获取并提供已上传的文件
// @Summary 获取已上传文件
// @Description 根据文件 ID 获取并提供已上传的临时或正式文件，若配置了缓存则优先走本地缓存，否则从 S3 等后端存储读取并流式返回
// @Tags upload
// @Produce octet-stream
// @Param id path string true "文件 ID"
// @Param quality query string false "图片质量 (low, medium, high, origin)，默认为 origin"
// @Success 200 {file} file "成功获取文件内容"
// @Failure 400 {object} response.Any "文件 ID 格式错误"
// @Failure 401 {object} response.Any "未登录"
// @Failure 404 {object} response.Any "文件未找到"
// @Failure 500 {object} response.Any "服务内部错误"
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
	if err := checkFileAccessPermission(c, upload); err != nil {
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

// fileTypeCategory 定义文件的大类，便于未来扩展不同的处理方式
type fileTypeCategory string

const (
	fileTypeImage fileTypeCategory = "image"
	fileTypeVideo fileTypeCategory = "video"
	fileTypeAudio fileTypeCategory = "audio"
	fileTypeOther fileTypeCategory = "other"
)

// getFileTypeCategory 判断并返回文件的大类
func getFileTypeCategory(upload *model.Upload) fileTypeCategory {
	mime := strings.ToLower(upload.MimeType)
	ext := strings.ToLower(upload.Extension)

	if strings.HasPrefix(mime, "image/") || isImageExtension(ext) {
		return fileTypeImage
	}
	if strings.HasPrefix(mime, "video/") {
		return fileTypeVideo
	}
	if strings.HasPrefix(mime, "audio/") {
		return fileTypeAudio
	}
	return fileTypeOther
}

// ServeUpload 将已存在的文件内容读取并流式响应给客户端，支持本地和 S3/CDN 驱动，并可选支持 WebP 图片压缩与本地缓存。
func ServeUpload(c *gin.Context, upload *model.Upload) {
	// 设置通用的缓存控制响应头
	setCacheHeaders(c, upload)

	category := getFileTypeCategory(upload)
	quality := normalizeImageQuality(c.Query("quality"))

	switch category {
	case fileTypeImage:
		// 如果是图片且不是原图质量，则提供压缩优化后的图片预览
		if quality != imageQualityOrigin {
			serveCompressedImage(c, upload, quality)
			return
		}
		// 请求原图质量时，退化到默认提供原文件
		fallthrough

	default:
		// 默认提供原文件，并执行协商缓存校验
		serveOriginalWithConditionalCheck(c, upload)
	}
}

func setCacheHeaders(c *gin.Context, upload *model.Upload) {
	if isFilePublic(c.Request.Context(), upload.Type) {
		c.Header("Cache-Control", "public, max-age=31536000")
	} else {
		c.Header("Cache-Control", "private, no-cache")
	}
}

func serveOriginalWithConditionalCheck(c *gin.Context, upload *model.Upload) {
	etag := fmt.Sprintf(`W/"%s"`, upload.Hash)
	c.Header("ETag", etag)

	if c.GetHeader("If-None-Match") == etag {
		c.AbortWithStatus(http.StatusNotModified)
		return
	}

	serveOriginal(c, upload)
}

func serveCompressedImage(c *gin.Context, upload *model.Upload, quality string) {
	etag := fmt.Sprintf(`W/"%s-%s"`, upload.Hash, quality)
	c.Header("ETag", etag)

	if c.GetHeader("If-None-Match") == etag {
		c.AbortWithStatus(http.StatusNotModified)
		return
	}

	webpBytes, _, err := ensureCompressedImageCache(c.Request.Context(), upload, quality)
	if err != nil {
		if len(webpBytes) > 0 {
			logger.WarnF(c.Request.Context(), "failed to cache compressed image: %v", err)
			c.Data(http.StatusOK, "image/webp", webpBytes)
			return
		}
		logger.ErrorF(c.Request.Context(), "failed to prepare compressed image cache: %v", err)
		serveOriginal(c, upload)
		return
	}

	c.Data(http.StatusOK, "image/webp", webpBytes)
}

func ensureCompressedImageCache(
	ctx context.Context,
	upload *model.Upload,
	quality string,
) ([]byte, bool, error) {
	cache := diskcache.GetGlobalCache()
	cacheKey := imageCompressionCacheKey(upload, quality)
	webpBytes, err := cache.Get(cacheKey)
	if err == nil {
		return webpBytes, true, nil
	}
	if !errors.Is(err, diskcache.ErrCacheMiss) {
		return nil, false, fmt.Errorf("read compressed image cache: %w", err)
	}

	result, err, _ := compressedImageFlight.Do(cacheKey, func() (any, error) {
		return generateCompressedImageCache(ctx, upload, quality, cacheKey)
	})
	if err != nil {
		return nil, false, err
	}

	res := result.(compressedImageCacheResult)
	return res.bytes, res.cached, res.err
}

func generateCompressedImageCache(
	ctx context.Context,
	upload *model.Upload,
	quality string,
	cacheKey string,
) (compressedImageCacheResult, error) {
	cache := diskcache.GetGlobalCache()

	webpBytes, err := cache.Get(cacheKey)
	if err == nil {
		return compressedImageCacheResult{bytes: webpBytes, cached: true}, nil
	}
	if !errors.Is(err, diskcache.ErrCacheMiss) {
		return compressedImageCacheResult{}, fmt.Errorf("read compressed image cache: %w", err)
	}

	origBytes, err := getOriginalFileBytes(ctx, upload)
	if err != nil {
		return compressedImageCacheResult{}, fmt.Errorf("read original image: %w", err)
	}

	webpBytes, err = CompressImageToWebP(bytes.NewReader(origBytes), quality)
	if err != nil {
		return compressedImageCacheResult{}, fmt.Errorf("compress image to WebP: %w", err)
	}

	if err := cache.Set(cacheKey, webpBytes, diskcache.NoExpiration); err != nil {
		return compressedImageCacheResult{
			bytes: webpBytes,
			err:   fmt.Errorf("write compressed image cache: %w", err),
		}, nil
	}

	return compressedImageCacheResult{bytes: webpBytes}, nil
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
	obj, err := openStoredObject(c.Request.Context(), upload)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	defer func() { _ = obj.Body.Close() }()
	c.DataFromReader(http.StatusOK, obj.ContentLength, obj.ContentType, obj.Body, nil)
}

// getOriginalFileBytes 获取原始文件所有字节
func getOriginalFileBytes(ctx context.Context, upload *model.Upload) ([]byte, error) {
	obj, err := openStoredObject(ctx, upload)
	if err != nil {
		return nil, err
	}
	defer func() { _ = obj.Body.Close() }()
	return io.ReadAll(obj.Body)
}

// isFilePublic 校验文件类型是否在公开访问白名单中
func isFilePublic(ctx context.Context, uploadType string) bool {
	whitelist := loadFileAccessWhitelist(ctx)
	_, ok := whitelist[strings.ToLower(uploadType)]
	return ok
}

func checkPrivateFileOwner(c *gin.Context, ownerID uint64) error {
	var currUser *model.User
	var err error
	if u, ok := util.GetFromContext[*model.User](c, oauth.UserObjKey); ok && u != nil {
		currUser = u
	} else {
		currUser, err = oauth.GetUserFromRequest(c)
		if err != nil {
			return err
		}
	}
	if currUser.IsAdmin {
		return nil
	}
	if currUser.ID != ownerID {
		return errors.New("forbidden: cross-user access denied")
	}
	return nil
}

// checkFileAccessPermission 校验文件是否可以被当前请求访问
func checkFileAccessPermission(c *gin.Context, upload *model.Upload) error {
	// 1. 私有文件校验（优先级高于当前白名单逻辑）
	if upload.AccessMode == 0 {
		return checkPrivateFileOwner(c, upload.UserID)
	}

	// 2. 如果类型为公开的则再进行校验白名单
	if !isFilePublic(c.Request.Context(), upload.Type) {
		// 必须进行鉴权
		if _, ok := util.GetFromContext[*model.User](c, oauth.UserObjKey); !ok {
			if _, err := oauth.GetUserFromRequest(c); err != nil {
				return err
			}
		}
	}
	return nil
}
