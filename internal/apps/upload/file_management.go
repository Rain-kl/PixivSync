// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package upload

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/Rain-kl/Wavelet/internal/apps/oauth"
	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/util"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type listMyFilesRequest struct {
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
	Keyword   string `form:"keyword"`
	Type      string `form:"type"`
	Extension string `form:"extension"`
}

type listMyFilesResponse struct {
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
	Items    []model.Upload `json:"items"`
}

// ListMyFiles 获取当前用户上传的文件列表
// @Summary 获取我的文件列表
// @Description 分页获取当前登录用户上传的文件，支持文件名关键词、业务类型、扩展名过滤
// @Tags upload
// @Produce json
// @Param page query int false "页码（默认 1）"
// @Param page_size query int false "每页数量（默认 20，最大 100）"
// @Param keyword query string false "文件名关键词（模糊匹配）"
// @Param type query string false "业务分类过滤"
// @Param extension query string false "扩展名过滤"
// @Security SessionCookie
// @Success 200 {object} util.ResponseAny{data=listMyFilesResponse} "查询成功"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Router /api/v1/upload/my [get]
func ListMyFiles(c *gin.Context) {
	currUser, _ := util.GetFromContext[*model.User](c, oauth.UserObjKey)
	ctx := c.Request.Context()

	var req listMyFilesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusOK, util.Err(ErrInvalidParams))
		return
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}

	query := db.DB(ctx).Model(&model.Upload{}).
		Where("user_id = ? AND status != ?", currUser.ID, model.UploadStatusDeleted)

	if req.Keyword != "" {
		query = query.Where("LOWER(file_name) LIKE ?", "%"+strings.ToLower(req.Keyword)+"%")
	}
	if req.Type != "" {
		query = query.Where("type = ?", req.Type)
	}
	if req.Extension != "" {
		query = query.Where("extension = ?", strings.ToLower(req.Extension))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(ErrQueryFileCountFailed))
		return
	}

	var items []model.Upload
	offset := (req.Page - 1) * req.PageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(req.PageSize).Find(&items).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(ErrQueryFileListFailed))
		return
	}

	c.JSON(http.StatusOK, util.OK(listMyFilesResponse{
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
		Items:    items,
	}))
}

// DeleteFile 软删除文件记录
// @Summary 删除文件
// @Description 将文件状态置为 deleted（软删除），不会立即清理底层存储对象
// @Tags upload
// @Produce json
// @Param id path string true "文件 ID"
// @Security SessionCookie
// @Success 200 {object} util.ResponseAny "删除成功"
// @Failure 403 {object} util.ResponseAny "无权操作"
// @Failure 404 {object} util.ResponseAny "文件不存在"
// @Router /api/v1/upload/{id} [delete]
func DeleteFile(c *gin.Context) {
	currUser, _ := util.GetFromContext[*model.User](c, oauth.UserObjKey)
	ctx := c.Request.Context()
	if storageReadOnly(ctx) {
		c.JSON(http.StatusConflict, util.Err(ErrStorageReadOnly))
		return
	}

	uploadID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, util.Err(ErrInvalidFileID))
		return
	}

	var upload model.Upload
	if err := db.DB(ctx).Where("id = ? AND status != ?", uploadID, model.UploadStatusDeleted).First(&upload).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.JSON(http.StatusOK, util.Err(ErrQueryUploadRecordFailed))
		return
	}
	if upload.UserID != currUser.ID && !currUser.IsAdmin {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	if err := db.DB(ctx).Model(&upload).Update("status", model.UploadStatusDeleted).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(ErrDeleteFileFailed))
		return
	}
	c.JSON(http.StatusOK, util.OKNil())
}

// GetDistinctUploadTypes 获取数据库中所有已存在的文件业务类型
// @Summary 获取文件业务类型列表
// @Description 返回数据库中所有已上传文件实际拥有的业务类型列表
// @Tags admin
// @Produce json
// @Security SessionCookie
// @Success 200 {object} util.ResponseAny{data=[]string} "业务类型列表"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Failure 403 {object} util.ResponseAny "无管理员权限"
// @Failure 500 {object} util.ResponseAny "内部错误"
// @Router /api/v1/admin/uploads/types [get]
func GetDistinctUploadTypes(c *gin.Context) {
	var dbTypes []string
	if err := db.DB(c.Request.Context()).Model(&model.Upload{}).
		Where("type IS NOT NULL AND type != ''").
		Distinct().
		Pluck("type", &dbTypes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
		return
	}
	sort.Strings(dbTypes)
	c.JSON(http.StatusOK, util.OK(dbTypes))
}
