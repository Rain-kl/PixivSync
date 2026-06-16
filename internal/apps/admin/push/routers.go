// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package push defines push notification HTTP routes.
package push

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/task"
	"github.com/Rain-kl/Wavelet/pkg/push"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Rain-kl/Wavelet/internal/common/response"
)

// UpdateEventRequest 更新事件请求参数
type UpdateEventRequest struct {
	Channels []string `json:"channels"`
	Targets  []string `json:"targets"`
	Template string   `json:"template" binding:"required"`
	Enabled  bool     `json:"enabled"`
}

// TestPushRequest 测试推送通道请求参数
type TestPushRequest struct {
	Config push.Config `json:"config" binding:"required"`
	Target string      `json:"target"`
}

// SyncEvents automatically registers/updates built-in events in the database.
func SyncEvents(ctx context.Context) error {
	for _, meta := range BuiltInEvents {
		var event model.PushEvent
		err := db.DB(ctx).Where("event_key = ?", meta.Key).First(&event).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			var defaultTemplateStr string
			if defaultTemplateBytes, err := json.Marshal(meta.DefaultTemplate); err == nil {
				defaultTemplateStr = string(defaultTemplateBytes)
			}
			event = model.PushEvent{
				EventKey: meta.Key,
				Name:     meta.Name,
				Channels: []string{},
				Targets:  []string{},
				Template: defaultTemplateStr,
				Enabled:  false,
			}
			if err := db.DB(ctx).Create(&event).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}
	return nil
}

// ListEvents 获取通知事件列表
// @Summary 获取所有通知事件
// @Description 返回系统配置的通知事件列表，包括预置和自定义事件，需要管理员权限
// @Tags admin-push
// @Produce json
// @Security SessionCookie
// @Success 200 {object} response.Any{data=[]model.PushEvent} "通知事件列表"
// @Router /api/v1/admin/push/events [get]
func ListEvents(c *gin.Context) {
	ctx := c.Request.Context()

	var events []model.PushEvent
	if err := db.DB(ctx).Order("created_at DESC").Find(&events).Error; err != nil {
		c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.OK(events))
}

// CreateEventRequest 创建事件请求参数
type CreateEventRequest struct {
	EventKey string   `json:"event_key"`
	TaskType string   `json:"task_type"` // 关联的异步任务类型
	Channels []string `json:"channels"`
	Targets  []string `json:"targets"`
	Template string   `json:"template"`
	Enabled  bool     `json:"enabled"`
}

func findBuiltInEvent(key string) (EventMetadata, bool) {
	for _, meta := range BuiltInEvents {
		if meta.Key == key {
			return meta, true
		}
	}
	return EventMetadata{}, false
}

// ListBuiltInEvents 获取内置通知事件列表
// @Summary 获取所有内置通知事件
// @Description 返回系统定义的所有内置通知事件元数据，供前端下拉框选择，需要管理员权限
// @Tags admin-push
// @Produce json
// @Security SessionCookie
// @Success 200 {object} response.Any{data=[]EventMetadata} "内置通知事件列表"
// @Router /api/v1/admin/push/events/builtin [get]
func ListBuiltInEvents(c *gin.Context) {
	c.JSON(http.StatusOK, response.OK(BuiltInEvents))
}

func getEventInfo(req CreateEventRequest) (string, string, []byte, error) {
	if req.TaskType != "" {
		// 1. 检查关联任务是否存在
		meta := task.GetTaskMetaByAsynqTask(req.TaskType)
		if meta == nil {
			return "", "", nil, errors.New("unsupported task type")
		}
		eventKey := "task_completed:" + req.TaskType
		eventName := "任务完成: " + meta.Name

		defaultTemplate := NotificationMessage{
			Title:   "任务完成: " + meta.Name,
			Content: "异步任务 {{task_name}} (ID: {{task_id}}) 已完成。状态: {{task_status}}，耗时: {{task_duration}} ms。",
			Level:   defaultLevelInfo,
		}
		defaultTemplateBytes, err := json.Marshal(defaultTemplate)
		if err != nil {
			return "", "", nil, err
		}
		return eventKey, eventName, defaultTemplateBytes, nil
	}

	if req.EventKey == "" {
		return "", "", nil, errors.New("either event_key or task_type must be provided")
	}

	// 1. 检查内置事件是否存在
	meta, found := findBuiltInEvent(req.EventKey)
	if !found {
		return "", "", nil, errors.New("unsupported built-in event key")
	}

	defaultTemplateBytes, err := json.Marshal(meta.DefaultTemplate)
	if err != nil {
		return "", "", nil, err
	}

	return req.EventKey, meta.Name, defaultTemplateBytes, nil
}

