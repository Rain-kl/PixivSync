// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package system_config

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/logger"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/storage"
	"github.com/Rain-kl/Wavelet/internal/util"
	mail "github.com/Rain-kl/Wavelet/internal/util/mail"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const maskedConfigValue = "******"

// CreateSystemConfigRequest 创建系统配置请求
type CreateSystemConfigRequest struct {
	Key         string `json:"key" binding:"required,max=64"`
	Value       string `json:"value" binding:"required"`
	Type        string `json:"type" binding:"required,oneof=system business"`
	Visibility  int    `json:"visibility" binding:"oneof=0 1"`
	Description string `json:"description" binding:"max=255"`
}

// UpdateSystemConfigRequest 更新系统配置请求
type UpdateSystemConfigRequest struct {
	Value       string `json:"value" binding:"required"`
	Visibility  *int   `json:"visibility" binding:"omitempty,oneof=0 1"`
	Description string `json:"description" binding:"max=255"`
}

// CreateSystemConfig 创建系统配置
// @Summary 创建系统配置
// @Description 创建一条新的系统配置项，配置键不可重复，同时将新配置同步到 Redis，需要管理员权限
// @Tags admin
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param request body system_config.CreateSystemConfigRequest true "创建请求参数"
// @Success 200 {object} util.ResponseAny{data=string} "创建成功"
// @Failure 400 {object} util.ResponseAny "参数错误或配置键已存在"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Failure 403 {object} util.ResponseAny "无管理员权限"
// @Failure 500 {object} util.ResponseAny "内部错误"
// @Router /api/v1/admin/system-configs [post]
func CreateSystemConfig(c *gin.Context) {
	var req CreateSystemConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}

	// 检查配置键是否已存在
	var existing model.SystemConfig
	if err := db.DB(c.Request.Context()).Where("key = ?", req.Key).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, util.Err(ConfigKeyExists))
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
		return
	}

	config := model.SystemConfig{
		Key:         req.Key,
		Value:       req.Value,
		Type:        req.Type,
		Visibility:  req.Visibility,
		Description: req.Description,
	}

	if err := db.DB(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		// 创建配置
		if err := tx.Create(&config).Error; err != nil {
			return err
		}

		if err := db.HSetJSON(c.Request.Context(), model.SystemConfigRedisHashKey, req.Key, &config); err != nil {
			return err
		}

		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
		return
	}

	c.JSON(http.StatusOK, util.OKNil())
}

// ListSystemConfigs 获取系统配置列表
// @Summary 获取系统配置列表
// @Description 返回所有系统配置列表，支持按配置类型（system/business）过滤，需要管理员权限
// @Tags admin
// @Produce json
// @Security SessionCookie
// @Param type query string false "配置类型（system/business）"
// @Success 200 {object} util.ResponseAny{data=[]model.SystemConfig} "系统配置列表"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Failure 403 {object} util.ResponseAny "无管理员权限"
// @Failure 500 {object} util.ResponseAny "内部错误"
// @Router /api/v1/admin/system-configs [get]
func ListSystemConfigs(c *gin.Context) {
	configType := c.Query("type")
	query := db.DB(c.Request.Context()).Order("created_at DESC")
	if configType != "" {
		query = query.Where("type = ?", configType)
	}

	var configs []model.SystemConfig
	if err := query.Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
		return
	}

	for i := range configs {
		configs[i].Value = maskSensitiveConfig(configs[i].Key, configs[i].Value)
	}

	c.JSON(http.StatusOK, util.OK(configs))
}

// GetSystemConfig 获取单个系统配置
// @Summary 获取单个系统配置
// @Description 根据配置键获取对应的系统配置详情，需要管理员权限
// @Tags admin
// @Produce json
// @Security SessionCookie
// @Param key path string true "配置键"
// @Success 200 {object} util.ResponseAny{data=model.SystemConfig} "系统配置详情"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Failure 403 {object} util.ResponseAny "无管理员权限"
// @Failure 404 {object} util.ResponseAny "配置不存在"
// @Failure 500 {object} util.ResponseAny "内部错误"
// @Router /api/v1/admin/system-configs/{key} [get]
func GetSystemConfig(c *gin.Context) {
	var config model.SystemConfig
	if err := db.DB(c.Request.Context()).Where("key = ?", c.Param("key")).First(&config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, util.Err(SystemConfigNotFound))
		} else {
			c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
		}
		return
	}

	config.Value = maskSensitiveConfig(config.Key, config.Value)

	c.JSON(http.StatusOK, util.OK(config))
}

