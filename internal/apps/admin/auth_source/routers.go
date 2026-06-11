// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

// Package auth_source 提供认证源管理功能
package auth_source

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/Rain-kl/Wavelet/internal/apps/admin"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/util"
	"github.com/gin-gonic/gin"
)

// AuthSourceRequest 创建或更新认证源的请求参数
type AuthSourceRequest struct {
	Name               string `json:"name"`
	Type               string `json:"type"`
	DisplayName        string `json:"display_name"`
	IsActive           bool   `json:"is_active"`
	ClientID           string `json:"client_id"`
	ClientSecret       string `json:"client_secret"`
	OpenIDDiscoveryURL string `json:"openid_discovery_url"`
	Scopes             string `json:"scopes"`
	IconURL            string `json:"icon_url"`
}

// ToggleAuthSourceRequest 切换认证源启用状态的请求参数
type ToggleAuthSourceRequest struct {
	IsActive bool `json:"is_active"`
}

// ListAuthSources 获取认证源列表
// @Summary 获取认证源列表
// @Description 返回所有已配置的 OAuth/OIDC 认证源列表，包括已启用和未启用的，需要管理员权限
// @Tags admin
// @Produce json
// @Security SessionCookie
// @Success 200 {object} util.ResponseAny{data=[]model.AuthSource} "认证源列表"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Failure 403 {object} util.ResponseAny "无管理员权限"
// @Failure 500 {object} util.ResponseAny "内部错误"
// @Router /api/v1/admin/auth-sources [get]
func ListAuthSources(c *gin.Context) {
	sources, err := model.GetAuthSources()
	if err != nil {
		c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
		return
	}
	c.JSON(http.StatusOK, util.OK(sources))
}

// CreateAuthSource 创建认证源
// @Summary 创建认证源
// @Description 创建一个新的 OAuth/OIDC 认证源配置，认证源名称必须唯一且符合命名规范，需要管理员权限
// @Tags admin
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param request body auth_source.AuthSourceRequest true "创建认证源参数"
// @Success 200 {object} util.ResponseAny{data=model.AuthSource} "创建成功，返回认证源信息"
// @Failure 400 {object} util.ResponseAny "参数错误或验证失败"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Failure 403 {object} util.ResponseAny "无管理员权限"
// @Router /api/v1/admin/auth-sources [post]
func CreateAuthSource(c *gin.Context) {
	var req AuthSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}

	source := model.AuthSource{
		Name:               req.Name,
		Type:               req.Type,
		DisplayName:        req.DisplayName,
		IsActive:           req.IsActive,
		ClientID:           req.ClientID,
		ClientSecret:       req.ClientSecret,
		OpenIDDiscoveryURL: req.OpenIDDiscoveryURL,
		Scopes:             req.Scopes,
		IconURL:            req.IconURL,
	}
	if err := model.CreateAuthSource(&source); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}
	source.Sanitize()
	c.JSON(http.StatusOK, util.OK(source))
}

// UpdateAuthSource 更新认证源
// @Summary 更新认证源
// @Description 更新指定 ID 的认证源配置。若 client_secret 字段为空，则保留原有密钥不变，需要管理员权限
// @Tags admin
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param id path uint64 true "认证源 ID 或名称"
// @Param request body auth_source.AuthSourceRequest true "更新认证源参数"
// @Success 200 {object} util.ResponseAny{data=model.AuthSource} "更新成功，返回更新后的认证源信息"
// @Failure 400 {object} util.ResponseAny "参数错误或验证失败"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Failure 403 {object} util.ResponseAny "无管理员权限"
// @Failure 500 {object} util.ResponseAny "内部错误"
// @Router /api/v1/admin/auth-sources/{id} [put]
func UpdateAuthSource(c *gin.Context) {
	id, err := parseSourceID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}

	var req AuthSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}

	source := model.AuthSource{
		ID:                 id,
		Name:               req.Name,
		Type:               req.Type,
		DisplayName:        req.DisplayName,
		IsActive:           req.IsActive,
		ClientID:           req.ClientID,
		ClientSecret:       req.ClientSecret,
		OpenIDDiscoveryURL: req.OpenIDDiscoveryURL,
		Scopes:             req.Scopes,
		IconURL:            req.IconURL,
	}
	keepSecret := source.ClientSecret == ""
	if err := model.UpdateAuthSource(&source, keepSecret); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}
	updated, err := model.GetAuthSourceByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
		return
	}
	updated.Sanitize()
	c.JSON(http.StatusOK, util.OK(updated))
}

// ToggleAuthSource 切换认证源启用状态
// @Summary 切换认证源启用状态
// @Description 启用或禁用指定认证源。尝试启用时将验证 Client ID 和 Client Secret 是否已配置，需要管理员权限
// @Tags admin
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param id path uint64 true "认证源 ID 或名称"
// @Param request body auth_source.ToggleAuthSourceRequest true "启用状态"
// @Success 200 {object} util.ResponseAny{data=string} "切换成功"
// @Failure 400 {object} util.ResponseAny "验证失败或认证源不存在"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Failure 403 {object} util.ResponseAny "无管理员权限"
// @Router /api/v1/admin/auth-sources/{id}/toggle [put]
func ToggleAuthSource(c *gin.Context) {
	id, err := parseSourceID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}

	var req ToggleAuthSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}

	if err := model.ToggleAuthSource(id, req.IsActive); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}
	c.JSON(http.StatusOK, util.OKNil())
}

// DeleteAuthSource 删除认证源
// @Summary 删除认证源
// @Description 删除指定认证源及其关联的所有外部帐号绑定记录，警告：删除后相关用户将无法通过该源登录，需要管理员权限
// @Tags admin
// @Produce json
// @Security SessionCookie
// @Param id path uint64 true "认证源 ID 或名称"
// @Success 200 {object} util.ResponseAny{data=string} "删除成功"
// @Failure 400 {object} util.ResponseAny "ID 无效或删除失败"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Failure 403 {object} util.ResponseAny "无管理员权限"
// @Router /api/v1/admin/auth-sources/{id} [delete]
func DeleteAuthSource(c *gin.Context) {
	id, err := parseSourceID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}
	if err := model.DeleteAuthSource(id); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}
	c.JSON(http.StatusOK, util.OKNil())
}

func parseSourceID(c *gin.Context) (uint64, error) {
	raw := c.Param("id")
	if raw == "" {
		return 0, errors.New(admin.InvalidAuthSourceID)
	}
	source, err := model.GetAuthSourceByName(raw)
	if err == nil {
		return source.ID, nil
	}
	var id uint64
	if _, scanErr := fmt.Sscanf(raw, "%d", &id); scanErr != nil || id == 0 {
		return 0, errors.New(admin.InvalidAuthSourceID)
	}
	return id, nil
}
