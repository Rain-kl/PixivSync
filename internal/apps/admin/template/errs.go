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

// Package template 提供模板管理功能
package template

// 模板管理相关错误消息
const (
	TemplateNotFound              = "模板不存在"
	TemplateKeyRequired           = "模板标识符不能为空"
	TemplateNameRequired          = "模板名称不能为空"
	TemplateContentRequired       = "模板内容不能为空"
	TemplateKeyExists             = "模板标识符已存在"
	SystemTemplateCannotDelete    = "系统预置模板不可删除"
	SystemTemplateCannotModifyKey = "系统预置模板不可修改标识符"
)
