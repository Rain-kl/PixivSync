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

package upload

const (
	maxUploadSize      = 32 * 1024 * 1024 // 32MB
	detectContentBytes = 512              // http.DetectContentType 需要的最小字节数
	uploadDirPerm      = 0755             // 上传目录权限
	uploadFilePerm     = 0644             // 上传文件权限
)
