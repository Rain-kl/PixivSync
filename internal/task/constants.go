/*
Copyright 2025 linux.do
Modified by Arctel.net, 2026

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package task 定义异步任务类型与调度常量
package task

// 异步任务类型标识
const (
	CleanupUnusedUploadsTask = "upload:cleanup_unused"
	SendEmailTask            = "mail:send"
)

// 任务队列名称
const (
	QueueDefault = "default"
)

// 管理员可下发的任务类型标识
const (
	TaskTypeCleanupUploads = "cleanup_unused_uploads"
	TaskTypeSendEmail      = "send_email"
)

// defaultMaxRetry 任务默认最大重试次数
const defaultMaxRetry = 3

// TaskParam 任务参数定义
//nolint:revive // TaskParam 保留完整名称以避免与通用 Param 混淆
type TaskParam struct {
	Name        string `json:"Name"`        // 参数键名
	Label       string `json:"Label"`       // 显示名称
	Type        string `json:"Type"`        // 类型：string, text, number
	Required    bool   `json:"Required"`    // 是否必填
	Placeholder string `json:"Placeholder"` // 占位符
	Description string `json:"Description"` // 描述
}

// TaskMeta 任务元数据
//nolint:revive // TaskMeta 保留完整名称以避免与通用 Meta 混淆
type TaskMeta struct {
	Type         string
	AsynqTask    string
	Name         string
	Description  string
	SupportsTime bool
	MaxRetry     int
	Queue        string
	Retryable    bool // 是否支持手动重试
	Params       []TaskParam
}

// DispatchableTasks 可下发的任务列表
var DispatchableTasks = []TaskMeta{
	{
		Type:         TaskTypeCleanupUploads,
		AsynqTask:    CleanupUnusedUploadsTask,
		Name:         "清理未使用上传",
		Description:  "清理超过1小时未使用的上传文件",
		SupportsTime: false,
		MaxRetry:     defaultMaxRetry,
		Queue:        QueueDefault,
		Retryable:    true,
	},
	{
		Type:         TaskTypeSendEmail,
		AsynqTask:    SendEmailTask,
		Name:         "发送邮件",
		Description:  "异步发送系统邮件",
		SupportsTime: false,
		MaxRetry:     defaultMaxRetry,
		Queue:        QueueDefault,
		Retryable:    true,
		Params: []TaskParam{
			{
				Name:        "to",
				Label:       "接收邮箱 (To)",
				Type:        "string",
				Required:    true,
				Placeholder: "receiver@example.com",
				Description: "接收邮件的目标邮箱地址",
			},
			{
				Name:        "subject",
				Label:       "邮件主题 (Subject)",
				Type:        "string",
				Required:    true,
				Placeholder: "请输入邮件主题",
				Description: "发送邮件的主题标题",
			},
			{
				Name:        "body",
				Label:       "邮件内容 (Body)",
				Type:        "text",
				Required:    true,
				Placeholder: "请输入邮件内容（支持 HTML 格式）",
				Description: "发送邮件的内容主体",
			},
		},
	},
}

// GetTaskMeta 根据任务类型获取元数据
func GetTaskMeta(taskType string) *TaskMeta {
	for _, t := range DispatchableTasks {
		if t.Type == taskType {
			return &t
		}
	}
	return nil
}
