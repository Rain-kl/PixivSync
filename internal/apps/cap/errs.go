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

// Package cap 提供人机验证中间件
package cap

const (
	errCapTokenMissing          = "验证码验证失败，缺少验证码凭证"
	errCapTokenInvalidOrExpired = "验证码校验失败或已过期，请重试"
)
