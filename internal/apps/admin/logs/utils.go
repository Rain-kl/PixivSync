// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

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
