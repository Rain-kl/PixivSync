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

package oauth

// OAuth 认证相关错误消息
const (
	InvalidState                    = "非法登录请求"
	IDTokenVerifyFailed             = "ID Token 验证失败"
	IDTokenVerifyFailedFormat       = "%s: %w"
	NonceMismatch                   = "nonce 不匹配，可能存在重放攻击"
	NoActiveAuthSource              = "未配置可用认证源"
	ServerAddressMissing            = "服务器地址 (server_address) 未配置或配置为空，请在后台系统设置中配置后再试"
	AuthSourceRequired              = "认证源不能为空"
	DiscoveryURLRequired            = "OIDC 认证源必须配置 Discovery URL"
	UsernameGenerateFailed          = "无法生成可用用户名"
	UsernameFromSourceFailed        = "无法从认证源获取用户名"
	AuthSourceDisabled              = "认证源未启用"
	InvalidExternalAccountBindingID = "绑定记录 ID 无效"
)
