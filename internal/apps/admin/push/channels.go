// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package push

import ("encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	pkgpush "github.com/Rain-kl/Wavelet/pkg/push"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Rain-kl/Wavelet/internal/common/response")

// ListChannelDefinitions 获取各种消息通道的表单配置定义列表
// @Summary 获取所有消息通道配置字段定义
// @Description 返回系统支持的所有消息通道类型（如飞书、邮件、自定义、Telegram）的动态表单定义，需要管理员权限
// @Tags admin-push
// @Produce json
// @Security SessionCookie
// @Success 200 {object} response.Any{data=[]Definition} "通道配置定义列表"
// @Router /api/v1/admin/push/channels/definitions [get]
func ListChannelDefinitions(c *gin.Context) {
	c.JSON(http.StatusOK, response.OK(ListDefinitions()))
}

// ListChannels 获取消息通道列表
// @Summary 获取所有消息通道
// @Description 返回系统配置的所有消息通道列表，需要管理员权限
// @Tags admin-push
// @Produce json
// @Security SessionCookie
// @Success 200 {object} response.Any{data=[]model.PushChannel} "消息通道列表"
// @Router /api/v1/admin/push/channels [get]
func ListChannels(c *gin.Context) {
	ctx := c.Request.Context()
	var channels []model.PushChannel
	if err := db.DB(ctx).Order("created_at DESC").Find(&channels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.OK(channels))
}

// CreateChannelRequest 创建通道参数
type CreateChannelRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Type        string `json:"type" binding:"required"`
	Token       string `json:"token"`
	URL         string `json:"url"`
	Other       string `json:"other"`
	Enabled     bool   `json:"enabled"`
}

// CreateChannel 创建消息通道
// @Summary 创建消息通道
// @Description 新建一个消息通道配置，需要管理员权限
// @Tags admin-push
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param request body CreateChannelRequest true "创建参数"
// @Success 200 {object} response.Any{data=model.PushChannel} "创建成功"
// @Router /api/v1/admin/push/channels [post]
func CreateChannel(c *gin.Context) {
	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Err(err.Error()))
		return
	}

	ctx := c.Request.Context()

	var count int64
	if err := db.DB(ctx).Model(&model.PushChannel{}).Where("name = ?", req.Name).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		return
	}
	if count > 0 {
		c.JSON(http.StatusBadRequest, response.Err("channel name already exists"))
		return
	}

	channel := model.PushChannel{
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Token:       req.Token,
		URL:         req.URL,
		Other:       req.Other,
		Enabled:     req.Enabled,
	}

	if err := channel.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, response.Err(err.Error()))
		return
	}

	if err := db.DB(ctx).Create(&channel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.OK(channel))
}

// UpdateChannelRequest 修改通道参数
type UpdateChannelRequest struct {
	Description string `json:"description"`
	Type        string `json:"type" binding:"required"`
	Token       string `json:"token"`
	URL         string `json:"url"`
	Other       string `json:"other"`
	Enabled     bool   `json:"enabled"`
}

// UpdateChannel 更新消息通道
// @Summary 更新消息通道
// @Description 修改消息通道配置，需要管理员权限
// @Tags admin-push
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param id path uint64 true "通道ID"
// @Param request body UpdateChannelRequest true "更新参数"
// @Success 200 {object} response.Any{data=model.PushChannel} "更新成功"
// @Router /api/v1/admin/push/channels/{id} [put]
func UpdateChannel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Err("invalid channel id"))
		return
	}

	var req UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Err(err.Error()))
		return
	}

	ctx := c.Request.Context()

	var channel model.PushChannel
	if err := db.DB(ctx).Where("id = ?", id).First(&channel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, response.Err("channel not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		return
	}

	channel.Description = req.Description
	channel.Type = req.Type
	channel.Token = req.Token
	channel.URL = req.URL
	channel.Other = req.Other
	channel.Enabled = req.Enabled

	if err := channel.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, response.Err(err.Error()))
		return
	}

	if err := db.DB(ctx).Save(&channel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.OK(channel))
}

// DeleteChannel 删除消息通道
// @Summary 删除消息通道
// @Description 根据ID删除消息通道，需要管理员权限
// @Tags admin-push
// @Produce json
// @Security SessionCookie
// @Param id path uint64 true "通道ID"
// @Success 200 {object} response.Any "删除成功"
// @Router /api/v1/admin/push/channels/{id} [delete]
func DeleteChannel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Err("invalid channel id"))
		return
	}

	ctx := c.Request.Context()
	var channel model.PushChannel
	if err := db.DB(ctx).Where("id = ?", id).First(&channel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, response.Err("channel not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		return
	}

	if err := db.DB(ctx).Delete(&channel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.OKNil())
}

