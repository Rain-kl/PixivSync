// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package user

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Rain-kl/Wavelet/internal/apps/oauth"
	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/db/idgen"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/util"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Rain-kl/Wavelet/internal/common/response"
)

// minPasswordLength 密码最小长度
const minPasswordLength = 8

// listUsersRequest 用户列表查询请求
type listUsersRequest struct {
	Page     int     `form:"page" binding:"min=1"`
	PageSize int     `form:"page_size" binding:"min=1,max=100"`
	UserID   *uint64 `form:"user_id" binding:"omitempty,gt=0"`
	Username string  `form:"username"`
}

type user struct {
	ID          uint64    `json:"id,string"`
	Username    string    `json:"username"`
	Nickname    string    `json:"nickname"`
	Email       string    `json:"email"`
	AvatarURL   string    `json:"avatar_url"`
	IsActive    bool      `json:"is_active"`
	IsAdmin     bool      `json:"is_admin"`
	Bio         string    `json:"bio"`
	Phone       string    `json:"phone"`
	Gender      string    `json:"gender"`
	Website     string    `json:"website"`
	Location    string    `json:"location"`
	LastLoginAt time.Time `json:"last_login_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// listUsersResponse 用户列表响应
type listUsersResponse struct {
	Users []user `json:"users"`
	Total int64  `json:"total"`
}

func parseUserID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, response.Err(userNotFound))
		return 0, false
	}
	return id, true
}

func toUser(u model.User) user {
	return user{
		ID:          u.ID,
		Username:    u.Username,
		Nickname:    u.Nickname,
		Email:       u.Email,
		AvatarURL:   u.AvatarURL,
		IsActive:    u.IsActive,
		IsAdmin:     u.IsAdmin,
		Bio:         u.Bio,
		Phone:       u.Phone,
		Gender:      u.Gender,
		Website:     u.Website,
		Location:    u.Location,
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

// ListUsers 获取用户列表
// @Summary 获取用户列表
// @Description 分页返回用户列表，支持按用户 ID 和用户名筛选，需要管理员权限
// @Tags admin
// @Produce json
// @Security SessionCookie
// @Param request query listUsersRequest true "查询参数"
// @Success 200 {object} response.Any{data=user.listUsersResponse} "用户列表"
// @Failure 400 {object} response.Any "参数错误"
// @Failure 401 {object} response.Any "未登录"
// @Failure 403 {object} response.Any "无管理员权限"
// @Failure 500 {object} response.Any "内部错误"
// @Router /api/v1/admin/users [get]
// ListUsers 获取用户列表
func ListUsers(c *gin.Context) {
	var req listUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Err(err.Error()))
		return
	}

	var modelUsers []model.User
	var total int64

	query := db.DB(c.Request.Context()).Model(&model.User{})

	username := strings.TrimSpace(req.Username)

	if req.UserID != nil {
		query = query.Where("id = ?", *req.UserID)
	}

	if username != "" {
		query = query.Where("username LIKE ?", username+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		return
	}

	offset := (req.Page - 1) * req.PageSize
	if err := query.
		Select("id, username, nickname, avatar_url, is_active, is_admin, " +
			"last_login_at, created_at, updated_at").
		Order("id DESC").
		Offset(offset).
		Limit(req.PageSize).
		Find(&modelUsers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		return
	}

	users := make([]user, 0, len(modelUsers))
	for _, modelUser := range modelUsers {
		users = append(users, toUser(modelUser))
	}

	c.JSON(http.StatusOK, response.OK(listUsersResponse{
		Users: users,
		Total: total,
	}))
}

// GetUser 获取用户详情
// @Summary 获取用户详情
// @Description 返回指定用户的完整个人资料和系统状态，需要管理员权限，不返回密码等敏感字段
// @Tags admin
// @Produce json
// @Security SessionCookie
// @Param id path int true "用户 ID"
// @Success 200 {object} response.Any{data=user.user} "用户详情"
// @Failure 400 {object} response.Any "参数错误"
// @Failure 401 {object} response.Any "未登录"
// @Failure 403 {object} response.Any "无管理员权限"
// @Failure 404 {object} response.Any "用户不存在"
// @Failure 500 {object} response.Any "内部错误"
// @Router /api/v1/admin/users/{id} [get]
func GetUser(c *gin.Context) {
	id, ok := parseUserID(c)
	if !ok {
		return
	}

	var targetUser model.User
	if err := db.DB(c.Request.Context()).
		Select("id, username, nickname, email, avatar_url, is_active, is_admin, "+
			"bio, phone, gender, website, location, last_login_at, created_at, updated_at").
		Where("id = ?", id).
		First(&targetUser).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, response.Err(userNotFound))
			return
		}
		c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.OK(toUser(targetUser)))
}

// updateUserStatusRequest 更新用户状态请求
type updateUserStatusRequest struct {
	IsActive bool `json:"is_active"`
}

// UpdateUserStatus 更新用户状态（启用/禁用）
// @Summary 更新用户状态
// @Description 启用或禁用指定用户，管理员账号无法被禁用，需要管理员权限
// @Tags admin
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param id path int true "用户 ID"
// @Param request body updateUserStatusRequest true "状态参数"
// @Success 200 {object} response.Any{data=string} "更新成功"
// @Failure 400 {object} response.Any "参数错误"
// @Failure 401 {object} response.Any "未登录"
// @Failure 403 {object} response.Any "无管理员权限或尝试禁用管理员"
// @Failure 404 {object} response.Any "用户不存在"
// @Failure 500 {object} response.Any "内部错误"
// @Router /api/v1/admin/users/{id}/status [put]
func UpdateUserStatus(c *gin.Context) {
	var req updateUserStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Err(err.Error()))
		return
	}

	id, ok := parseUserID(c)
	if !ok {
		return
	}

	var targetUser struct {
		ID      uint64 `gorm:"column:id"`
		IsAdmin bool   `gorm:"column:is_admin"`
	}
	if err := db.DB(c.Request.Context()).
		Model(&model.User{}).
		Select("id, is_admin").
		Where("id = ?", id).
		First(&targetUser).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, response.Err(userNotFound))
			return
		}
		c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		return
	}

	if !req.IsActive && targetUser.IsAdmin {
		c.JSON(http.StatusForbidden, response.Err(cannotDisable))
		return
	}

	if err := db.DB(c.Request.Context()).
		Model(&model.User{}).
		Where("id = ?", id).
		Update("is_active", req.IsActive).Error; err != nil {
		c.JSON(http.StatusInternalServerError, response.Err(updateUserFailed))
		return
	}

	c.JSON(http.StatusOK, response.OKNil())
}

// DeleteUser 删除用户
// @Summary 删除用户
// @Description 删除指定非管理员用户，需要管理员权限，不能删除当前登录用户
// @Tags admin
// @Produce json
// @Security SessionCookie
// @Param id path int true "用户 ID"
// @Success 200 {object} response.Any{data=string} "删除成功"
// @Failure 400 {object} response.Any "参数错误"
// @Failure 401 {object} response.Any "未登录"
// @Failure 403 {object} response.Any "无管理员权限、尝试删除管理员或当前用户"
// @Failure 404 {object} response.Any "用户不存在"
// @Failure 500 {object} response.Any "内部错误"
// @Router /api/v1/admin/users/{id} [delete]
func DeleteUser(c *gin.Context) {
	id, ok := parseUserID(c)
	if !ok {
		return
	}

	currUser, _ := util.GetFromContext[*model.User](c, oauth.UserObjKey)
	if currUser != nil && currUser.ID == id {
		c.JSON(http.StatusForbidden, response.Err(cannotDeleteSelf))
		return
	}

	var targetUser struct {
		ID      uint64 `gorm:"column:id"`
		IsAdmin bool   `gorm:"column:is_admin"`
	}
	if err := db.DB(c.Request.Context()).
		Model(&model.User{}).
		Select("id, is_admin").
		Where("id = ?", id).
		First(&targetUser).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, response.Err(userNotFound))
			return
		}
		c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		return
	}

	if targetUser.IsAdmin {
		c.JSON(http.StatusForbidden, response.Err(cannotDelete))
		return
	}

	if err := db.DB(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", id).Delete(&model.AccessToken{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", id).Delete(&model.ExternalAccount{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", id).Delete(&model.User{}).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, response.Err(deleteUserFailed))
		return
	}

	c.JSON(http.StatusOK, response.OKNil())
}

// createUserRequest 创建用户请求
type createUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=8,max=64"`
	Nickname string `json:"nickname" binding:"omitempty,max=64"`
	Email    string `json:"email" binding:"required,email,max=255"`
	IsActive bool   `json:"is_active"`
	IsAdmin  bool   `json:"is_admin"`
}

