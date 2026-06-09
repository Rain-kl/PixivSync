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

package util

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// StringArray custom type for handling JSON arrays
type StringArray []string

// Scan 实现 sql.Scanner 接口，从数据库读取 JSON 数组
func (sa *StringArray) Scan(value interface{}) error {
	bytesValue, ok := value.([]byte)
	if !ok {
		return fmt.Errorf(errInvalidCustomValue, value)
	}
	return json.Unmarshal(bytesValue, sa)
}

// Value 实现 driver.Valuer 接口，将 JSON 数组序列化为数据库存储值
func (sa StringArray) Value() (driver.Value, error) {
	return json.Marshal(sa)
}
