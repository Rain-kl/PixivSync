// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package push

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/logger"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/task"
)

func init() {
	task.OnTaskCompleted = handleTaskCompleted
}

// handleTaskCompleted handles task completions and triggers appropriate push events.
func handleTaskCompleted(ctx context.Context, execution *model.TaskExecution, result *task.TaskResult, execErr error) {
	// Query all active push events configured for this task type
	var events []model.PushEvent
	err := db.DB(ctx).Where("task_type = ? AND enabled = ?", execution.TaskType, true).Find(&events).Error
	if err != nil {
		logger.ErrorF(ctx, "push_task_completed_listener: failed to query push events for task type %s: %v", execution.TaskType, err)
		return
	}

	if len(events) == 0 {
		return
	}

	// Build the notification body context
	body := map[string]any{
		"task_id":       execution.TaskID,
		"task_name":     execution.TaskName,
		"task_type":     execution.TaskType,
		"task_status":   string(execution.Status),
		"task_duration": execution.Duration,
		"time":          time.Now().Format("2006-01-02 15:04:05"),
	}

	if execErr != nil {
		body["task_error"] = execErr.Error()
	} else {
		body["task_error"] = ""
	}

	if result != nil {
		body["task_result"] = result.Message
	} else {
		body["task_result"] = ""
	}

	// Parse payload parameters if it is valid JSON
	var payloadMap map[string]any
	if execution.Payload != "" {
		if err := json.Unmarshal([]byte(execution.Payload), &payloadMap); err == nil {
			body["payload"] = payloadMap
			extractUserFromMap(ctx, payloadMap, body)
		}
	}

	// Parse result detail parameters if it is valid JSON
	if result != nil && result.Detail != "" {
		var detailMap map[string]any
		if err := json.Unmarshal([]byte(result.Detail), &detailMap); err == nil {
			body["detail"] = detailMap
			extractUserFromMap(ctx, detailMap, body)
		}
	}

	// Trigger notifications for all configured events
	for _, event := range events {
		meta := EventMetadata{
			Key:         event.EventKey,
			Name:        event.Name,
			Description: "异步任务执行完毕触发的自动通知",
		}
		DefaultTrigger.Trigger(ctx, meta, body)
	}
}

// extractUserFromMap tries to find user information from a map and load the full User model.
func extractUserFromMap(ctx context.Context, data map[string]any, body map[string]any) {
	if u, exists := body["user"]; exists && u != nil {
		return
	}

	if uVal, ok := data["user"]; ok && uVal != nil {
		body["user"] = uVal
		return
	}

	if userID, ok := extractUserID(data); ok && userID > 0 {
		var u model.User
		if err := db.DB(ctx).Where("id = ?", userID).First(&u).Error; err == nil {
			body["user"] = &u
			return
		}
	}

	if username := extractUsername(data); username != "" {
		var u model.User
		if err := db.DB(ctx).Where("username = ?", username).First(&u).Error; err == nil {
			body["user"] = &u
			return
		}
	}
}

// extractUserID extracts and validates a userID from map fields.
func extractUserID(data map[string]any) (uint64, bool) {
	for _, k := range []string{"user_id", "userId", "uid"} {
		val, ok := data[k]
		if !ok || val == nil {
			continue
		}
		switch v := val.(type) {
		case float64:
			if v >= 0 {
				return uint64(v), true
			}
		case int:
			if v >= 0 {
				return uint64(v), true
			}
		case int64:
			if v >= 0 {
				return uint64(v), true
			}
		case uint64:
			return v, true
		case string:
			if id, err := strconv.ParseUint(v, 10, 64); err == nil {
				return id, true
			}
		}
	}
	return 0, false
}

// extractUsername extracts a username string from map fields.
func extractUsername(data map[string]any) string {
	for _, k := range []string{"username", "user_name"} {
		if val, ok := data[k]; ok && val != nil {
			if s, ok := val.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}
