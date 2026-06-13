// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package pixez

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rain-kl/Wavelet/internal/apps/oauth"
	"github.com/Rain-kl/Wavelet/internal/config"
	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/diskcache"
	"github.com/Rain-kl/Wavelet/internal/model"
	pixezsvc "github.com/Rain-kl/Wavelet/internal/service/pixez"
	"github.com/Rain-kl/Wavelet/internal/storage"
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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
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
	group.GET("/dashboard", GetDashboard)
	group.GET("/bookmark-export-runs", ListBookmarkExportRuns)
	group.GET("/bookmarks/illusts", ListBookmarkIllusts)
	group.GET("/bookmarks/illusts/:illust_id/detail", GetBookmarkIllustDetail)
	group.GET("/bookmarks/novels", ListBookmarkNovels)
	group.GET("/bookmarks/novels/:novel_id/detail", GetBookmarkNovelDetail)
	group.GET("/users", ListUsers)
	group.POST("/users", AddUser)
	group.GET("/login-url", GetLoginURL)
	group.POST("/login-callback", LoginCallback)
	group.GET("/users/:pixiv_user_id", GetUser)
	group.GET("/users/:pixiv_user_id/profile", GetUserProfile)
	group.POST("/users/:pixiv_user_id/refresh-token", RefreshUserToken)
	group.PUT("/users/:pixiv_user_id", UpsertUser)
	group.DELETE("/users/:pixiv_user_id", DeleteUser)
	group.GET("/users/:pixiv_user_id/sync-data", GetUserData)
	group.POST("/users/:pixiv_user_id/sync-data", PostUserData)
	group.GET("/users/:pixiv_user_id/sync-data/hashes", GetUserDataHashes)
	group.GET("/users/:pixiv_user_id/bookmarks/illust/removed", ListRemovedBookmarkIllusts)
	group.GET("/illusts/:illust_id/mirror", CheckIllustMirror)
	group.POST("/illusts/mirror/batch", BatchCheckIllustMirror)
	group.GET("/mirror/illusts", ListMirroredIllusts)
	group.GET("/mirror/illusts/:illust_id/detail", GetMirroredIllustManagementDetail)
	group.GET("/mirror/novels", ListMirroredNovels)
	group.GET("/mirror/novels/:novel_id/detail", GetMirroredNovelManagementDetail)

	mirrorGroup := r.Group("/mirror")
	mirrorGroup.Use(oauth.LoginRequired())
	mirrorGroup.GET("/v1/illust/detail", GetMirroredIllustDetail)
	mirrorGroup.GET("/pximg/*path", ServeMirroredImage)
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

func TestPixezManagementDashboard(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	router := setupPixezTestRouter()
	auth := authHeader(t, "pixez-management-dashboard")
	seedPixezManagementData(t)

	w := performJSON(router, http.MethodGet, "/api/pixez/dashboard", nil, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("dashboard status = %d, want 200. Body: %s", w.Code, w.Body.String())
	}
	resp := decodeResponse(t, w)
	if resp.ErrorMsg != "" {
		t.Fatalf("dashboard error_msg = %q", resp.ErrorMsg)
	}

	var dashboard pixezDashboardResponse
	if err := json.Unmarshal(resp.Data, &dashboard); err != nil {
		t.Fatalf("decode dashboard data failed: %v. Body: %s", err, string(resp.Data))
	}
	if got, want := dashboard.Accounts, int64(1); got != want {
		t.Errorf("GetDashboard().Accounts = %d, want %d", got, want)
	}
	if got, want := dashboard.Illusts.Total, int64(3); got != want {
		t.Errorf("GetDashboard().Illusts.Total = %d, want %d", got, want)
	}
	if got, want := dashboard.Illusts.Succeeded, int64(1); got != want {
		t.Errorf("GetDashboard().Illusts.Succeeded = %d, want %d", got, want)
	}
	if got, want := dashboard.Queue.Running, int64(1); got != want {
		t.Errorf("GetDashboard().Queue.Running = %d, want %d", got, want)
	}
	if got, want := len(dashboard.RecentRuns), 1; got != want {
		t.Errorf("GetDashboard().RecentRuns length = %d, want %d", got, want)
	}
}

