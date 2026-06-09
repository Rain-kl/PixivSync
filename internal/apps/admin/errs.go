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

// Package admin 提供管理后台功能
package admin

// 管理后台错误消息常量
const (
	AdminRequired          = "未经授权访问"
	InvalidAuthSourceID    = "认证源 ID 无效"
	InvalidCursorParam     = "无效的 cursor 参数"
	InvalidTaskExecutionID = "无效的任务执行记录 ID"
)
