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

import "time"

const (
	verificationCodeRange  = 900000           // 验证码随机范围
	verificationCodeOffset = 100000           // 验证码偏移量（保证 6 位）
	emailCodeExpiry        = 5 * time.Minute  // 验证码有效期
	emailCodeCooldown      = 60 * time.Second // 验证码发送冷却时间
	minPasswordLength      = 8                // 密码最小长度
)
