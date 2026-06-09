/*
Copyright 2026 Arctel.net

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

package user

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/Rain-kl/Wavelet/internal/apps/oauth"
	"github.com/Rain-kl/Wavelet/internal/common"
	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/db/idgen"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/util"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Code     string `json:"code"`
}

type registerRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	Nickname    string `json:"nickname"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Code        string `json:"code"`
}

func isPasswordLoginEnabled() bool {
	enabled, err := model.GetBoolByKey(context.Background(), model.ConfigKeyPasswordLoginEnabled)
	if err != nil {
		return true
	}
	return enabled
}

func isPasswordRegisterEnabled() bool {
	enabled, err := model.GetBoolByKey(context.Background(), model.ConfigKeyPasswordRegisterEnabled)
	if err != nil {
		return true
	}
	return enabled
}

func isRegistrationEnabled() bool {
	enabled, err := model.GetBoolByKey(context.Background(), model.ConfigKeyRegistrationEnabled)
	if err != nil {
		return true
	}
	return enabled
}

func setLoginSession(c *gin.Context, user *model.User) error {
	session := sessions.Default(c)
	session.Set(oauth.UserIDKey, user.ID)
	session.Set(oauth.UserNameKey, user.Username)
	if err := session.Save(); err != nil {
		return err
	}
	return nil
}

// Login 用户密码登录
// @Summary 用户密码登录
// @Description 使用用户名和密码登录，登录成功后建立 Session。若管理员已关闭密码登录功能则返回错误。
// @Tags user
// @Accept json
// @Produce json
// @Param request body user.loginRequest true "登录请求参数"
// @Success 200 {object} util.ResponseAny{data=oauth.BasicUserInfo} "登录成功，返回用户信息"
// @Failure 400 {object} util.ResponseAny "用户名或密码错误、帐号已禁用等"
// @Failure 500 {object} util.ResponseAny "服务内部错误"
// @Router /api/v1/user/login [post]
func Login(c *gin.Context) {
	if !isPasswordLoginEnabled() {
		c.JSON(http.StatusOK, util.Err(errPasswordLoginDisabled))
		return
	}
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusOK, util.Err(errInvalidParams))
		return
	}

	var user model.User
	ctx := c.Request.Context()
	if err := db.DB(ctx).Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(errUsernameOrPasswordWrong))
		return
	}
	if !user.IsActive {
		c.JSON(http.StatusOK, util.Err(common.BannedAccount))
		return
	}

	// 判定是否是明文密码存储
	isPlaintext := !user.IsPasswordEncrypted()

	if !user.CheckPassword(req.Password) {
		c.JSON(http.StatusOK, util.Err(errUsernameOrPasswordWrong))
		return
	}

	if isEmailLoginVerificationEnabled() {
		if emailErr := handleLoginEmailVerification(ctx, c, &req, &user); emailErr != nil {
			return
		}
	}

	session := sessions.Default(c)
	needChangePassword := isPlaintext

	if isPlaintext {
		session.Set("need_change_password", true)
	} else {
		session.Delete("need_change_password")
	}

	user.LastLoginAt = time.Now()
	if err := db.DB(ctx).Model(&user).Update("last_login_at", user.LastLoginAt).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(err.Error()))
		return
	}
	if err := setLoginSession(c, &user); err != nil {
		c.JSON(http.StatusOK, util.Err(errSaveSessionFailed))
		return
	}

	// 检查是否有未完成 of OAuth/OIDC 绑定
	completePendingOAuthBinding(session, &user)

	c.JSON(http.StatusOK, util.OK(oauth.BuildBasicUserInfo(&user, needChangePassword)))
}

// Register 用户注册
// @Summary 用户注册
// @Description 使用用户名和密码注册新账号，注册成功后自动登录并建立 Session。密码长度不能少于 8 位。
// @Tags user
// @Accept json
// @Produce json
// @Param request body user.registerRequest true "注册请求参数"
// @Success 200 {object} util.ResponseAny{data=oauth.BasicUserInfo} "注册并登录成功，返回用户信息"
// @Failure 400 {object} util.ResponseAny "参数错误、用户名已存在或注册已关闭"
// @Failure 500 {object} util.ResponseAny "服务内部错误"
// @Router /api/v1/user/register [post]
func Register(c *gin.Context) {
	if !isRegistrationEnabled() || !isPasswordRegisterEnabled() {
		c.JSON(http.StatusOK, util.Err(errRegistrationDisabled))
		return
	}

	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)
	req.Nickname = strings.TrimSpace(req.Nickname)
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	req.Email = strings.TrimSpace(req.Email)
	req.Code = strings.TrimSpace(req.Code)

	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusOK, util.Err(errInvalidParams))
		return
	}
	if len(req.Password) < minPasswordLength {
		c.JSON(http.StatusOK, util.Err(errPasswordTooShort))
		return
	}

	ctx := c.Request.Context()

	// 邮箱注册验证校验
	if err := validateRegisterEmailVerification(ctx, &req); err != nil {
		c.JSON(http.StatusOK, util.Err(err.Error()))
		return
	}

	user := model.User{
		ID:          idgen.NextUint64ID(),
		Username:    req.Username,
		Nickname:    req.Nickname,
		Email:       req.Email,
		AvatarURL:   "",
		IsActive:    true,
		IsAdmin:     false,
		LastLoginAt: time.Now(),
	}
	if user.Nickname == "" {
		user.Nickname = req.DisplayName
	}
	if user.Nickname == "" {
		user.Nickname = req.Username
	}
	if err := user.SetEncryptedPassword(req.Password); err != nil {
		c.JSON(http.StatusOK, util.Err(err.Error()))
		return
	}

	if err := user.RegisterUser(ctx, db.DB(ctx)); err != nil {
		c.JSON(http.StatusOK, util.Err(err.Error()))
		return
	}

	if err := setLoginSession(c, &user); err != nil {
		c.JSON(http.StatusOK, util.Err(errSaveSessionFailed))
		return
	}

	c.JSON(http.StatusOK, util.OK(oauth.BuildBasicUserInfo(&user, false)))
}

// Logout 用户退出登录
// @Summary 用户退出登录
// @Description 清除用户登录 Session，完成退出
// @Tags user
// @Produce json
// @Security SessionCookie
// @Success 200 {object} util.ResponseAny{data=string} "退出成功"
// @Failure 500 {object} util.ResponseAny "Session 清除失败"
// @Router /api/v1/user/logout [get]
func Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Options(util.GetSessionOptions(-1))
	session.Clear()
	if err := session.Save(); err != nil {
		c.JSON(http.StatusOK, util.Err(err.Error()))
		return
	}
	c.JSON(http.StatusOK, util.OK(""))
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// ChangePassword 修改用户密码
// @Summary 修改用户密码
// @Description 修改当前登录用户的密码。修改成功后，如果是首次明文登录的升级提示，则清除修改密码的提示状态。
// @Tags user
// @Accept json
// @Produce json
// @Param request body user.changePasswordRequest true "修改密码请求参数"
// @Success 200 {object} util.ResponseAny{data=string} "修改密码成功"
// @Failure 400 {object} util.ResponseAny "原密码错误或新密码不符合要求"
// @Failure 401 {object} util.ResponseAny "请先登录"
// @Router /api/v1/user/change-password [post]
func ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, util.Err(err.Error()))
		return
	}

	req.OldPassword = strings.TrimSpace(req.OldPassword)
	req.NewPassword = strings.TrimSpace(req.NewPassword)

	if req.OldPassword == "" || req.NewPassword == "" {
		c.JSON(http.StatusOK, util.Err(errInvalidParams))
		return
	}
	if len(req.NewPassword) < minPasswordLength {
		c.JSON(http.StatusOK, util.Err(errNewPasswordTooShort))
		return
	}

	userObj, _ := util.GetFromContext[*model.User](c, oauth.UserObjKey)
	if userObj == nil {
		c.JSON(http.StatusUnauthorized, util.Err(errLoginRequired))
		return
	}

	ctx := c.Request.Context()
	var dbUser model.User
	if err := db.DB(ctx).Where("id = ?", userObj.ID).First(&dbUser).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(errUserNotFound))
		return
	}

	// 校验旧密码
	if !dbUser.CheckPassword(req.OldPassword) {
		c.JSON(http.StatusOK, util.Err(errOldPasswordIncorrect))
		return
	}

	// 加密并更新为新密码
	if err := dbUser.SetEncryptedPassword(req.NewPassword); err != nil {
		c.JSON(http.StatusOK, util.Err(errPasswordEncryptFailed))
		return
	}

	if err := db.DB(ctx).Model(&dbUser).Update("password", dbUser.Password).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(err.Error()))
		return
	}

	// 清除 Session 中修改密码提示状态
	session := sessions.Default(c)
	session.Delete("need_change_password")
	_ = session.Save()

	c.JSON(http.StatusOK, util.OK("密码修改成功"))
}
