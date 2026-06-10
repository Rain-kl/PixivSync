// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package task 定义异步任务类型与调度常量
package task

// 异步任务类型标识
const (
	CleanupUnusedUploadsTask = "upload:cleanup_unused"
	SendEmailTask            = "mail:send"
	PixezMirrorIllustTask    = "pixez:mirror_illust"
	PixezMirrorNovelTask     = "pixez:mirror_novel"
	PixezExportIllustTask    = "pixez:export_bookmark_illusts"
	PixezExportNovelTask     = "pixez:export_bookmark_novels"
	PixezAutoMirrorTask      = "pixez:auto_enqueue_bookmark_mirrors"
	PixezImportLegacyTask    = "pixez:import_legacy_server"
)

// 任务队列名称
const (
	QueueDefault = "default"
)

// 管理员可下发的任务类型标识
const (
	TaskTypeCleanupUploads                  = "cleanup_unused_uploads"
	TaskTypeSendEmail                       = "send_email"
	TaskTypePixezMirrorIllust               = "pixez_mirror_illust"
	TaskTypePixezMirrorNovel                = "pixez_mirror_novel"
	TaskTypePixezExportIllustBookmarks      = "pixez_export_bookmark_illusts"
	TaskTypePixezExportNovelBookmarks       = "pixez_export_bookmark_novels"
	TaskTypePixezAutoEnqueueBookmarkMirrors = "pixez_auto_enqueue_bookmark_mirrors"
	TaskTypePixezImportLegacyServer         = "pixez_import_legacy_server"
)

// defaultMaxRetry 任务默认最大重试次数
const defaultMaxRetry = 3

// TaskParam 任务参数定义
//
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
//
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
	{
		Type:         TaskTypePixezMirrorIllust,
		AsynqTask:    PixezMirrorIllustTask,
		Name:         "PixEz 镜像插画",
		Description:  "抓取 Pixiv 插画详情并把原图写入 Wavelet Upload 存储",
		SupportsTime: false,
		MaxRetry:     defaultMaxRetry,
		Queue:        QueueDefault,
		Retryable:    true,
		Params: []TaskParam{
			{Name: "illust_id", Label: "插画 ID", Type: "number", Required: true, Placeholder: "123456", Description: "Pixiv 插画 ID"},
		},
	},
	{
		Type:         TaskTypePixezMirrorNovel,
		AsynqTask:    PixezMirrorNovelTask,
		Name:         "PixEz 镜像小说",
		Description:  "抓取 Pixiv 小说详情和正文并保存为镜像 read-model",
		SupportsTime: false,
		MaxRetry:     defaultMaxRetry,
		Queue:        QueueDefault,
		Retryable:    true,
		Params: []TaskParam{
			{Name: "novel_id", Label: "小说 ID", Type: "number", Required: true, Placeholder: "123456", Description: "Pixiv 小说 ID"},
		},
	},
	{
		Type:         TaskTypePixezExportIllustBookmarks,
		AsynqTask:    PixezExportIllustTask,
		Name:         "PixEz 导出插画收藏",
		Description:  "导出已同步 Pixiv 账号的插画收藏并增量维护 removed 状态",
		SupportsTime: false,
		MaxRetry:     defaultMaxRetry,
		Queue:        QueueDefault,
		Retryable:    true,
		Params: []TaskParam{
			{Name: "pixiv_user_id", Label: "Pixiv 用户 ID", Type: "string", Required: false, Placeholder: "留空表示全部账号", Description: "只导出指定 Pixiv 用户时填写"},
		},
	},
	{
		Type:         TaskTypePixezExportNovelBookmarks,
		AsynqTask:    PixezExportNovelTask,
		Name:         "PixEz 导出小说收藏",
		Description:  "导出已同步 Pixiv 账号的小说收藏并增量维护 removed 状态",
		SupportsTime: false,
		MaxRetry:     defaultMaxRetry,
		Queue:        QueueDefault,
		Retryable:    true,
		Params: []TaskParam{
			{Name: "pixiv_user_id", Label: "Pixiv 用户 ID", Type: "string", Required: false, Placeholder: "留空表示全部账号", Description: "只导出指定 Pixiv 用户时填写"},
		},
	},
	{
		Type:         TaskTypePixezAutoEnqueueBookmarkMirrors,
		AsynqTask:    PixezAutoMirrorTask,
		Name:         "PixEz 收藏自动入队镜像",
		Description:  "扫描收藏 read-model，把未镜像或失败的收藏批量下发镜像任务",
		SupportsTime: false,
		MaxRetry:     defaultMaxRetry,
		Queue:        QueueDefault,
		Retryable:    true,
		Params: []TaskParam{
			{Name: "target_type", Label: "目标类型", Type: "string", Required: false, Placeholder: "illust 或 novel，留空表示全部", Description: "限制扫描插画或小说收藏"},
			{Name: "limit", Label: "数量上限", Type: "number", Required: false, Placeholder: "50", Description: "本次最多入队数量"},
		},
	},
	{
		Type:         TaskTypePixezImportLegacyServer,
		AsynqTask:    PixezImportLegacyTask,
		Name:         "PixEz 导入旧后端",
		Description:  "从旧 server/pixez-sync.db 和 server/data/mirror 导入业务数据",
		SupportsTime: false,
		MaxRetry:     1,
		Queue:        QueueDefault,
		Retryable:    false,
		Params: []TaskParam{
			{Name: "sqlite_path", Label: "旧 SQLite 路径", Type: "string", Required: false, Placeholder: "server/pixez-sync.db", Description: "旧 PixEz Sync SQLite 文件"},
			{Name: "mirror_dir", Label: "旧镜像目录", Type: "string", Required: false, Placeholder: "server/data/mirror", Description: "旧插画镜像文件目录"},
			{Name: "dry_run", Label: "只预览", Type: "string", Required: false, Placeholder: "false", Description: "true 表示只统计不写入"},
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