func TestPixezManagementBookmarkListAndDetail(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	router := setupPixezTestRouter()
	auth := authHeader(t, "pixez-management-bookmarks")
	seedPixezManagementData(t)

	w := performJSON(router, http.MethodGet, "/api/pixez/bookmarks/illusts?mirror_status=failed&q=Broken", nil, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("list illusts status = %d, want 200. Body: %s", w.Code, w.Body.String())
	}
	resp := decodeResponse(t, w)
	if resp.ErrorMsg != "" {
		t.Fatalf("list illusts error_msg = %q", resp.ErrorMsg)
	}
	var list struct {
		Items []pixezIllustBookmarkDTO `json:"items"`
		Total int64                    `json:"total"`
	}
	if err := json.Unmarshal(resp.Data, &list); err != nil {
		t.Fatalf("decode illust list failed: %v. Body: %s", err, string(resp.Data))
	}
	if got, want := list.Total, int64(1); got != want {
		t.Errorf("ListBookmarkIllusts(total) = %d, want %d", got, want)
	}
	if got, want := list.Items[0].IllustID, int64(1002); got != want {
		t.Errorf("ListBookmarkIllusts(items[0].IllustID) = %d, want %d", got, want)
	}
	if list.Items[0].CoverURL == "" {
		t.Error("ListBookmarkIllusts(items[0].CoverURL) is empty, want cover URL from illust_json")
	}

	w = performJSON(router, http.MethodGet, "/api/pixez/bookmarks/illusts/1001/detail", nil, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("illust detail status = %d, want 200. Body: %s", w.Code, w.Body.String())
	}
	resp = decodeResponse(t, w)
	if resp.ErrorMsg != "" {
		t.Fatalf("illust detail error_msg = %q", resp.ErrorMsg)
	}
	var detail pixezIllustBookmarkDetailDTO
	if err := json.Unmarshal(resp.Data, &detail); err != nil {
		t.Fatalf("decode illust detail failed: %v. Body: %s", err, string(resp.Data))
	}
	if detail.Mirror == nil {
		t.Fatal("GetBookmarkIllustDetail().Mirror is nil, want mirror diagnostics")
	}
	if got, want := len(detail.ImageFiles), 1; got != want {
		t.Errorf("GetBookmarkIllustDetail().ImageFiles length = %d, want %d", got, want)
	}
	if got, want := len(detail.RetryURLs), 1; got != want {
		t.Errorf("GetBookmarkIllustDetail().RetryURLs length = %d, want %d", got, want)
	}
}

