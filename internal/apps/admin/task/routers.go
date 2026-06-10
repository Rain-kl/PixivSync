// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package task

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Rain-kl/Wavelet/internal/apps/admin"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/task"
	taskhandlers "github.com/Rain-kl/Wavelet/internal/task/handlers"
	"github.com/Rain-kl/Wavelet/internal/util"
	"github.com/gin-gonic/gin"
)

func init() {
	taskhandlers.Register()
}

// ListTaskTypes 获取支持的任务类型列表
// @Summary 获取支持的任务类型
// @Description 返回系统支持的所有可调度任务类型列表，包括任务名称、描述、是否支持时间范围等元数据，需要管理员权限
// @Tags admin
// @Produce json
// @Security SessionCookie
// @Success 200 {object} util.ResponseAny{data=[]task.TaskMeta} "任务类型列表"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Failure 403 {object} util.ResponseAny "无管理员权限"
// @Router /api/v1/admin/tasks/types [get]
func ListTaskTypes(c *gin.Context) {
	c.JSON(http.StatusOK, util.OK(task.DispatchableTasks))
}

// DispatchTaskRequest 下发任务请求
type DispatchTaskRequest struct {
	TaskType  string     `json:"task_type" binding:"required"`
	StartTime *time.Time `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`
	UserID    *uint64    `json:"user_id"`
	Payload   string     `json:"payload"`
}

// DispatchTask 下发任务
// @Summary 下发异步任务
// @Description 手动触发指定类型的异步任务，支持指定时间范围和用户，需要管理员权限
// @Tags admin
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param request body DispatchTaskRequest true "任务请求参数"
// @Success 200 {object} util.ResponseAny{data=string} "任务已入队"
// @Failure 400 {object} util.ResponseAny "任务类型不存在或参数错误"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Failure 403 {object} util.ResponseAny "无管理员权限"
// @Failure 500 {object} util.ResponseAny "任务入队失败"
// @Router /api/v1/admin/tasks/dispatch [post]
func DispatchTask(c *gin.Context) {
	var req DispatchTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}

	meta := task.GetTaskMeta(req.TaskType)
	if meta == nil {
		c.JSON(http.StatusBadRequest, util.Err(InvalidTaskType))
		return
	}

	var payloadBytes []byte
	if strings.TrimSpace(req.Payload) != "" {
		payloadBytes = []byte(req.Payload)
	}

	validated, err := task.ValidateAndNormalizePayload(meta.AsynqTask, payloadBytes)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}

	taskID, err := task.DispatchTask(c.Request.Context(), req.TaskType, validated, "manual")
	if err != nil {
		c.JSON(http.StatusInternalServerError, util.Err(fmt.Sprintf("%s: %v", TaskDispatchFailed, err)))
		return
	}

	c.JSON(http.StatusOK, util.OK(taskID))
}

// ListTaskExecutions 查询任务执行记录列表
// @Summary 查询任务执行记录
// @Description 分页查询任务执行记录，支持按状态和任务类型筛选，需要管理员权限
// @Tags admin
// @Produce json
// @Security SessionCookie
// @Param status query string false "状态筛选 (pending/running/succeeded/failed)"
// @Param task_type query string false "任务类型筛选"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页条数" default(20)
// @Success 200 {object} util.ResponseAny{data=object} "任务执行记录列表"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Failure 403 {object} util.ResponseAny "无管理员权限"
// @Router /api/v1/admin/tasks/executions [get]
func ListTaskExecutions(c *gin.Context) {
	var req model.ListTaskExecutionsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}

	executions, total, err := model.ListTaskExecutions(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
		return
	}

	c.JSON(http.StatusOK, util.OK(gin.H{
		"items":     executions,
		"total":     total,
		"page":      req.Page,
		"page_size": req.PageSize,
	}))
}

// GetTaskExecution 查询单条任务执行详情
// @Summary 查询任务执行详情
// @Description 根据 ID 查询任务执行记录详情，包含完整执行日志，需要管理员权限
// @Tags admin
// @Produce json
// @Security SessionCookie
// @Param id path int true "任务执行记录 ID"
// @Success 200 {object} util.ResponseAny{data=model.TaskExecution} "任务执行详情"
// @Failure 400 {object} util.ResponseAny "参数错误"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Failure 403 {object} util.ResponseAny "无管理员权限"
// @Failure 404 {object} util.ResponseAny "记录不存在"
// @Router /api/v1/admin/tasks/executions/{id} [get]
func GetTaskExecution(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.Err(admin.InvalidTaskExecutionID))
		return
	}

	execution, err := model.GetTaskExecutionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, util.Err(TaskNotFound))
		return
	}

	c.JSON(http.StatusOK, util.OK(execution))
}

// RetryTask 重试失败的任务
// @Summary 重试失败任务
// @Description 重新下发一条失败的任务，创建新的执行记录，需要管理员权限
// @Tags admin
// @Produce json
// @Security SessionCookie
// @Param id path int true "任务执行记录 ID"
// @Success 200 {object} util.ResponseAny{data=string} "新任务的 TaskID"
// @Failure 400 {object} util.ResponseAny "任务不支持重试或参数错误"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Failure 403 {object} util.ResponseAny "无管理员权限"
// @Failure 404 {object} util.ResponseAny "记录不存在"
// @Failure 500 {object} util.ResponseAny "重试失败"
// @Router /api/v1/admin/tasks/executions/{id}/retry [post]
func RetryTask(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.Err(admin.InvalidTaskExecutionID))
		return
	}

	newTaskID, err := task.RetryTask(c.Request.Context(), id)
	if err != nil {
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "不存在"):
			c.JSON(http.StatusNotFound, util.Err(errMsg))
		case strings.Contains(errMsg, "只有失败") || strings.Contains(errMsg, "不支持重试") || strings.Contains(errMsg, "已达到最大重试"):
			c.JSON(http.StatusBadRequest, util.Err(errMsg))
		default:
			c.JSON(http.StatusInternalServerError, util.Err(fmt.Sprintf("%s: %v", TaskRetryFailed, err)))
		}
		return
	}

	c.JSON(http.StatusOK, util.OK(newTaskID))
}
