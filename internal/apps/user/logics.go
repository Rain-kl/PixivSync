// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package user

import ("context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/gin-contrib/sessions"

	"github.com/Rain-kl/Wavelet/internal/apps/oauth"
	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/task"
	"github.com/Rain-kl/Wavelet/internal/util"
	pkgu "github.com/Rain-kl/Wavelet/pkg/util"
	"github.com/gin-gonic/gin"

	"github.com/Rain-kl/Wavelet/internal/common/response")

type sendEmailCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
	Scene string `json:"scene" binding:"required"`
}

func isEmailLoginVerificationEnabled(ctx context.Context) bool {
	enabled, err := model.GetBoolByKey(ctx, model.ConfigKeyEmailLoginVerificationEnabled)
	if err != nil {
		return false
	}
	return enabled
}

func isEmailRegisterVerificationEnabled(ctx context.Context) bool {
	enabled, err := model.GetBoolByKey(ctx, model.ConfigKeyEmailRegisterVerificationEnabled)
	if err != nil {
		return false
	}
	return enabled
}

func isSMTPConfigured(ctx context.Context) bool {
	var host, port, username, password string

	var scHost model.SystemConfig
	if err := scHost.GetByKey(ctx, model.ConfigKeySMTPHost); err == nil {
		host = scHost.Value
	}
	var scPort model.SystemConfig
	if err := scPort.GetByKey(ctx, model.ConfigKeySMTPPort); err == nil {
		port = scPort.Value
	}
	var scUser model.SystemConfig
	if err := scUser.GetByKey(ctx, model.ConfigKeySMTPUsername); err == nil {
		username = scUser.Value
	}
	var scPass model.SystemConfig
	if err := scPass.GetByKey(ctx, model.ConfigKeySMTPPassword); err == nil {
		password = scPass.Value
	}

	return host != "" && port != "" && username != "" && password != ""
}

func generateVerificationCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(verificationCodeRange))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()+verificationCodeOffset), nil
}

func getEmailCodeKey(scene, email string) string {
	return fmt.Sprintf("email_code:%s:%s", scene, email)
}

func getEmailCooldownKey(scene, email string) string {
	return fmt.Sprintf("email_code:cooldown:%s:%s", scene, email)
}

func sendEmailVerificationCode(ctx context.Context, email, scene, templateName string) error {
	if !isSMTPConfigured(ctx) {
		return errors.New(errSMTPConfigIncomplete)
	}

	code, err := generateVerificationCode()
	if err != nil {
		return errors.New(errGenerateEmailCodeFailed)
	}
	codeKey := getEmailCodeKey(scene, email)
	cooldownKey := getEmailCooldownKey(scene, email)

	emailSubject, emailBody, err := model.RenderTemplate(
		ctx,
		templateName,
		map[string]any{"Code": code},
	)
	if err != nil {
		return fmt.Errorf(errRenderEmailTemplateFailed, err)
	}

	if err := db.SetJSON(ctx, codeKey, code, emailCodeExpiry); err != nil {
		return errors.New(errGenerateEmailCodeFailed)
	}
	_ = db.SetJSON(ctx, cooldownKey, "1", emailCodeCooldown)

	payload := SendEmailPayload{
		To:      email,
		Subject: emailSubject,
		Body:    emailBody,
	}
	payloadBytes, _ := json.Marshal(payload)
	_, err = task.DispatchTask(ctx, TaskTypeSendEmail, payloadBytes, "system")
	if err != nil {
		return errors.New(errDispatchEmailTaskFailed)
	}
	return nil
}

func verifyEmailCode(ctx context.Context, email, scene, code string) bool {
	codeKey := getEmailCodeKey(scene, email)
	var storedCode string
	if err := db.GetJSON(ctx, codeKey, &storedCode); err != nil {
		return false
	}
	if storedCode != code {
		return false
	}
	_ = db.Redis.Del(ctx, db.PrefixedKey(codeKey)).Err()
	return true
}

func handleLoginEmailVerification(ctx context.Context, c *gin.Context, req *loginRequest, user *model.User) error {
	if req.Code != "" {
		if !verifyEmailCode(ctx, user.Email, "login", req.Code) {
			c.JSON(http.StatusOK, response.Err(errEmailCodeInvalidOrExpired))
			return errors.New("handled")
		}
		return nil
	}

	// 如果 SMTP 未配置，或者用户没有绑定邮箱（无法发送验证码），则使用临时码 888888
	if !isSMTPConfigured(ctx) || user.Email == "" {
		codeKey := getEmailCodeKey("login", user.Email)
		if err := db.SetJSON(ctx, codeKey, "888888", emailCodeExpiry); err != nil {
			c.JSON(http.StatusOK, response.Err(errGenerateEmailCodeFailed))
			return errors.New("handled")
		}
		var msg string
		if !isSMTPConfigured(ctx) {
			msg = errSMTPInvalidUseTempCodePrefix + errSMTPInvalidUseTempCode
		} else {
			msg = errSMTPInvalidUseTempCodePrefix + "该账号未绑定邮箱，使用临时码登录"
		}
		c.JSON(http.StatusOK, response.Err(msg))
		return errors.New("handled")
	}

	cooldownKey := getEmailCooldownKey("login", user.Email)
	var temp string
	err := db.GetJSON(ctx, cooldownKey, &temp)
	if err != nil {
		if err := sendEmailVerificationCode(ctx, user.Email, "login", "login_email"); err != nil {
			c.JSON(http.StatusOK, response.Err(err.Error()))
			return errors.New("handled")
		}
	}

	maskedEmail := pkgu.MaskEmail(user.Email)
	c.JSON(http.StatusOK, response.Err(errNeedEmailCodePrefix+maskedEmail))
	return errors.New("handled")
}