// UpdateSystemConfig 更新系统配置
// @Summary 更新系统配置
// @Description 根据配置键更新对应的配置内容，同时将更新同步到 Redis，需要管理员权限
// @Tags admin
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param key path string true "配置键"
// @Param request body system_config.UpdateSystemConfigRequest true "更新请求参数"
// @Success 200 {object} util.ResponseAny{data=string} "更新成功"
// @Failure 400 {object} util.ResponseAny "参数错误"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Failure 403 {object} util.ResponseAny "无管理员权限"
// @Failure 404 {object} util.ResponseAny "配置不存在"
// @Failure 500 {object} util.ResponseAny "内部错误"
// @Router /api/v1/admin/system-configs/{key} [put]
func UpdateSystemConfig(c *gin.Context) {
	var req UpdateSystemConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}

	key := c.Param("key")

	// 检查配置是否存在
	var config model.SystemConfig
	if err := db.DB(c.Request.Context()).Where("key = ?", key).First(&config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, util.Err(SystemConfigNotFound))
		} else {
			c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
		}
		return
	}

	var originalDriver storage.Driver
	if key == model.ConfigKeyStorageConfig {
		var currentCfg storage.Config
		if err := json.Unmarshal([]byte(config.Value), &currentCfg); err == nil {
			originalDriver = currentCfg.Driver
		}
	}

	if err := db.DB(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		// 更新配置
		updates := map[string]interface{}{
			"description": req.Description,
		}
		if req.Visibility != nil {
			updates["visibility"] = *req.Visibility
			config.Visibility = *req.Visibility
		}
		if key != model.ConfigKeySMTPPassword || req.Value != maskedConfigValue {
			updates["value"] = req.Value
			config.Value = req.Value
		}
		if err := tx.Model(&config).Updates(updates).Error; err != nil {
			return err
		}

		if err := db.HSetJSON(c.Request.Context(), model.SystemConfigRedisHashKey, key, &config); err != nil {
			return err
		}

		if key == model.ConfigKeyStorageConfig && originalDriver != "" {
			var newCfg storage.Config
			if err := json.Unmarshal([]byte(req.Value), &newCfg); err == nil {
				if newCfg.Driver == originalDriver {
					// Mark failed storage:migrate task execution as succeeded
					if err := tx.Model(&model.TaskExecution{}).
						Where("task_type = ? AND status = ?", "storage:migrate", model.TaskExecutionStatusFailed).
						Updates(map[string]any{
							"status":      model.TaskExecutionStatusSucceeded,
							"result":      "存储配置直接更新，故障迁移任务自动标记为已解决",
							"finished_at": time.Now(),
						}).Error; err != nil {
						logger.ErrorF(c.Request.Context(), "自动更新迁移任务状态失败: %v", err)
					}
				}
			}
		}

		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
		return
	}

	if key == model.ConfigKeyStorageConfig {
		storage.ResetCache()
		storage.PublishCacheInvalidation(c.Request.Context())
	}

	c.JSON(http.StatusOK, util.OKNil())
}

// TestSMTPRequest 测试 SMTP 配置请求
type TestSMTPRequest struct {
	SMTPHost     string `json:"smtp_host" binding:"required,max=255"`
	SMTPPort     int    `json:"smtp_port" binding:"required"`
	SMTPUsername string `json:"smtp_username" binding:"required,max=255"`
	SMTPPassword string `json:"smtp_password" binding:"required,max=255"`
	To           string `json:"to" binding:"required,email"`
}

// TestSMTPResponse 测试 SMTP 配置响应
type TestSMTPResponse struct {
	Success bool   `json:"success"`
	Log     string `json:"log"`
	Error   string `json:"error"`
}

// TestSMTP 测试 SMTP 邮件发送
// @Summary 测试 SMTP 邮件发送
// @Description 使用传入的配置进行 SMTP 邮件发送测试，支持使用 ****** 占位符使用保存的数据库密码
// @Tags admin
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param request body system_config.TestSMTPRequest true "测试请求参数"
// @Success 200 {object} util.ResponseAny{data=system_config.TestSMTPResponse} "测试执行完毕"
// @Failure 400 {object} util.ResponseAny "参数错误"
// @Router /api/v1/admin/system-configs/smtp/test [post]
func TestSMTP(c *gin.Context) {
	var req TestSMTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}

	password := req.SMTPPassword
	if password == maskedConfigValue {
		var sc model.SystemConfig
		if err := sc.GetByKey(c.Request.Context(), model.ConfigKeySMTPPassword); err == nil {
			password = sc.Value
		}
	}

	cfg := mail.Config{
		Host:     req.SMTPHost,
		Port:     req.SMTPPort,
		Username: req.SMTPUsername,
		Password: password,
	}

	subject := "Wavelet SMTP Test Mail"
	body := `<h3>SMTP Mail Connection Test</h3>
<p>If you received this message, your SMTP configuration is correct and mail sending is working properly.</p>
<p>Sent from Wavelet.</p>`

	logs, err := mail.SendMailWithLog(c.Request.Context(), cfg, req.To, subject, body)
	resp := TestSMTPResponse{
		Success: err == nil,
		Log:     logs,
	}
	if err != nil {
		resp.Error = err.Error()
	}

	c.JSON(http.StatusOK, util.OK(resp))
}

func maskSensitiveConfig(key, value string) string {
	if value == "" {
		return value
	}
	switch key {
	case model.ConfigKeySMTPPassword:
		return maskedConfigValue
	case model.ConfigKeyStorageConfig:
		var cfg storage.Config
		if err := json.Unmarshal([]byte(value), &cfg); err == nil {
			masked := storage.MaskSecrets(cfg)
			if val, err := json.Marshal(masked); err == nil {
				return string(val)
			}
		}
	}
	return value
}
