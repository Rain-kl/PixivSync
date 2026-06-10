// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package user 提供用户认证与帐户管理功能
package user

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Rain-kl/Wavelet/internal/apps/oauth"
	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/util"
	"github.com/gin-gonic/gin"
)

type createTokenRequest struct {
	Name string `json:"name"`
}

type tokenResponse struct {
	Token  string            `json:"token"`
	Record model.AccessToken `json:"record"`
}

// ListAccessTokens 获取当前用户的 AccessToken 列表
// @Summary 获取当前用户的 AccessToken 列表
// @Description 返回当前登录用户的所有 active access tokens（脱敏后）
// @Tags user
// @Produce json
// @Security SessionCookie
// @Success 200 {object} util.ResponseAny{data=[]model.AccessToken} "令牌列表"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Router /api/v1/user/access-tokens [get]
// ListAccessTokens 获取当前用户的 AccessToken 列表
func ListAccessTokens(c *gin.Context) {
	currUser, _ := util.GetFromContext[*model.User](c, oauth.UserObjKey)
	ctx := c.Request.Context()

	var tokens []model.AccessToken
	if err := db.DB(ctx).Where("user_id = ?", currUser.ID).Order("created_at desc").Find(&tokens).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(err.Error()))
		return
	}

	c.JSON(http.StatusOK, util.OK(tokens))
}

// CreateAccessToken 创建一个新的 AccessToken
// @Summary 创建一个新的 AccessToken
// @Description 为当前用户新建一个 API 访问令牌，仅在此接口返回一次明文令牌值，请妥善保存。
// @Tags user
// @Accept json
// @Produce json
// @Param request body user.createTokenRequest true "令牌名称"
// @Security SessionCookie
// @Success 200 {object} util.ResponseAny{data=user.tokenResponse} "新建令牌成功"
// @Failure 400 {object} util.ResponseAny "参数错误或超限"
// @Router /api/v1/user/access-tokens [post]
func CreateAccessToken(c *gin.Context) {
	currUser, _ := util.GetFromContext[*model.User](c, oauth.UserObjKey)
	ctx := c.Request.Context()

	var req createTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, util.Err(errBindParamsFailed))
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		c.JSON(http.StatusOK, util.Err(errTokenNameRequired))
		return
	}

	// 检查最大限制（基于 ConfigKeyMaxAPIKeysPerUser 配置，默认值为 5）
	maxLimit := 5
	if val, err := model.GetIntByKey(ctx, model.ConfigKeyMaxAPIKeysPerUser); err == nil {
		maxLimit = val
	}

	var count int64
	if err := db.DB(ctx).Model(&model.AccessToken{}).Where("user_id = ?", currUser.ID).Count(&count).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(err.Error()))
		return
	}

	if int(count) >= maxLimit {
		c.JSON(http.StatusOK, util.Err(errAccessTokenLimitReached))
		return
	}

	// 生成 Token
	tokenStr, err := model.GenerateTokenString()
	if err != nil {
		c.JSON(http.StatusOK, util.Err(errGenerateTokenFailed))
		return
	}

	tokenHash := model.HashToken(tokenStr)
	maskedToken := model.MaskTokenString(tokenStr)

	tokenRecord := model.AccessToken{
		UserID:      currUser.ID,
		Name:        req.Name,
		TokenHash:   tokenHash,
		MaskedToken: maskedToken,
	}

	if err := db.DB(ctx).Create(&tokenRecord).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(err.Error()))
		return
	}

	c.JSON(http.StatusOK, util.OK(tokenResponse{
		Token:  tokenStr,
		Record: tokenRecord,
	}))
}

// DeleteAccessToken 删除一个 AccessToken
// @Summary 删除一个 AccessToken
// @Description 撤销并删除一个属于当前用户的 API 访问令牌
// @Tags user
// @Produce json
// @Param id path string true "令牌ID"
// @Security SessionCookie
// @Success 200 {object} util.ResponseAny{data=string} "删除成功"
// @Failure 400 {object} util.ResponseAny "参数错误"
// @Router /api/v1/user/access-tokens/{id} [delete]
func DeleteAccessToken(c *gin.Context) {
	currUser, _ := util.GetFromContext[*model.User](c, oauth.UserObjKey)
	ctx := c.Request.Context()

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, util.Err(errInvalidTokenID))
		return
	}

	tx := db.DB(ctx).Where("id = ? AND user_id = ?", id, currUser.ID).Delete(&model.AccessToken{})
	if tx.Error != nil {
		c.JSON(http.StatusOK, util.Err(tx.Error.Error()))
		return
	}

	if tx.RowsAffected == 0 {
		c.JSON(http.StatusOK, util.Err(errTokenNotFoundOrForbidden))
		return
	}

	c.JSON(http.StatusOK, util.OK("删除成功"))
}

// RotateAccessToken 轮换一个 AccessToken
// @Summary 轮换一个 AccessToken
// @Description 轮换（重新生成）一个属于当前用户的 API 访问令牌的密钥，旧令牌将立即失效
// @Tags user
// @Produce json
// @Param id path string true "令牌ID"
// @Security SessionCookie
// @Success 200 {object} util.ResponseAny{data=user.tokenResponse} "令牌轮换成功"
// @Failure 400 {object} util.ResponseAny "参数错误"
// @Router /api/v1/user/access-tokens/{id}/rotate [post]
func RotateAccessToken(c *gin.Context) {
	currUser, _ := util.GetFromContext[*model.User](c, oauth.UserObjKey)
	ctx := c.Request.Context()

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, util.Err(errInvalidTokenID))
		return
	}

	var tokenRecord model.AccessToken
	if err := db.DB(ctx).Where("id = ? AND user_id = ?", id, currUser.ID).First(&tokenRecord).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(errTokenNotFoundOrForbidden))
		return
	}

	// 生成新的 Token
	newTokenStr, err := model.GenerateTokenString()
	if err != nil {
		c.JSON(http.StatusOK, util.Err(errGenerateTokenFailed))
		return
	}

	newTokenHash := model.HashToken(newTokenStr)
	newMaskedToken := model.MaskTokenString(newTokenStr)

	tokenRecord.TokenHash = newTokenHash
	tokenRecord.MaskedToken = newMaskedToken
	tokenRecord.LastUsedAt = nil // 轮换后重置使用时间

	if err := db.DB(ctx).Save(&tokenRecord).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(err.Error()))
		return
	}

	c.JSON(http.StatusOK, util.OK(tokenResponse{
		Token:  newTokenStr,
		Record: tokenRecord,
	}))
}
