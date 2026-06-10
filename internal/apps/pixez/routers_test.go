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

package pixez

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Rain-kl/Wavelet/internal/apps/oauth"
	"github.com/Rain-kl/Wavelet/internal/config"
	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/testhelper"
	"github.com/Rain-kl/Wavelet/internal/util"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

type testResponse struct {
	ErrorMsg string          `json:"error_msg"`
	Data     json.RawMessage `json:"data"`
}

func setupPixezTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	config.Config.App.SessionCookieName = "test_session_id"
	config.Config.App.SessionSecret = "test_session_secret"
	config.Config.App.SessionAge = 3600

	r := gin.New()
	store := cookie.NewStore([]byte(config.Config.App.SessionSecret))
	store.Options(util.GetSessionOptions(config.Config.App.SessionAge))
	r.Use(sessions.Sessions(config.Config.App.SessionCookieName, store))

	group := r.Group("/api/pixez")
	group.Use(oauth.LoginRequired())
	group.GET("/ping", Ping)
	group.GET("/users", ListUsers)
	group.GET("/users/:pixiv_user_id", GetUser)
	group.PUT("/users/:pixiv_user_id", UpsertUser)
	group.DELETE("/users/:pixiv_user_id", DeleteUser)
	group.GET("/users/:pixiv_user_id/sync-data", GetUserData)
	group.POST("/users/:pixiv_user_id/sync-data", PostUserData)
	group.GET("/users/:pixiv_user_id/sync-data/hashes", GetUserDataHashes)
	group.GET("/users/:pixiv_user_id/bookmarks/illust/removed", ListRemovedBookmarkIllusts)
	group.GET("/illusts/:illust_id/mirror", CheckIllustMirror)
	group.POST("/illusts/mirror/batch", BatchCheckIllustMirror)

	mirrorGroup := r.Group("/mirror")
	mirrorGroup.Use(oauth.LoginRequired())
	mirrorGroup.GET("/v1/illust/detail", GetMirroredIllustDetail)
	return r
}

func authHeader(t *testing.T, tokenName string) string {
	t.Helper()
	token, err := model.GenerateTokenString()
	if err != nil {
		t.Fatalf("GenerateTokenString() error = %v", err)
	}
	return seedAccessToken(t, tokenName, token)
}

func seedAccessToken(t *testing.T, name, token string) string {
	t.Helper()
	user := model.User{ID: 1001, Username: "admin", IsActive: true, IsAdmin: true}
	if err := db.DB(context.Background()).Create(&user).Error; err != nil {
		t.Fatalf("seed user failed: %v", err)
	}
	record := model.AccessToken{
		UserID:      user.ID,
		Name:        name,
		TokenHash:   model.HashToken(token),
		MaskedToken: model.MaskTokenString(token),
	}
	if err := db.DB(context.Background()).Create(&record).Error; err != nil {
		t.Fatalf("seed access token failed: %v", err)
	}
	return "Bearer " + token
}

func performJSON(router http.Handler, method, path string, body any, auth string) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func decodeResponse(t *testing.T, w *httptest.ResponseRecorder) testResponse {
	t.Helper()
	var resp testResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v. Body: %s", err, w.Body.String())
	}
	return resp
}

func TestPixezAuthRequired(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	router := setupPixezTestRouter()
	w := performJSON(router, http.MethodGet, "/api/pixez/ping", nil, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401. Body: %s", w.Code, w.Body.String())
	}
}

func TestPixezUserCRUD(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	router := setupPixezTestRouter()
	auth := authHeader(t, "pixez-test")

	payload := map[string]any{
		"name":               "Pixiv User",
		"account":            "pixiv_account",
		"mail_address":       "pixiv@example.com",
		"user_image":         "https://example.com/avatar.png",
		"access_token":       "access-token",
		"refresh_token":      "refresh-token",
		"device_token":       "device-token",
		"is_premium":         1,
		"x_restrict":         2,
		"is_mail_authorized": 1,
	}

	w := performJSON(router, http.MethodPut, "/api/pixez/users/12345", payload, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("upsert status = %d, want 200. Body: %s", w.Code, w.Body.String())
	}
	if resp := decodeResponse(t, w); resp.ErrorMsg != "" {
		t.Fatalf("upsert error_msg = %q", resp.ErrorMsg)
	}

	w = performJSON(router, http.MethodGet, "/api/pixez/users", nil, auth)
	resp := decodeResponse(t, w)
	if resp.ErrorMsg != "" {
		t.Fatalf("list error_msg = %q", resp.ErrorMsg)
	}
	if bytes.Contains(resp.Data, []byte("access-token")) || bytes.Contains(resp.Data, []byte("refresh-token")) {
		t.Fatalf("list response leaked token fields: %s", string(resp.Data))
	}

	var safeUsers []model.PixezPixivUserSafeDTO
	if err := json.Unmarshal(resp.Data, &safeUsers); err != nil {
		t.Fatalf("decode safe users failed: %v", err)
	}
	if len(safeUsers) != 1 || safeUsers[0].PixivUserID != "12345" {
		t.Fatalf("unexpected safe users: %+v", safeUsers)
	}

	w = performJSON(router, http.MethodGet, "/api/pixez/users/12345", nil, auth)
	resp = decodeResponse(t, w)
	var fullUser model.PixezPixivUser
	if err := json.Unmarshal(resp.Data, &fullUser); err != nil {
		t.Fatalf("decode full user failed: %v", err)
	}
	if fullUser.AccessToken != "access-token" || fullUser.RefreshToken != "refresh-token" {
		t.Fatalf("full user token fields missing: %+v", fullUser)
	}

	w = performJSON(router, http.MethodDelete, "/api/pixez/users/12345", nil, auth)
	if resp := decodeResponse(t, w); resp.ErrorMsg != "" {
		t.Fatalf("delete error_msg = %q", resp.ErrorMsg)
	}
}

