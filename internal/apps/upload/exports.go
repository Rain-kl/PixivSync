// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package upload re-exports upload subsystem handlers and utilities for stable cross-module imports.
package upload

import (
	"github.com/Rain-kl/Wavelet/internal/apps/upload/cache"
	"github.com/Rain-kl/Wavelet/internal/apps/upload/filesrv"
	"github.com/Rain-kl/Wavelet/internal/apps/upload/handler"
	uploadstats "github.com/Rain-kl/Wavelet/internal/apps/upload/stats"
	uploadtask "github.com/Rain-kl/Wavelet/internal/apps/upload/task"
	"github.com/Rain-kl/Wavelet/internal/apps/upload/util"
	"github.com/Rain-kl/Wavelet/internal/task"
)

// HTTP handlers
var (
	UploadFile             = handler.UploadFile
	DownloadFile           = handler.DownloadFile
	BatchDownloadFiles     = handler.BatchDownloadFiles
	ListFiles              = handler.ListFiles
	DeleteFile             = handler.DeleteFile
	GetDistinctUploadTypes = handler.GetDistinctUploadTypes
	ListMyFiles            = handler.ListMyFiles
	DeleteMyFile           = handler.DeleteMyFile
	UpdateMyFile           = handler.UpdateMyFile
	GetFileStats           = handler.GetFileStats
	ServeFileByID          = filesrv.ServeFileByID
	ServeUpload            = filesrv.ServeUpload
)

// Cache management
var (
	ResetAccessCaches              = cache.ResetAccessCaches
	PublishAccessCacheInvalidation = cache.PublishAccessCacheInvalidation
)

// Stats
var (
	ApplyUploadStatsAdd    = uploadstats.ApplyUploadStatsAdd
	ApplyUploadStatsRemove = uploadstats.ApplyUploadStatsRemove
	RebuildUploadStats     = uploadstats.RebuildUploadStats
)

// Utilities
var (
	CompressImageToWebP = util.CompressImageToWebP
	ValidateS3Key       = util.ValidateS3Key
)

// Task identifiers and metadata
const (
	StorageMigrationTask = uploadtask.StorageMigrationTask
	SystemCleanupTask    = uploadtask.SystemCleanupTask
	WarmImageCacheTask   = uploadtask.WarmImageCacheTask
)

var (
	// StorageMigrationMeta describes the storage migration async task.
	StorageMigrationMeta = uploadtask.StorageMigrationMeta
	// SystemCleanupMeta describes the orphaned upload cleanup task.
	SystemCleanupMeta = uploadtask.SystemCleanupMeta
	// WarmImageCacheMeta describes the image compression cache warmup task.
	WarmImageCacheMeta = uploadtask.WarmImageCacheMeta
)

// MigrationHandler executes storage migration tasks.
type MigrationHandler = uploadtask.MigrationHandler

// SystemCleanupHandler removes orphaned upload files.
type SystemCleanupHandler = uploadtask.SystemCleanupHandler

// WarmImageCacheHandler pre-warms compressed image caches.
type WarmImageCacheHandler = uploadtask.WarmImageCacheHandler

// WarmImageCachePayload is the payload for image cache warmup tasks.
type WarmImageCachePayload = uploadtask.WarmImageCachePayload

// Ensure task handler types implement required interfaces.
var (
	_ task.TaskHandler = (*MigrationHandler)(nil)
	_ task.TaskHandler = (*SystemCleanupHandler)(nil)
	_ interface {
		task.TaskHandler
		ValidatePayload([]byte) ([]byte, error)
	} = (*WarmImageCacheHandler)(nil)
)