// CreateEvent 创建通知事件
// @Summary 创建通知事件
// @Description 绑定系统内置事件或异步任务、推送渠道、接收目标并创建通知事件配置，需要管理员权限
// @Tags admin-push
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param request body CreateEventRequest true "创建参数"
// @Success 200 {object} response.Any{data=model.PushEvent} "创建成功"
// @Router /api/v1/admin/push/events [post]
func CreateEvent(c *gin.Context) {
	var req CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Err(err.Error()))
		return
	}

	ctx := c.Request.Context()

	eventKey, eventName, defaultTemplateBytes, err := getEventInfo(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Err(err.Error()))
		return
	}

	// 2. 检查是否已经创建过该事件的配置
	var count int64
	if err := db.DB(ctx).Model(&model.PushEvent{}).Where("event_key = ?", eventKey).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		return
	}
	if count > 0 {
		c.JSON(http.StatusBadRequest, response.Err("this notification event is already configured"))
		return
	}

	// 3. 模板处理
	templateStr := strings.TrimSpace(req.Template)
	if templateStr == "" {
		templateStr = string(defaultTemplateBytes)
	} else {
		var tempMap map[string]any
		if err := json.Unmarshal([]byte(templateStr), &tempMap); err != nil {
			c.JSON(http.StatusBadRequest, response.Err("custom template is not a valid JSON format"))
			return
		}
	}

	// 4. 创建事件记录
	channels := req.Channels
	if channels == nil {
		channels = []string{}
	}
	targets := req.Targets
	if targets == nil {
		targets = []string{}
	}

	event := model.PushEvent{
		EventKey: eventKey,
		Name:     eventName,
		TaskType: req.TaskType,
		Channels: channels,
		Targets:  targets,
		Template: templateStr,
		Enabled:  req.Enabled,
	}

	if err := event.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, response.Err(err.Error()))
		return
	}

	if err := db.DB(ctx).Create(&event).Error; err != nil {
		c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.OK(event))
}

// DeleteEvent 删除通知事件配置
// @Summary 删除通知事件配置
// @Description 删除数据库中的特定通知事件配置，需要管理员权限
// @Tags admin-push
// @Produce json
// @Security SessionCookie
// @Param id path int true "事件 ID"
// @Success 200 {object} response.Any{data=string} "删除成功"
// @Router /api/v1/admin/push/events/{id} [delete]
func DeleteEvent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Err("invalid event id"))
		return
	}

	ctx := c.Request.Context()
	var event model.PushEvent
	if err := db.DB(ctx).First(&event, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, response.Err("notification event not found"))
		} else {
			c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		}
		return
	}

	if err := db.DB(ctx).Delete(&event).Error; err != nil {
		c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.OKNil())
}

// UpdateEvent 更新通知事件
// @Summary 更新通知事件
// @Description 更新已有通知事件的推送渠道、接收目标和内容模板，需要管理员权限
// @Tags admin-push
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param id path int true "事件 ID"
// @Param request body push.UpdateEventRequest true "更新参数"
// @Success 200 {object} response.Any{data=string} "修改成功"
// @Router /api/v1/admin/push/events/{id} [put]
func UpdateEvent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Err("invalid event id"))
		return
	}

	var req UpdateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Err(err.Error()))
		return
	}

	var event model.PushEvent
	if err := db.DB(c.Request.Context()).First(&event, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, response.Err("notification event not found"))
		} else {
			c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		}
		return
	}

	event.Channels = req.Channels
	event.Targets = req.Targets
	event.Template = req.Template
	event.Enabled = req.Enabled

	if err := event.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, response.Err(err.Error()))
		return
	}

	if err := db.DB(c.Request.Context()).Save(&event).Error; err != nil {
		c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.OKNil())
}