func TestPixezSyncDataSelectiveReplaceAndHashes(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	router := setupPixezTestRouter()
	auth := authHeader(t, "pixez-sync-test")

	firstPayload := map[string]any{
		"ban_comments": []map[string]any{
			{"comment_id": "c2", "name": "Beta"},
			{"comment_id": "c1", "name": "Alpha"},
		},
		"ban_tags": []map[string]any{
			{"name": "tag1", "translate_name": "Tag One"},
		},
		"illust_histories": []map[string]any{
			{"illust_id": 10, "user_id": 20, "picture_url": "https://example.com/1.jpg", "title": "T", "user_name": "U", "time": 300},
		},
	}
	w := performJSON(router, http.MethodPost, "/api/pixez/users/12345/sync-data", firstPayload, auth)
	if resp := decodeResponse(t, w); resp.ErrorMsg != "" {
		t.Fatalf("first post error_msg = %q", resp.ErrorMsg)
	}

	replacePayload := map[string]any{
		"ban_comments": []map[string]any{
			{"comment_id": "c3", "name": "Gamma"},
		},
	}
	w = performJSON(router, http.MethodPost, "/api/pixez/users/12345/sync-data", replacePayload, auth)
	if resp := decodeResponse(t, w); resp.ErrorMsg != "" {
		t.Fatalf("replace post error_msg = %q", resp.ErrorMsg)
	}

	w = performJSON(router, http.MethodGet, "/api/pixez/users/12345/sync-data?tables=ban_comments,ban_tags", nil, auth)
	resp := decodeResponse(t, w)
	var data model.PixezUserDataPayload
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("decode sync data failed: %v", err)
	}
	if data.BanComments == nil || len(*data.BanComments) != 1 || (*data.BanComments)[0].CommentID != "c3" {
		t.Fatalf("ban_comments not selectively replaced: %+v", data.BanComments)
	}
	if data.BanTags == nil || len(*data.BanTags) != 1 {
		t.Fatalf("ban_tags should be preserved when omitted from replace payload: %+v", data.BanTags)
	}
	if data.IllustHistories != nil {
		t.Fatalf("illust_histories should be omitted by tables query: %+v", data.IllustHistories)
	}

	w = performJSON(router, http.MethodGet, "/api/pixez/users/12345/sync-data/hashes", nil, auth)
	resp = decodeResponse(t, w)
	var hashes map[string]string
	if err := json.Unmarshal(resp.Data, &hashes); err != nil {
		t.Fatalf("decode hashes failed: %v", err)
	}
	if hashes["ban_comments"] != "5aed06fb7b53efe853848522a18b6c79" {
		t.Fatalf("ban_comments hash = %q, want old-server compatible hash", hashes["ban_comments"])
	}
	if hashes["ban_illusts"] != "empty" {
		t.Fatalf("empty table hash = %q, want empty", hashes["ban_illusts"])
	}
}

func TestPixezMirrorAPIReturnsEnvelope(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()
	if err := db.DB(ctx).Create(&model.PixezMirrorIllust{
		IllustID:        123,
		TaskID:          "task-1",
		Status:          model.PixezMirrorStatusSuccess,
		ImageFilesJSON:  "[]",
		RequestURLsJSON: "[]",
		RetryURLsJSON:   "[]",
		TotalCount:      1,
		SuccessCount:    1,
	}).Error; err != nil {
		t.Fatalf("seed mirror record failed: %v", err)
	}

	router := setupPixezTestRouter()
	auth := authHeader(t, "pixez-mirror-status")
	w := performJSON(router, http.MethodGet, "/api/pixez/illusts/123/mirror", nil, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200. Body: %s", w.Code, w.Body.String())
	}
	resp := decodeResponse(t, w)
	if resp.ErrorMsg != "" {
		t.Fatalf("error_msg = %q", resp.ErrorMsg)
	}
	var status map[string]any
	if err := json.Unmarshal(resp.Data, &status); err != nil {
		t.Fatalf("decode mirror status failed: %v", err)
	}
	if status["status"] != model.PixezMirrorStatusSuccess || status["mirrored"] != true {
		t.Fatalf("unexpected mirror status: %+v", status)
	}
}

func TestMirrorRouteReturnsPixivShapeWithoutEnvelope(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()
	if err := db.DB(ctx).Create(&model.PixezMirrorIllust{
		IllustID:        123,
		Status:          model.PixezMirrorStatusSuccess,
		DetailJSON:      `{"illust":{"id":123,"image_urls":{"large":"https://i.pximg.net/img-original/img/2026/01/01/00/00/00/123_p0.jpg"}}}`,
		ImageFilesJSON:  "[]",
		RequestURLsJSON: "[]",
		RetryURLsJSON:   "[]",
		TotalCount:      1,
		SuccessCount:    1,
	}).Error; err != nil {
		t.Fatalf("seed mirror detail failed: %v", err)
	}

	router := setupPixezTestRouter()
	auth := authHeader(t, "pixez-mirror-read")
	w := performJSON(router, http.MethodGet, "/mirror/v1/illust/detail?illust_id=123", nil, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200. Body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if bytes.Contains(w.Body.Bytes(), []byte(`"error_msg"`)) || !bytes.Contains(w.Body.Bytes(), []byte(`"illust"`)) {
		t.Fatalf("mirror response should be raw Pixiv shape, got: %s", body)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`/mirror/pximg/img-original/img/2026/01/01/00/00/00/123_p0.jpg`)) {
		t.Fatalf("mirror response did not rewrite pximg URLs: %s", body)
	}
}
