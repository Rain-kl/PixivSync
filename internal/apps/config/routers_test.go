/*
Copyright 2026 linux.do
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

package config

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/testhelper"
	"github.com/Rain-kl/Wavelet/internal/util"
	"github.com/gin-gonic/gin"
)

func TestGetPublicConfigUsesVisibility(t *testing.T) {
	dbConn, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	if err := dbConn.Create(&model.SystemConfig{
		Key:         "custom_public_key",
		Value:       "custom_public_value",
		Type:        "system",
		Visibility:  model.ConfigVisibilityVisible,
		Description: "custom public config",
	}).Error; err != nil {
		t.Fatalf("Create(custom_public_key) error = %v", err)
	}
	if err := dbConn.Model(&model.SystemConfig{}).
		Where("key = ?", model.ConfigKeySiteName).
		Update("visibility", model.ConfigVisibilityHidden).Error; err != nil {
		t.Fatalf("Update(%s.visibility) error = %v", model.ConfigKeySiteName, err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/config/public", GetPublicConfig)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/config/public", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GetPublicConfig() status = %d, want %d; body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp util.ResponseAny
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal(GetPublicConfig()) error = %v", err)
	}
	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		t.Fatalf("json.Marshal(GetPublicConfig().data) error = %v", err)
	}
	var configs map[string]string
	if err := json.Unmarshal(dataBytes, &configs); err != nil {
		t.Fatalf("json.Unmarshal(GetPublicConfig().data) error = %v", err)
	}

	if got := configs["custom_public_key"]; got != "custom_public_value" {
		t.Errorf("GetPublicConfig()[custom_public_key] = %q, want %q", got, "custom_public_value")
	}
	if _, ok := configs[model.ConfigKeySiteName]; ok {
		t.Errorf("GetPublicConfig()[%s] is present, want hidden", model.ConfigKeySiteName)
	}
}
