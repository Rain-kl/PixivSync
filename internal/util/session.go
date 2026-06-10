// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"github.com/Rain-kl/Wavelet/internal/config"
	"github.com/gin-contrib/sessions"
)

// GetSessionOptions 根据配置构建 Session 选项
func GetSessionOptions(maxAge int) sessions.Options {
	return sessions.Options{
		Path:     "/",
		Domain:   config.Config.App.SessionDomain,
		MaxAge:   maxAge,
		HttpOnly: config.Config.App.SessionHTTPOnly,
		Secure:   config.Config.App.SessionSecure,
	}
}