func seedPixezManagementData(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().Add(-2 * time.Minute)
	finishedAt := now.Add(90 * time.Second)
	pixivUserID := "98765432"

	user := model.PixezPixivUser{
		PixivUserID:  pixivUserID,
		Name:         "Pixiv Admin",
		Account:      "pixiv_admin",
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := db.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("seed PixezPixivUser failed: %v", err)
	}

	illusts := []model.PixezBookmarkIllust{
		{
			PixivUserID:      pixivUserID,
			Restrict:         "public",
			IllustID:         1001,
			Title:            "Finished Illust",
			UserID:           2001,
			UserName:         "Painter",
			PageCount:        2,
			Visible:          true,
			IllustJSON:       `{"image_urls":{"square_medium":"https://i.pximg.net/c/360x360/1001.jpg"}}`,
			LastExportRunID:  "pixez_illust_run_1",
			LastSeenAt:       now,
			MirrorStatus:     model.PixezBookmarkMirrorDone,
			MirrorRetryCount: 1,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			PixivUserID:      pixivUserID,
			Restrict:         "public",
			IllustID:         1002,
			Title:            "Broken Illust",
			UserID:           2002,
			UserName:         "Painter",
			PageCount:        1,
			Visible:          true,
			IllustJSON:       `{"image_urls":{"square_medium":"https://i.pximg.net/c/360x360/1002.jpg"}}`,
			LastExportRunID:  "pixez_illust_run_1",
			LastSeenAt:       now,
			MirrorStatus:     model.PixezBookmarkMirrorFailed,
			MirrorRetryCount: 2,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			PixivUserID:     pixivUserID,
			Restrict:        "public",
			IllustID:        1003,
			Title:           "Queued Illust",
			UserID:          2003,
			UserName:        "Painter",
			PageCount:       1,
			Visible:         true,
			IllustJSON:      `{"image_urls":{"square_medium":"https://i.pximg.net/c/360x360/1003.jpg"}}`,
			LastExportRunID: "pixez_illust_run_1",
			LastSeenAt:      now,
			MirrorStatus:    model.PixezBookmarkMirrorProcessing,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
	}
	if err := db.DB(ctx).Create(&illusts).Error; err != nil {
		t.Fatalf("seed PixezBookmarkIllust failed: %v", err)
	}

	novel := model.PixezBookmarkNovel{
		PixivUserID:     pixivUserID,
		Restrict:        "public",
		NovelID:         3001,
		Title:           "Novel Title",
		UserID:          4001,
		UserName:        "Writer",
		TextLength:      12000,
		Visible:         true,
		CoverURL:        "https://i.pximg.net/c/240x480/3001.jpg",
		NovelJSON:       `{"image_urls":{"medium":"https://i.pximg.net/c/240x480/3001.jpg"}}`,
		LastExportRunID: "pixez_novel_run_1",
		LastSeenAt:      now,
		MirrorStatus:    model.PixezBookmarkMirrorNone,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.DB(ctx).Create(&novel).Error; err != nil {
		t.Fatalf("seed PixezBookmarkNovel failed: %v", err)
	}

	mirror := model.PixezMirrorIllust{
		IllustID:        1001,
		TaskID:          "task_1001",
		Status:          model.PixezMirrorStatusSuccess,
		ImageFilesJSON:  `[{"pixiv_url":"https://i.pximg.net/img-original/1001_p0.jpg","page":0,"upload_id":"1","file_name":"1001_p0.jpg","hash":"sha256","mime":"image/jpeg","size":1234,"storage_key":"uploads/1001.jpg"}]`,
		RequestURLsJSON: `["https://i.pximg.net/img-original/1001_p0.jpg","https://i.pximg.net/img-original/1001_p1.jpg"]`,
		RetryURLsJSON:   `["https://i.pximg.net/img-original/1001_p1.jpg"]`,
		TotalCount:      2,
		SuccessCount:    1,
		FailedCount:     1,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.DB(ctx).Create(&mirror).Error; err != nil {
		t.Fatalf("seed PixezMirrorIllust failed: %v", err)
	}

	run := model.PixezBookmarkExportRun{
		ID:           "pixez_illust_run_1",
		TargetType:   model.PixezMirrorTargetIllust,
		PixivUserID:  pixivUserID,
		Restrict:     "public",
		Status:       model.PixezBookmarkExportStatusSuccess,
		TotalCount:   3,
		NewCount:     2,
		UpdatedCount: 1,
		RemovedCount: 0,
		StartedAt:    now,
		FinishedAt:   &finishedAt,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := db.DB(ctx).Create(&run).Error; err != nil {
		t.Fatalf("seed PixezBookmarkExportRun failed: %v", err)
	}

	executions := []model.TaskExecution{
		{
			TaskID:      "pixez_running",
			TaskType:    "pixez_mirror",
			TaskName:    "PixEz 镜像资源",
			Status:      model.TaskExecutionStatusRunning,
			TriggeredBy: "manual",
			StartedAt:   &now,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			TaskID:      "pixez_pending",
			TaskType:    "pixez_export_bookmarks",
			TaskName:    "PixEz 导出收藏",
			Status:      model.TaskExecutionStatusPending,
			TriggeredBy: "manual",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}
	for i := range executions {
		if err := model.CreateTaskExecution(ctx, &executions[i]); err != nil {
			t.Fatalf("seed TaskExecution(%s) failed: %v", executions[i].TaskID, err)
		}
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

func TestMirrorManagementIncludesNonBookmarkMirrors(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()
	records := []model.PixezMirrorIllust{
		{
			IllustID:        501,
			TaskID:          "bookmark-triggered",
			Status:          model.PixezMirrorStatusSuccess,
			DetailJSON:      `{"illust":{"id":501,"title":"Bookmarked","user":{"id":601,"name":"Artist A"},"image_urls":{"square_medium":"https://i.pximg.net/501.jpg"},"page_count":1,"visible":true}}`,
			ImageFilesJSON:  "[]",
			RequestURLsJSON: "[]",
			RetryURLsJSON:   "[]",
			TotalCount:      1,
			SuccessCount:    1,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			IllustID:        502,
			TaskID:          "manual-triggered",
			Status:          model.PixezMirrorStatusSuccess,
			DetailJSON:      `{"illust":{"id":502,"title":"Manual Mirror","user":{"id":602,"name":"Artist B"},"image_urls":{"square_medium":"https://i.pximg.net/502.jpg"},"page_count":1,"visible":true}}`,
			ImageFilesJSON:  "[]",
			RequestURLsJSON: "[]",
			RetryURLsJSON:   "[]",
			TotalCount:      1,
			SuccessCount:    1,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
	}
	if err := db.DB(ctx).Create(&records).Error; err != nil {
		t.Fatalf("seed mirror records failed: %v", err)
	}
	if err := db.DB(ctx).Create(&model.PixezBookmarkIllust{
		PixivUserID:     "123",
		Restrict:        "public",
		IllustID:        501,
		Title:           "Bookmarked",
		IllustJSON:      records[0].DetailJSON,
		LastExportRunID: "run-1",
		LastSeenAt:      now,
	}).Error; err != nil {
		t.Fatalf("seed bookmark record failed: %v", err)
	}

	router := setupPixezTestRouter()
	auth := authHeader(t, "pixez-mirror-management")
	w := performJSON(router, http.MethodGet, "/api/pixez/mirror/illusts", nil, auth)
	resp := decodeResponse(t, w)
	if resp.ErrorMsg != "" {
		t.Fatalf("list mirror records error_msg = %q", resp.ErrorMsg)
	}
	var list struct {
		Items []pixezMirroredIllustDTO `json:"items"`
		Total int64                    `json:"total"`
	}
	if err := json.Unmarshal(resp.Data, &list); err != nil {
		t.Fatalf("decode mirror list failed: %v", err)
	}
	if got, want := list.Total, int64(2); got != want {
		t.Errorf("ListMirroredIllusts().Total = %d, want %d", got, want)
	}

	w = performJSON(router, http.MethodGet, "/api/pixez/mirror/illusts/502/detail", nil, auth)
	resp = decodeResponse(t, w)
	if resp.ErrorMsg != "" {
		t.Fatalf("get manual mirror detail error_msg = %q", resp.ErrorMsg)
	}
	var detail pixezMirroredIllustDetailDTO
	if err := json.Unmarshal(resp.Data, &detail); err != nil {
		t.Fatalf("decode manual mirror detail failed: %v", err)
	}
	if got, want := detail.Item.Title, "Manual Mirror"; got != want {
		t.Errorf("GetMirroredIllustManagementDetail().Item.Title = %q, want %q", got, want)
	}

	novel := model.PixezMirrorNovel{
		NovelID:         701,
		TaskID:          "automatic-triggered",
		Status:          model.PixezMirrorStatusSuccess,
		DetailJSON:      `{"novel":{"id":701,"title":"Automatic Novel","user":{"id":801,"name":"Writer"},"image_urls":{"medium":"https://i.pximg.net/701.jpg"},"text_length":2048,"x_restrict":1,"is_original":true,"visible":true}}`,
		TextJSON:        `{"text":"mirrored novel"}`,
		RequestURLsJSON: "[]",
		RetryURLsJSON:   "[]",
		TotalCount:      1,
		SuccessCount:    1,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.DB(ctx).Create(&novel).Error; err != nil {
		t.Fatalf("seed non-bookmark novel mirror failed: %v", err)
	}

	w = performJSON(router, http.MethodGet, "/api/pixez/mirror/novels", nil, auth)
	resp = decodeResponse(t, w)
	if resp.ErrorMsg != "" {
		t.Fatalf("list novel mirror records error_msg = %q", resp.ErrorMsg)
	}
	var novelList struct {
		Items []pixezMirroredNovelDTO `json:"items"`
		Total int64                   `json:"total"`
	}
	if err := json.Unmarshal(resp.Data, &novelList); err != nil {
		t.Fatalf("decode novel mirror list failed: %v", err)
	}
	if got, want := novelList.Total, int64(1); got != want {
		t.Errorf("ListMirroredNovels().Total = %d, want %d", got, want)
	}

	w = performJSON(router, http.MethodGet, "/api/pixez/mirror/novels/701/detail", nil, auth)
	resp = decodeResponse(t, w)
	if resp.ErrorMsg != "" {
		t.Fatalf("get automatic novel mirror detail error_msg = %q", resp.ErrorMsg)
	}
	var novelDetail pixezMirroredNovelDetailDTO
	if err := json.Unmarshal(resp.Data, &novelDetail); err != nil {
		t.Fatalf("decode automatic novel mirror detail failed: %v", err)
	}
	if got, want := novelDetail.Item.Title, "Automatic Novel"; got != want {
		t.Errorf("GetMirroredNovelManagementDetail().Item.Title = %q, want %q", got, want)
	}
	if !novelDetail.Item.IsOriginal || novelDetail.Item.XRestrict != 1 {
		t.Errorf("GetMirroredNovelManagementDetail().Item metadata = %+v, want original R-18 novel", novelDetail.Item)
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

func TestServeMirroredImageQuality(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	cache := diskcache.GetGlobalCache()
	if err := cache.Clear(); err != nil {
		t.Fatalf("clear disk cache before test: %v", err)
	}
	t.Cleanup(func() {
		if err := cache.Clear(); err != nil {
			t.Errorf("clear disk cache after test: %v", err)
		}
		if err := os.RemoveAll("uploads"); err != nil {
			t.Errorf("remove test upload cache: %v", err)
		}
	})

	var imageBytes bytes.Buffer
	source := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := range 8 {
		for x := range 8 {
			source.Set(x, y, color.RGBA{R: uint8(x * 20), G: uint8(y * 20), B: 100, A: 255})
		}
	}
	if err := png.Encode(&imageBytes, source); err != nil {
		t.Fatalf("encode test image failed: %v", err)
	}
	ctx := context.Background()
	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "76228932_p0.png")
	if err := os.WriteFile(imagePath, imageBytes.Bytes(), 0644); err != nil {
		t.Fatalf("write test image failed: %v", err)
	}

	cfg := storage.DefaultConfig()
	cfg.Local.Root = tempDir
	if err := storage.SaveActiveConfig(ctx, cfg); err != nil {
		t.Fatalf("save active config failed: %v", err)
	}

	upload := model.Upload{
		ID:            76228932,
		UserID:        1001,
		FileName:      "76228932_p0.png",
		FilePath:      "76228932_p0.png",
		FileSize:      int64(imageBytes.Len()),
		MimeType:      "image/png",
		Extension:     "png",
		StorageDriver: "local",
		Type:          "pixez_mirror",
		Status:        model.UploadStatusUsed,
		AccessMode:    1,
	}
	if err := db.DB(ctx).Create(&upload).Error; err != nil {
		t.Fatalf("seed mirror upload failed: %v", err)
	}
	files := []model.PixezMirrorImageFile{{
		PixivURL: "https://i.pximg.net/img-original/img/2019/08/13/06/45/48/76228932_p0.png",
		Page:     0,
		UploadID: upload.ID,
		FileName: upload.FileName,
	}}
	if err := db.DB(ctx).Create(&model.PixezMirrorIllust{
		IllustID:        76228932,
		Status:          model.PixezMirrorStatusSuccess,
		ImageFilesJSON:  string(mustMarshalJSON(t, files)),
		RequestURLsJSON: "[]",
		RetryURLsJSON:   "[]",
		SuccessCount:    1,
		TotalCount:      1,
	}).Error; err != nil {
		t.Fatalf("seed mirror record failed: %v", err)
	}

	router := setupPixezTestRouter()
	auth := authHeader(t, "pixez-mirror-image-quality")
	path := "/mirror/pximg/c/360x360_70/img-master/img/2019/08/13/06/45/48/76228932_p0_square1200.png"

	original := performJSON(router, http.MethodGet, path, nil, auth)
	if original.Code != http.StatusOK {
		t.Fatalf("ServeMirroredImage(origin) status = %d, want 200", original.Code)
	}
	if got, want := original.Header().Get("Content-Type"), "image/png"; got != want {
		t.Errorf("ServeMirroredImage(origin) Content-Type = %q, want %q", got, want)
	}

	medium := performJSON(router, http.MethodGet, path+"?quality=medium", nil, auth)
	if medium.Code != http.StatusOK {
		t.Fatalf("ServeMirroredImage(medium) status = %d, want 200. Body: %s", medium.Code, medium.Body.String())
	}
	if got, want := medium.Header().Get("Content-Type"), "image/webp"; got != want {
		t.Errorf("ServeMirroredImage(medium) Content-Type = %q, want %q", got, want)
	}
	if bytes.Equal(medium.Body.Bytes(), imageBytes.Bytes()) {
		t.Error("ServeMirroredImage(medium) returned original bytes, want WebP conversion")
	}
}

func mustMarshalJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal test JSON failed: %v", err)
	}
	return data
}

func TestPixezLoginEndpoints(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	router := setupPixezTestRouter()
	auth := authHeader(t, "pixez-login-test")

	// 1. Test GET /login-url
	w := performJSON(router, http.MethodGet, "/api/pixez/login-url", nil, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("GetLoginURL status = %d, want 200", w.Code)
	}
	resp := decodeResponse(t, w)
	if resp.ErrorMsg != "" {
		t.Fatalf("GetLoginURL error_msg = %q", resp.ErrorMsg)
	}
	var loginData struct {
		CodeVerifier string `json:"code_verifier"`
		LoginURL     string `json:"login_url"`
	}
	if err := json.Unmarshal(resp.Data, &loginData); err != nil {
		t.Fatalf("decode login URL data failed: %v", err)
	}
	if len(loginData.CodeVerifier) != 128 {
		t.Errorf("verifier length = %d, want 128", len(loginData.CodeVerifier))
	}
	if !strings.Contains(loginData.LoginURL, "code_challenge=") || !strings.Contains(loginData.LoginURL, "code_challenge_method=S256") {
		t.Errorf("unexpected login URL: %s", loginData.LoginURL)
	}

	// 2. Test POST /login-callback
	oldTransport := pixezsvc.DefaultClient.HTTPClient.Transport
	defer func() {
		pixezsvc.DefaultClient.HTTPClient.Transport = oldTransport
	}()
	pixezsvc.DefaultClient.HTTPClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "oauth.secure.pixiv.net" && req.URL.Path == "/auth/token" {
			return jsonResponse(http.StatusOK, `{
				"response": {
					"access_token": "mock-access-cb",
					"refresh_token": "mock-refresh-cb",
					"user": {
						"id": "22222",
						"name": "Callback User",
						"account": "cb_account",
						"mail_address": "cb@example.com",
						"profile_image_urls": {
							"px_170x170": "https://example.com/cb_avatar.png"
						},
						"is_premium": false,
						"x_restrict": 0,
						"is_mail_authorized": true
					}
				}
			}`), nil
		}
		t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		return nil, nil
	})

	// Test pasting full custom URL callback
	callbackPayload := map[string]any{
		"code":          "pixiv://account?code=mock-code-value",
		"code_verifier": loginData.CodeVerifier,
	}
	w = performJSON(router, http.MethodPost, "/api/pixez/login-callback", callbackPayload, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("LoginCallback status = %d, want 200. Body: %s", w.Code, w.Body.String())
	}
	resp = decodeResponse(t, w)
	if resp.ErrorMsg != "" {
		t.Fatalf("LoginCallback error_msg = %q", resp.ErrorMsg)
	}

	var safeUser model.PixezPixivUserSafeDTO
	if err := json.Unmarshal(resp.Data, &safeUser); err != nil {
		t.Fatalf("decode safe user failed: %v", err)
	}
	if safeUser.PixivUserID != "22222" || safeUser.Name != "Callback User" {
		t.Fatalf("unexpected safe user: %+v", safeUser)
	}

	// Verify persistence in DB
	var dbUser model.PixezPixivUser
	if err := db.DB(context.Background()).Where("pixiv_user_id = ?", "22222").First(&dbUser).Error; err != nil {
		t.Fatalf("failed to find user in DB: %v", err)
	}
	if dbUser.AccessToken != "mock-access-cb" || dbUser.RefreshToken != "mock-refresh-cb" {
		t.Fatalf("persisted tokens incorrect: %+v", dbUser)
	}
}

func TestExtractCode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"my-raw-code-value", "my-raw-code-value"},
		{"pixiv://account?code=mock-code-value", "mock-code-value"},
		{"https://app-api.pixiv.net/web/v1/users/auth/pixiv/callback?code=another-code", "another-code"},
		{"https://accounts.pixiv.net/post-redirect?return_to=https%3A%2F%2Fapp-api.pixiv.net%2Fweb%2Fv1%2Fusers%2Fauth%2Fpixiv%2Fstart%3Fcode_challenge%3DUJkziWHJaZK462K9HBRvE4jM2vLcDXW_RflVPu4ygVI%26code_challenge_method%3DS256%26client%3Dpixiv-android%26via%3Dlogin", ""},
		{"https://some-other-url.com/index.html", ""},
		{"pixiv://account?foo=bar&code=xyz123&test=1", "xyz123"},
	}

	for _, tc := range tests {
		got := extractCode(tc.input)
		if got != tc.expected {
			t.Errorf("extractCode(%q) = %q; want %q", tc.input, got, tc.expected)
		}
	}
}