// SendEmailCode 发送邮箱验证码
// @Summary 发送邮箱验证码
// @Description 向指定邮箱发送验证码（用于注册场景）
// @Tags user
// @Accept json
// @Produce json
// @Param request body user.sendEmailCodeRequest true "发送验证码请求参数"
// @Success 200 {object} response.Any "发送成功"
// @Failure 400 {object} response.Any "参数错误"
// @Router /api/v1/user/send-email-code [post]
func SendEmailCode(c *gin.Context) {
	var req sendEmailCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Err(err.Error()))
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" {
		c.JSON(http.StatusOK, response.Err(errEmailRequired))
		return
	}

	if req.Scene != "register" {
		c.JSON(http.StatusOK, response.Err(errUnsupportedEmailScene))
		return
	}

	ctx := c.Request.Context()

	var count int64
	if err := db.DB(ctx).Model(&model.User{}).Where("email = ?", req.Email).Count(&count).Error; err != nil {
		c.JSON(http.StatusOK, response.Err(err.Error()))
		return
	}
	if count > 0 {
		c.JSON(http.StatusOK, response.Err(errEmailAlreadyRegistered))
		return
	}

	cooldownKey := getEmailCooldownKey("register", req.Email)
	var temp string
	err := db.GetJSON(ctx, cooldownKey, &temp)
	if err == nil {
		c.JSON(http.StatusOK, response.Err(errEmailCodeCooldown))
		return
	}

	if err := sendEmailVerificationCode(ctx, req.Email, "register", "register_email"); err != nil {
		c.JSON(http.StatusOK, response.Err(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.OKNil())
}

func validateRegisterEmailVerification(ctx context.Context, req *registerRequest) error {
	if !isEmailRegisterVerificationEnabled(ctx) {
		return nil
	}
	if req.Email == "" || req.Code == "" {
		return errors.New(errEmailOrCodeRequired)
	}
	if !verifyEmailCode(ctx, req.Email, "register", req.Code) {
		return errors.New(errEmailCodeInvalidOrExpired)
	}
	return nil
}

type updateProfileRequest struct {
	Nickname  string `json:"nickname"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	Bio       string `json:"bio"`
	Phone     string `json:"phone"`
	Gender    string `json:"gender"`
	Website   string `json:"website"`
	Location  string `json:"location"`
}

// UpdateProfile 修改当前登录用户的个人资料
// @Summary 修改当前登录用户的个人资料
// @Description 修改当前登录用户的昵称、邮箱、头像、简介、电话、性别、个人网站和所在地。
// @Tags user
// @Accept json
// @Produce json
// @Param request body user.updateProfileRequest true "更新请求参数"
// @Success 200 {object} response.Any{data=oauth.BasicUserInfo} "修改成功，返回更新后的用户信息"
// @Failure 400 {object} response.Any "邮箱已被占用或参数错误"
// @Failure 401 {object} response.Any "未登录"
// @Router /api/v1/user/profile [put]
func UpdateProfile(c *gin.Context) {
	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Err(err.Error()))
		return
	}

	userObj, _ := util.GetFromContext[*model.User](c, oauth.UserObjKey)
	if userObj == nil {
		c.JSON(http.StatusUnauthorized, response.Err(errLoginRequired))
		return
	}

	ctx := c.Request.Context()
	var dbUser model.User
	if err := db.DB(ctx).Where("id = ?", userObj.ID).First(&dbUser).Error; err != nil {
		c.JSON(http.StatusOK, response.Err(errUserNotFound))
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email != "" && req.Email != dbUser.Email {
		if !strings.Contains(req.Email, "@") || !strings.Contains(req.Email, ".") {
			c.JSON(http.StatusOK, response.Err(errEmailFormatInvalid))
			return
		}

		var count int64
		if err := db.DB(ctx).Model(&model.User{}).Where("email = ? AND id != ?", req.Email, dbUser.ID).Count(&count).Error; err != nil {
			c.JSON(http.StatusOK, response.Err(err.Error()))
			return
		}
		if count > 0 {
			c.JSON(http.StatusOK, response.Err(errEmailAlreadyBound))
			return
		}
	}

	dbUser.Nickname = strings.TrimSpace(req.Nickname)
	if dbUser.Nickname == "" {
		dbUser.Nickname = dbUser.Username
	}
	dbUser.Email = req.Email
	dbUser.AvatarURL = req.AvatarURL
	dbUser.Bio = req.Bio
	dbUser.Phone = strings.TrimSpace(req.Phone)
	dbUser.Gender = strings.TrimSpace(req.Gender)
	dbUser.Website = strings.TrimSpace(req.Website)
	dbUser.Location = strings.TrimSpace(req.Location)

	if err := db.DB(ctx).Save(&dbUser).Error; err != nil {
		c.JSON(http.StatusOK, response.Err(err.Error()))
		return
	}

	session := sessions.Default(c)
	needChange := session.Get("need_change_password") == true

	c.JSON(http.StatusOK, response.OK(oauth.BuildBasicUserInfo(&dbUser, needChange)))
}
