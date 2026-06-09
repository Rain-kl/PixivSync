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

package oauth

import (
	"encoding/json"
	"time"
)

// Session 用户信息字段 Key
const (
	UserNameKey                     = "username"
	UserIDKey                       = "user_id"
	UserObjKey                      = "user_obj"
	PendingOAuthSourceIDKey         = "pending_oauth_source_id"
	PendingOAuthExternalIDKey       = "pending_oauth_external_id"
	PendingOAuthExternalUsernameKey = "pending_oauth_external_username"
	PendingOAuthEmailKey            = "pending_oauth_email"
)

// OAuth State 缓存 Key 格式与过期时间
const (
	OAuthStateCacheKeyFormat     = "oauth:state:%s"
	OAuthStateCacheKeyExpiration = 10 * time.Minute
)

// OAuth 授权用途常量
const (
	OAuthPurposeLogin = "login"
	OAuthPurposeBind  = "bind"
)

type oauthStatePayload struct {
	SourceName string `json:"source_name"`
	Purpose    string `json:"purpose"`
}

func encodeOAuthStatePayload(payload oauthStatePayload) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func decodeOAuthStatePayload(value string) (oauthStatePayload, error) {
	var payload oauthStatePayload
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		return oauthStatePayload{}, err
	}
	return payload, nil
}
