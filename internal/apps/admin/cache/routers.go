// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

// Package cache provides HTTP handlers for managing disk cache.
package cache

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/diskcache"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/util"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type updateCacheConfigRequest struct {
	MaxSizeMB  int64 `json:"max_size_mb" binding:"required,min=1"`
	TTLMinutes int64 `json:"ttl_minutes" binding:"required,min=0"`
	LRUEnabled bool  `json:"lru_enabled"`
}

// GetCacheStatus 获取磁盘缓存状态与当前统计数据
// @Summary 获取缓存状态
// @Description 获取当前系统磁盘缓存的使用情况（已占用字节、Key 数量等）与策略配置
// @Tags admin
// @Produce json
// @Security SessionCookie
// @Success 200 {object} util.ResponseAny{data=diskcache.Status} "获取成功"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Failure 403 {object} util.ResponseAny "无管理员权限"
// @Failure 500 {object} util.ResponseAny "内部错误"
// @Router /api/v1/admin/cache/status [get]
func GetCacheStatus(c *gin.Context) {
	status := diskcache.GetGlobalCache().Status()
	c.JSON(http.StatusOK, util.OK(status))
}

// UpdateCacheConfig 更新磁盘缓存策略配置
// @Summary 更新缓存配置
// @Description 更改磁盘缓存最大容量限制、文件生存时间（TTL）以及是否启用 LRU 淘汰淘汰算法，并进行热更新
// @Tags admin
// @Accept json
// @Produce json
// @Param request body cache.updateCacheConfigRequest true "缓存配置请求体"
// @Security SessionCookie
// @Success 200 {object} util.ResponseAny "更新成功"
// @Failure 400 {object} util.ResponseAny "参数错误"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Failure 403 {object} util.ResponseAny "无管理员权限"
// @Failure 500 {object} util.ResponseAny "服务内部错误"
// @Router /api/v1/admin/cache/config [post]
func UpdateCacheConfig(c *gin.Context) {
	var req updateCacheConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}

	ctx := c.Request.Context()

	// Update Max Size
	if err := saveOrUpdateConfig(ctx, model.ConfigKeyDiskCacheMaxSizeMB, strconv.FormatInt(req.MaxSizeMB, 10)); err != nil {
		c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
		return
	}

	// Update Default TTL
	if err := saveOrUpdateConfig(ctx, model.ConfigKeyDiskCacheTTLMinutes, strconv.FormatInt(req.TTLMinutes, 10)); err != nil {
		c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
		return
	}

	// Update LRU Enabled
	if err := saveOrUpdateConfig(ctx, model.ConfigKeyDiskCacheLRUEnabled, strconv.FormatBool(req.LRUEnabled)); err != nil {
		c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
		return
	}

	// Trigger hot reloading in global cache
	diskcache.GetGlobalCache().ReloadConfig(ctx)

	c.JSON(http.StatusOK, util.OKNil())
}

// ClearCache 一键清空所有磁盘缓存数据
// @Summary 清空缓存
// @Description 清除系统磁盘缓存目录中的所有临时文件，并重置缓存容量和 Key 追踪数据
// @Tags admin
// @Produce json
// @Security SessionCookie
// @Success 200 {object} util.ResponseAny "清理成功"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Failure 403 {object} util.ResponseAny "无管理员权限"
// @Failure 500 {object} util.ResponseAny "服务内部错误"
// @Router /api/v1/admin/cache/clear [post]
func ClearCache(c *gin.Context) {
	if err := diskcache.GetGlobalCache().Clear(); err != nil {
		c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
		return
	}
	c.JSON(http.StatusOK, util.OKNil())
}

func saveOrUpdateConfig(ctx context.Context, key string, value string) error {
	var sc model.SystemConfig
	err := db.DB(ctx).Where("key = ?", key).First(&sc).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		sc = model.SystemConfig{
			Key:        key,
			Value:      value,
			Type:       "system",
			Visibility: 0,
		}
		if err := db.DB(ctx).Create(&sc).Error; err != nil {
			return err
		}
	} else {
		sc.Value = value
		if err := db.DB(ctx).Save(&sc).Error; err != nil {
			return err
		}
	}

	// Sync config value in Redis cache
	if db.Redis != nil {
		_ = db.HSetJSON(ctx, model.SystemConfigRedisHashKey, key, &sc)
	}
	return nil
}
