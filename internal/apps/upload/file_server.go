// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package upload

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Rain-kl/Wavelet/internal/apps/oauth"
	"github.com/Rain-kl/Wavelet/internal/common"
	"github.com/Rain-kl/Wavelet/internal/db"
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
// @Success 200 {file} file "成功获取文件内容"
// @Failure 400 {object} util.ResponseAny "文件 ID 格式错误"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Failure 404 {object} util.ResponseAny "文件未找到"
// @Failure 500 {object} util.ResponseAny "服务内部错误"
// @Router /f/{id} [get]
func ServeFileByID(c *gin.Context) {
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Security-Policy", "sandbox")

	idStr := c.Param("id")
	uploadID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid upload ID"})
		return
	}

	var upload model.Upload
	if err := db.DB(c.Request.Context()).
		Where("id = ? AND status IN (?, ?)", uploadID, model.UploadStatusPending, model.UploadStatusUsed).
		First(&upload).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.AbortWithStatus(http.StatusNotFound)
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

	if upload.StorageDriver == "local" || (upload.StorageDriver == "" && !storage.IsEnabled()) {
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
