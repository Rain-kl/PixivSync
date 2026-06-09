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

package user

const (
	errBindParamsFailed          = "参数绑定失败"
	errInvalidParams             = "无效的参数"
	errPasswordLoginDisabled     = "管理员关闭了密码登录"
	errUsernameOrPasswordWrong   = "用户名或密码错误"
	errLoginEmailMissing         = "该账号未绑定邮箱，请联系管理员绑定邮箱后再登录"
	errNeedEmailCodePrefix       = "need_email_code:"
	errEmailCodeInvalidOrExpired = "验证码错误或已过期"
	errPasswordUpgradeFailed     = "升级密码安全算法失败，请重试"
	errSaveSessionFailed         = "无法保存会话信息，请重试"
	errRegistrationDisabled      = "管理员关闭了注册"
	errPasswordTooShort          = "密码长度不能少于 8 位"
	errEmailOrCodeRequired       = "邮箱或验证码未填写"
	errNewPasswordTooShort       = "新密码长度不能少于 8 位"
	errLoginRequired             = "请先登录"
	errUserNotFound              = "未找到该用户"
	errOldPasswordIncorrect      = "原密码不正确"
	errPasswordEncryptFailed     = "密码加密失败，请重试"
	errEmailRequired             = "邮箱地址不能为空"
	errUnsupportedEmailScene     = "不支持的验证场景"
	errEmailAlreadyRegistered    = "该邮箱已被注册"
	errEmailCodeCooldown         = "验证码发送频繁，请稍后再试"
	errEmailFormatInvalid        = "邮箱格式不正确"
	errEmailAlreadyBound         = "该邮箱已被其他账号绑定"
	errRenderEmailTemplateFailed = "渲染验证邮件模板失败：%w"
	errGenerateEmailCodeFailed   = "生成验证码失败，请重试"
	errDispatchEmailTaskFailed   = "投递验证邮件发送任务失败，请重试"
	errTokenNameRequired         = "令牌名称不能为空"
	errAccessTokenLimitReached   = "已达到访问令牌最大创建数量限制"
	errGenerateTokenFailed       = "生成令牌失败"
	errInvalidTokenID            = "无效的令牌ID"
	errTokenNotFoundOrForbidden  = "令牌不存在或无权操作"
	errTaskPayloadRequired       = "任务参数不能为空"
	errInvalidJSONFormat         = "无效的 JSON 格式: %w"
	errEmailTaskFieldsRequired   = "to、subject、body 不能为空"
	errParseEmailPayloadFailed   = "解析邮件发送参数失败: %w"
	errSMTPConfigIncomplete      = "系统 SMTP 邮件服务配置不完整"
	errSendMailFailed            = "发送邮件失败: %w"
)
