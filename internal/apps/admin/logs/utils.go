/*
Copyright 2025-2026 linux.do
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

package logs

import (
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
)

// getUpgrader 返回 WebSocket 升级器
func getUpgrader() *websocket.Upgrader {
	return &websocket.Upgrader{
		CheckOrigin: func(_ *http.Request) bool {
			return true // CORS 由 Gin 中间件处理
		},
	}
}

// parsePositiveInt 解析非负整数字符串
func parsePositiveInt(s string, result *int) (bool, error) {
	if s == "" {
		*result = 0
		return true, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return false, err
	}
	*result = n
	return true, nil
}