// TestChannelRequest 测试通道连通性参数
type TestChannelRequest struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Token  string `json:"token"`
	URL    string `json:"url"`
	Other  string `json:"other"`
	Target string `json:"target"`
}

// TestChannel 测试通道连通性
// @Summary 测试通道连通性
// @Description 触发一次临时的或现有的通道连通性推送测试，需要管理员权限
// @Tags admin-push
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param request body TestChannelRequest true "测试参数"
// @Success 200 {object} response.Any "测试触发成功"
// @Router /api/v1/admin/push/channels/test [post]
func TestChannel(c *gin.Context) {
	var req TestChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Err(err.Error()))
		return
	}

	ctx := c.Request.Context()
	var url, token, other, channelType string

	if req.Name != "" {
		var channel model.PushChannel
		if err := db.DB(ctx).Where("name = ?", req.Name).First(&channel).Error; err != nil {
			c.JSON(http.StatusBadRequest, response.Err("channel not found"))
			return
		}
		url = channel.URL
		token = channel.Token
		other = channel.Other
		channelType = channel.Type
	} else {
		url = req.URL
		token = req.Token
		other = req.Other
		channelType = req.Type
	}

	// 对邮件类型应用全局配置作为回退
	if channelType == channelEmail {
		url, token, other = resolveSMTPConfig(ctx, url, token, other)
	}

	tempChannel := model.PushChannel{
		Name:    "test_temp",
		URL:     url,
		Token:   token,
		Other:   other,
		Type:    channelType,
		Enabled: true,
	}

	if err := tempChannel.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, response.Err(err.Error()))
		return
	}
	url = tempChannel.URL

	var config pkgpush.Config
	var renderedJSON string

	switch channelType {
	case channelLark:
		config = pkgpush.Config{
			Channel: channelLark,
			URL:     url,
			Secret:  token,
		}
		renderedJSON = other
	case channelEmail:
		config = pkgpush.Config{
			Channel: channelEmail,
			URL:     url,
			Key:     token,
			Secret:  other,
		}
	case channelTelegram:
		config = pkgpush.Config{
			Channel: channelTelegram,
			URL:     url,
			Secret:  token,
			Key:     other,
		}
	default:
		config = pkgpush.Config{
			Channel: channelCustom,
			URL:     url,
		}
		customPushReq := CustomPushRequest{
			Title:       "通道测试通知",
			Content:     "这是一条来自系统的消息通道连通性测试消息。",
			Description: "系统通道测试",
			URL:         "https://example.com",
			To:          req.Target,
		}
		renderedJSON = renderCustomPayload(other, customPushReq)
	}

	payload := SendPayload{
		EventKey: "test_channel",
		Config:   config,
		Target:   req.Target,
		Body: NotificationMessage{
			Title:   "通道测试通知",
			Content: "这是一条来自系统的消息通道连通性测试消息。",
			Level:   defaultLevelInfo,
		},
		Template: renderedJSON,
	}

	if err := enqueuePushTask(ctx, payload); err != nil {
		c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.OKNil())
}

// CustomPushRequest 外部公开推送请求参数
type CustomPushRequest struct {
	Title       string `json:"title" form:"title"`
	Description string `json:"description" form:"description"`
	Content     string `json:"content" form:"content"`
	URL         string `json:"url" form:"url"`
	To          string `json:"to" form:"to"`
	Token       string `json:"token" form:"token"`
}

func escapeJSONString(s string) string {
	b, _ := json.Marshal(s)
	const minJSONLen = 2
	if len(b) >= minJSONLen {
		return string(b[1 : len(b)-1])
	}
	return s
}

func renderCustomPayload(template string, req CustomPushRequest) string {
	result := template
	result = strings.ReplaceAll(result, "$title", escapeJSONString(req.Title))
	result = strings.ReplaceAll(result, "$description", escapeJSONString(req.Description))
	result = strings.ReplaceAll(result, "$content", escapeJSONString(req.Content))
	result = strings.ReplaceAll(result, "$url", escapeJSONString(req.URL))
	result = strings.ReplaceAll(result, "$to", escapeJSONString(req.To))
	return result
}
