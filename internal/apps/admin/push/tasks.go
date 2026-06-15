// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package push defines push notification HTTP routes and background tasks.
package push

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/task"
	"github.com/Rain-kl/Wavelet/pkg/push"
)

const (
	// SendNotificationTask 发送推送通知任务标识
	SendNotificationTask = "push:send"
	// TaskTypeSendNotification 推送通知管理类型
	TaskTypeSendNotification = "send_notification"
)

// SendNotificationMeta represents the task metadata.
var SendNotificationMeta = task.TaskMeta{
	Type:         TaskTypeSendNotification,
	AsynqTask:    SendNotificationTask,
	Name:         "推送通知",
	Description:  "异步执行系统通知的多渠道派发与推送",
	SupportsTime: false,
	MaxRetry:     task.DefaultMaxRetry,
	Queue:        task.QueueDefault,
	Retryable:    true,
	Params: []task.TaskParam{
		{
			Name:        "event_key",
			Label:       "事件标识",
			Type:        "string",
			Required:    true,
			Placeholder: "admin_login",
		},
		{
			Name:     "target",
			Label:    "目标接收者",
			Type:     "string",
			Required: false,
		},
	},
}

// PushHandler 通知推送异步任务处理器
//
//nolint:revive
type PushHandler struct{}

// ValidatePayload 校验并标准化推送参数
func (h *PushHandler) ValidatePayload(payload []byte) ([]byte, error) {
	if len(payload) == 0 {
		return nil, errors.New("payload is required")
	}

	var req SendPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("invalid json format: %w", err)
	}

	if req.Config.Channel == "" {
		return nil, errors.New("channel type is required")
	}

	return json.Marshal(req)
}

// Execute 异步执行推送操作并记录推送历史审计
func (h *PushHandler) Execute(ctx context.Context, payload []byte) (*task.TaskResult, error) {
	var req SendPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		task.AppendLog(ctx, "解析推送参数失败: %v", err)
		return nil, fmt.Errorf("parse payload failed: %w", err)
	}

	task.AppendLog(ctx, "开始推送通知: 事件 = %s, 渠道 = %s, 接收目标 = %s", req.EventKey, req.Config.Channel, req.Target)

	pusher, err := push.GetPusher(req.Config.Channel)
	if err != nil {
		errWrap := fmt.Errorf("get pusher failed: %w", err)
		task.AppendLog(ctx, "推送失败: %v", errWrap)
		if task.IsFinalAttempt(ctx) {
			h.recordHistory(ctx, req, "failed", errWrap.Error())
		}
		return nil, errWrap
	}

	// 执行真正的消息推送，扁平化为原始 json 格式
	flatBody := req.Body.Flatten()
	err = pusher.Send(ctx, req.Config, req.Target, flatBody, req.Template, nil)

	title := req.Body.Title
	content := req.Body.Content

	if err != nil {
		task.AppendLog(ctx, "消息推送失败 (标题: %s): %v", title, err)
		if task.IsFinalAttempt(ctx) {
			h.recordHistory(ctx, req, "failed", err.Error())
		}
		return nil, fmt.Errorf("pusher.Send failed: %w", err)
	}

	task.AppendLog(ctx, "消息推送成功 (标题: %s, 内容摘要: %s)", title, content)
	h.recordHistory(ctx, req, "success", "")

	return &task.TaskResult{
		Message: fmt.Sprintf("推送成功: [%s] -> %s", req.Config.Channel, req.Target),
	}, nil
}

func (h *PushHandler) recordHistory(ctx context.Context, req SendPayload, status string, errMsg string) {
	title := req.Body.Title
	content := req.Body.Content
	level := req.Body.Level

	if title == "" {
		title = "系统通知"
	}
	if level == "" {
		level = defaultLevelInfo
	}

	target := req.Target
	if target == "" {
		// 如果目标人为空 (例如 webhook bot)，用其地址填充前缀或默认词作为归档
		if req.Config.URL != "" {
			target = req.Config.URL
			// 隐藏敏感 URL 细节
			//nolint:mnd
			if len(target) > 50 {
				target = target[:47] + "..."
			}
		} else {
			target = "default"
		}
	}

	history := model.PushHistory{
		EventKey:  req.EventKey,
		Channel:   req.Config.Channel,
		Target:    target,
		Title:     title,
		Content:   content,
		Level:     level,
		Status:    status,
		ErrorMsg:  errMsg,
		CreatedAt: time.Now(),
	}

	// 记录到数据库
	if dbErr := db.DB(ctx).Create(&history).Error; dbErr != nil {
		task.AppendLog(ctx, "写入推送历史审计记录失败: %v", dbErr)
	}
}
