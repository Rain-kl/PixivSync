// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package pixez_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Rain-kl/Wavelet/internal/apps/pixez"
	"github.com/Rain-kl/Wavelet/internal/bootstrap"
	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/router/root"
	pixezsvc "github.com/Rain-kl/Wavelet/internal/service/pixez"
	"github.com/Rain-kl/Wavelet/internal/task"
	"github.com/Rain-kl/Wavelet/internal/testhelper"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/hibiken/asynq"
)

type artistTransport func(*http.Request) (*http.Response, error)

func (f artistTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestArtistSubscriptionAPIQueuesScansAndKeepsCategoriesIndependent(t *testing.T) {
	_, redis, cleanup := testhelper.SetupTestEnvironment(t)
	t.Cleanup(cleanup)
	t.Chdir(t.TempDir())
	bootstrap.RegisterTasks()
	previousQueue := task.AsynqClient
	task.AsynqClient = asynq.NewClient(asynq.RedisClientOpt{Addr: redis.Addr()})
	t.Cleanup(func() {
		if err := task.AsynqClient.Close(); err != nil {
			t.Error(err)
		}
		task.AsynqClient = previousQueue
	})
	ctx := context.Background()
	for _, row := range []any{
		&model.User{ID: 1001, Username: "admin", IsActive: true},
		&model.AccessToken{UserID: 1001, Name: "subscription-test", TokenHash: model.HashToken("subscription-test-token")},
		&model.PixezPixivUser{PixivUserID: "100", AccessToken: "test-pixiv-token"},
	} {
		if err := db.DB(ctx).Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	var novel pixezsvc.Novel
	novel.ID, novel.User.ID, novel.User.Name, novel.Title = 99, 7, "作者", "收藏小说"
	if err := pixezsvc.IndexNovel(ctx, novel, true, ""); err != nil {
		t.Fatal(err)
	}
	router := testhelper.NewTestGinEngine()
	router.Use(sessions.Sessions("subscription-test", cookie.NewStore([]byte("test-session-secret"))))
	root.RegisterCustomRootRoutes(router)
	request := func(method, path, body string, loggedIn bool, expected int) json.RawMessage {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		if loggedIn {
			r.Header.Set("Authorization", "Bearer subscription-test-token")
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != expected {
			t.Fatalf("%s %s: status=%d body=%s", method, path, w.Code, w.Body.String())
		}
		var result struct {
			Error string          `json:"error_msg"`
			Data  json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if expected == http.StatusOK && result.Error != "" {
			t.Fatal(result.Error)
		}
		return result.Data
	}
	request(http.MethodGet, "/api/pixez/artists", "", false, http.StatusUnauthorized)
	request(http.MethodPut, "/api/pixez/artists/7/subscriptions/novel", `{}`, true, http.StatusBadRequest)
	request(http.MethodGet, "/api/pixez/artists?type=unsupported", "", true, http.StatusBadRequest)
	request(http.MethodPut, "/api/pixez/artists/7/subscriptions/novel", `{"enabled":true}`, true, http.StatusOK)
	request(http.MethodPut, "/api/pixez/artists/7/subscriptions/novel", `{"enabled":true}`, true, http.StatusOK)
	var scans []model.TaskExecution
	if err := db.DB(ctx).Where("task_type = ?", pixez.PixezArtistScanTask).Find(&scans).Error; err != nil {
		t.Fatal(err)
	}
	if len(scans) != 1 {
		t.Fatalf("repeat enable queued %d scans", len(scans))
	}
	previousClient := pixezsvc.DefaultClient
	pixezsvc.DefaultClient = pixezsvc.NewClient(&http.Client{Transport: artistTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1/user/novels" || r.Header.Get("Authorization") == "" {
			t.Fatalf("unexpected scan request: %s", r.URL.Path)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"novels":[{"id":100,"title":"新小说","user":{"id":7,"name":"作者"}}],"next_url":null}`))}, nil
	})})
	t.Cleanup(func() { pixezsvc.DefaultClient = previousClient })
	result, err := (&pixez.ArtistScanTaskHandler{}).Execute(ctx, []byte(scans[0].Payload))
	if err != nil || result == nil {
		t.Fatal(result, err)
	}
	var mirrors int64
	if err := db.DB(ctx).Model(&model.TaskExecution{}).Where("task_type = ?", pixez.PixezMirrorTask).Count(&mirrors).Error; err != nil {
		t.Fatal(err)
	}
	if mirrors != 1 {
		t.Fatalf("expected one missing-work backup task, got %d", mirrors)
	}
	var page struct {
		Total   int64             `json:"total"`
		Results []json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(request(http.MethodGet, "/api/pixez/artists/7/works?type=novel", "", true, http.StatusOK), &page); err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || len(page.Results) != 2 {
		t.Fatal(page)
	}
	request(http.MethodPut, "/api/pixez/artists/7/subscriptions/illust", `{"enabled":true}`, true, http.StatusOK)
	request(http.MethodPut, "/api/pixez/artists/7/subscriptions/novel", `{"enabled":false}`, true, http.StatusOK)
	request(http.MethodPost, "/api/pixez/artists/7/subscriptions/novel/sync", "", true, http.StatusBadRequest)
	subs, err := pixezsvc.ArtistSubscriptions(ctx, 7)
	if err != nil || len(subs) != 2 {
		t.Fatal(subs, err)
	}
	for _, sub := range subs {
		if sub.Enabled != (sub.TargetType == model.PixezMirrorTargetIllust) {
			t.Fatal("category switches coupled", subs)
		}
	}
}
