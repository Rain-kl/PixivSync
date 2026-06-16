// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

// Package custom_events defines custom push notification events.
package custom_events

import (
	"context"
	"time"

	"github.com/Rain-kl/Wavelet/internal/apps/admin/push"
	"github.com/Rain-kl/Wavelet/internal/model"
)

// AdminLogin is the metadata definition for the admin login event.
var AdminLogin = push.EventMetadata{
	Key:  "admin_login",
	Name: "管理员登录",
	DefaultTemplate: push.NotificationMessage{
		Title:   "管理员登录提醒",
		Content: "管理员 {{user.username}} 于 {{time}} 从 IP {{ip}} 登录系统。",
		Level:   "INFO",
	},
	Description: "当管理员成功登录系统时触发此通知",
}

func init() {
	push.RegisterBuiltInEvent(AdminLogin)
}

// TriggerAdminLoginEvent triggers the admin login event asynchronously.
func TriggerAdminLoginEvent(ctx context.Context, user *model.User, ip string) {
	if user == nil || !user.IsAdmin {
		return
	}

	body := map[string]any{
		"user": user,
		"ip":   ip,
		"time": time.Now().Format("2006-01-02 15:04:05"),
	}
	push.DefaultTrigger.Trigger(ctx, AdminLogin, body)
}
