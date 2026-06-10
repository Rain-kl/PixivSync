// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package handlers 注册异步任务处理器
package handlers

import (
	"github.com/Rain-kl/Wavelet/internal/apps/pixez"
	"github.com/Rain-kl/Wavelet/internal/apps/upload"
	"github.com/Rain-kl/Wavelet/internal/apps/user"
	"github.com/Rain-kl/Wavelet/internal/task"
)

// Register registers all built-in task handlers.
func Register() {
	task.RegisterHandler(task.CleanupUnusedUploadsTask, &upload.CleanupUnusedUploadsHandler{})
	task.RegisterHandler(task.SendEmailTask, &user.SendEmailHandler{})
	task.RegisterHandler(task.PixezMirrorTask, &pixez.MirrorTaskHandler{})
	task.RegisterHandler(task.PixezExportBookmarksTask, &pixez.ExportBookmarksTaskHandler{})
	task.RegisterHandler(task.PixezAutoMirrorTask, &pixez.AutoEnqueueBookmarkMirrorsTaskHandler{})
	task.RegisterHandler(task.PixezImportLegacyTask, &pixez.ImportLegacyServerTaskHandler{})
}
