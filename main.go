// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package main 是 Wavelet 平台的程序入口
package main

import "github.com/Rain-kl/Wavelet/internal/cmd"

// @title Wavelet API
// @version 1.0.0
// @description Wavelet 平台后端 API，提供用户认证、系统配置、任务调度等通用功能。
// @contact.name Wavelet
// @contact.url https://github.com/Rain-kl/Wavelet
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @BasePath /
// @securityDefinitions.apikey SessionCookie
// @in cookie
// @name session
func main() {
	cmd.Execute()
}
