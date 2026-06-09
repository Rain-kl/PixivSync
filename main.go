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