// ToggleEvent 快捷切换通知事件启用状态
// @Summary 快捷切换通知事件启用状态
// @Description 启用或禁用指定的通知事件
// @Tags admin-push
// @Produce json
// @Security SessionCookie
// @Param id path int true "事件 ID"
// @Success 200 {object} response.Any{data=string} "切换成功"
// @Router /api/v1/admin/push/events/{id}/toggle [post]
func ToggleEvent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Err("invalid event id"))
		return
	}

	var event model.PushEvent
	if err := db.DB(c.Request.Context()).First(&event, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, response.Err("notification event not found"))
		} else {
			c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		}
		return
	}

	event.Enabled = !event.Enabled
	if event.Enabled && len(event.Channels) == 0 {
		c.JSON(http.StatusBadRequest, response.Err("cannot enable event without any push channels configured"))
		return
	}
	if err := db.DB(c.Request.Context()).Model(&event).Update("enabled", event.Enabled).Error; err != nil {
		c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.OK(event.Enabled))
}

// pushHistoriesResponse 推送历史分页响应
//
//nolint:unused
type pushHistoriesResponse struct {
	Total   int64               `json:"total"`
	Results []model.PushHistory `json:"results"`
}

// ListHistories 分页获取通知推送历史
// @Summary 分页获取通知推送历史
// @Description 返回分页的通知历史日志数据，需要管理员权限
// @Tags admin-push
// @Produce json
// @Security SessionCookie
// @Param page query int false "当前页码"
// @Param page_size query int false "分页大小"
// @Param event_key query string false "过滤事件名称"
// @Param status query string false "过滤发送状态"
// @Success 200 {object} response.Any{data=pushHistoriesResponse} "推送历史列表"
// @Router /api/v1/admin/push/histories [get]
func ListHistories(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")
	eventKey := c.Query("event_key")
	status := c.Query("status")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		pageSize = 20
	}

	query := db.DB(c.Request.Context()).Model(&model.PushHistory{}).Order("created_at DESC")
	if eventKey != "" {
		query = query.Where("event_key = ?", eventKey)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		return
	}

	var results []model.PushHistory
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.OK(map[string]any{
		"total":   total,
		"results": results,
	}))
}

// TestPush 测试推送通道发送
// @Summary 测试推送通道发送
// @Description 接收临时通知渠道配置并在本地同步调用 Pusher.Send 发送测试消息
// @Tags admin-push
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param request body push.TestPushRequest true "测试请求体"
// @Success 200 {object} response.Any{data=string} "测试成功"
// @Router /api/v1/admin/push/test [post]
func TestPush(c *gin.Context) {
	var req TestPushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Err(err.Error()))
		return
	}

	pusher, err := push.GetPusher(req.Config.Channel)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Err(err.Error()))
		return
	}

	// 校验配置
	if err := pusher.ValidateConfig(req.Config); err != nil {
		c.JSON(http.StatusBadRequest, response.Err(fmt.Sprintf("validation failed: %v", err)))
		return
	}

	// 邮件渠道需要从系统设置中拉取发件人 SMTP 信息做测试 (除非配了独立的)
	if req.Config.Channel == channelEmail && (req.Config.URL == "" || req.Config.Key == "") {
		var smtpHost, smtpPort, smtpUser, smtpPass model.SystemConfig
		ctx := c.Request.Context()
		_ = smtpHost.GetByKey(ctx, model.ConfigKeySMTPHost)
		_ = smtpPort.GetByKey(ctx, model.ConfigKeySMTPPort)
		_ = smtpUser.GetByKey(ctx, model.ConfigKeySMTPUsername)
		_ = smtpPass.GetByKey(ctx, model.ConfigKeySMTPPassword)

		if smtpHost.Value != "" && smtpUser.Value != "" {
			port := smtpPort.Value
			if port == "" {
				port = "587"
			}
			req.Config.URL = smtpHost.Value + ":" + port
			req.Config.Key = smtpUser.Value
			req.Config.Secret = smtpPass.Value
		}
	}

	testBody := map[string]any{
		keyTitle:   "测试通道推送",
		keyContent: "当您收到这条消息，说明当前渠道连通性测试通过。",
		keyLevel:   defaultLevelInfo,
	}

	err = pusher.Send(c.Request.Context(), req.Config, req.Target, testBody, "", nil)
	if err != nil {
		c.JSON(http.StatusOK, response.Err(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.OKNil())
}
