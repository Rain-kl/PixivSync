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

package risk_control

import (
	"time"
)

// UserAccessLog 用户访问记录
type UserAccessLog struct {
	ID        uint64    `json:"id,string"`
	UserID    uint64    `json:"user_id,string"`
	Path      string    `json:"path"`
	Method    string    `json:"method"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Headers   string    `json:"headers"`
	Status    int32     `json:"status"`
	Latency   int64     `json:"latency"` // 耗时毫秒
	CreatedAt time.Time `json:"created_at"`
}