// CreateUser 创建用户
// @Summary 创建用户
// @Description 创建一个本地密码登录的新用户，需要管理员权限
// @Tags admin
// @Accept json
// @Produce json
// @Security SessionCookie
// @Param request body user.createUserRequest true "创建用户参数"
// @Success 200 {object} response.Any{data=user.user} "创建成功"
// @Failure 400 {object} response.Any "参数错误或用户名已存在"
// @Failure 401 {object} response.Any "未登录"
// @Failure 403 {object} response.Any "无管理员权限"
// @Failure 500 {object} response.Any "内部错误"
// @Router /api/v1/admin/users [post]
func CreateUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Err(err.Error()))
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Nickname = strings.TrimSpace(req.Nickname)
	req.Password = strings.TrimSpace(req.Password)
	req.Email = strings.TrimSpace(req.Email)

	if req.Username == "" {
		c.JSON(http.StatusBadRequest, response.Err(usernameRequired))
		return
	}
	if req.Email == "" {
		c.JSON(http.StatusBadRequest, response.Err(emailRequired))
		return
	}
	if len(req.Password) < minPasswordLength {
		c.JSON(http.StatusBadRequest, response.Err(passwordTooShort))
		return
	}

	ctx := c.Request.Context()
	var count int64
	if err := db.DB(ctx).Model(&model.User{}).Where("username = ?", req.Username).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		return
	}
	if count > 0 {
		c.JSON(http.StatusBadRequest, response.Err(usernameExists))
		return
	}

	var emailCount int64
	if err := db.DB(ctx).Model(&model.User{}).Where("email = ?", req.Email).Count(&emailCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		return
	}
	if emailCount > 0 {
		c.JSON(http.StatusBadRequest, response.Err(emailExists))
		return
	}

	newUser := model.User{
		ID:          idgen.NextUint64ID(),
		Username:    req.Username,
		Nickname:    req.Nickname,
		Email:       req.Email,
		IsActive:    req.IsActive,
		IsAdmin:     req.IsAdmin,
		LastLoginAt: time.Time{},
	}
	if newUser.Nickname == "" {
		newUser.Nickname = req.Username
	}

	if err := newUser.SetEncryptedPassword(req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		return
	}

	if err := db.DB(ctx).Create(&newUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, response.Err(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.OK(toUser(newUser)))
}
