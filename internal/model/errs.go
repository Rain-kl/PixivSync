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

package model

const (
	errRegistrationDisabled                 = "注册已关闭"
	errDatabaseNotInitialized               = "database not initialized"
	errUsernameExists                       = "用户名已存在"
	errEmailAlreadyBound                    = "该邮箱已被其他账号绑定"
	errConfigIntParseFailed                 = "配置 %s 的值 '%s' 无法转换为整数: %w"
	errConfigDecimalParseFailed             = "配置 %s 的值 '%s' 无法转换为decimal: %w"
	errConfigBoolParseFailed                = "配置 %s 的值 '%s' 无法转换为布尔值: %w"
	errParseMenuDisplayConfigFailed         = "解析目录显示配置失败: %w"
	errTemplateKeyRequired                  = "模板标识符不能为空"
	errTemplateNameRequired                 = "模板名称不能为空"
	errTemplateContentRequired              = "模板内容不能为空"
	errTemplateUnavailable                  = "模板 %s 不存在或不可用: %w"
	errTemplateRenderFailed                 = "模板 %s 渲染失败: %w"
	errAuthSourceNameRequired               = "认证源名称不能为空"
	errAuthSourceNameInvalid                = "认证源名称只能包含字母、数字、短横线或下划线，且必须以字母或数字开头"
	errAuthSourceTypeUnsupported            = "认证源类型仅支持 oidc"
	errAuthSourceDiscoveryURLRequired       = "OIDC 认证源必须配置 Discovery URL"
	errAuthSourceClientCredentialsRequired  = "启用认证源前必须配置 Client ID 和 Client Secret"
	errAuthSourceIDRequired                 = "认证源 ID 不能为空"
	errExternalAccountBindingIncomplete     = "外部账号绑定信息不完整"
	errExternalAccountAlreadyBoundToAnother = "该外部账号已绑定到其他用户"
	errUserIDRequired                       = "用户 ID 不能为空"
	errExternalAccountBindingIDRequired     = "绑定记录 ID 不能为空"
)
