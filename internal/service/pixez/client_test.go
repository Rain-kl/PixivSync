// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package pixez

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/testhelper"
)

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

func TestGetIllustDetailRefreshesToken(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()
	user := model.PixezPixivUser{
		PixivUserID:  "100",
		Name:         "Pixiv User",
		Account:      "pixiv_user",
		AccessToken:  "old-token",
		RefreshToken: "old-refresh",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := db.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("seed pixiv user failed: %v", err)
	}

	var seenAuth []string
	client := NewClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Host {
		case pixivAPIHost:
			seenAuth = append(seenAuth, req.Header.Get("Authorization"))
			if req.Header.Get("Authorization") == "Bearer old-token" {
				return jsonResponse(http.StatusUnauthorized, `{"error":{"message":"expired"}}`), nil
			}
			return jsonResponse(http.StatusOK, `{"illust":{"id":123,"title":"fresh"}}`), nil
		case "oauth.secure.pixiv.net":
			return jsonResponse(http.StatusOK, `{"response":{"access_token":"new-token","refresh_token":"new-refresh"}}`), nil
		default:
			t.Fatalf("unexpected request host: %s", req.URL.Host)
			return nil, nil
		}
	})})

	raw, detail, err := client.GetIllustDetail(ctx, user, 123)
	if err != nil {
		t.Fatalf("GetIllustDetail() error = %v", err)
	}
	if detail.Illust.ID != 123 || !strings.Contains(string(raw), `"fresh"`) {
		t.Fatalf("unexpected detail payload: id=%d raw=%s", detail.Illust.ID, string(raw))
	}
	if len(seenAuth) != 2 || seenAuth[0] != "Bearer old-token" || seenAuth[1] != "Bearer new-token" {
		t.Fatalf("unexpected auth sequence: %#v", seenAuth)
	}

	var refreshed model.PixezPixivUser
	if err := db.DB(ctx).Where("pixiv_user_id = ?", "100").First(&refreshed).Error; err != nil {
		t.Fatalf("load refreshed user failed: %v", err)
	}
	if refreshed.AccessToken != "new-token" || refreshed.RefreshToken != "new-refresh" {
		t.Fatalf("token refresh not persisted: %+v", refreshed)
	}
}

func TestPixivPlaceholderAndImageURLCollection(t *testing.T) {
	var illust Illust
	illust.ID = 1
	illust.ImageUrls.SquareMedium = "https://i.pximg.net/c/360x360/limit_unknown_360.png"
	if !IsLimitUnknownIllust(illust) {
		t.Fatal("IsLimitUnknownIllust() = false, want true")
	}

	var novel BookmarkNovel
	novel.ID = 2
	novel.ImageUrls.Medium = "https://i.pximg.net/c/240x480/limit_unknown_100.png"
	if !IsLimitUnknownNovel(novel) {
		t.Fatal("IsLimitUnknownNovel() = false, want true")
	}

	detail := IllustDetail{}
	detail.Illust.MetaSinglePage.OriginalImageURL = "https://i.pximg.net/img-original/1_p0.jpg"
	detail.Illust.MetaPages = append(detail.Illust.MetaPages, struct {
		ImageUrls struct {
			SquareMedium string `json:"square_medium"`
			Medium       string `json:"medium"`
			Large        string `json:"large"`
			Original     string `json:"original"`
		} `json:"image_urls"`
	}{})
	detail.Illust.MetaPages[0].ImageUrls.Original = "https://i.pximg.net/img-original/1_p0.jpg"
	detail.Illust.MetaPages = append(detail.Illust.MetaPages, struct {
		ImageUrls struct {
			SquareMedium string `json:"square_medium"`
			Medium       string `json:"medium"`
			Large        string `json:"large"`
			Original     string `json:"original"`
		} `json:"image_urls"`
	}{})
	detail.Illust.MetaPages[1].ImageUrls.Original = "https://i.pximg.net/img-original/1_p1.jpg"

	urls := CollectIllustImageURLs(detail)
	if len(urls) != 2 || urls[0] != "https://i.pximg.net/img-original/1_p0.jpg" || urls[1] != "https://i.pximg.net/img-original/1_p1.jpg" {
		t.Fatalf("CollectIllustImageURLs() = %#v", urls)
	}
}

func TestAddPixivUserByRefreshToken(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()
	client := NewClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "oauth.secure.pixiv.net" && req.URL.Path == "/auth/token" {
			return jsonResponse(http.StatusOK, `{
				"response": {
					"access_token": "mock-access",
					"refresh_token": "mock-refresh",
					"user": {
						"id": "12345",
						"name": "Manually Added User",
						"account": "manual_account",
						"mail_address": "manual@example.com",
						"profile_image_urls": {
							"px_170x170": "https://example.com/avatar_manual.png"
						},
						"is_premium": true,
						"x_restrict": 1,
						"is_mail_authorized": true
					}
				}
			}`), nil
		}
		t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		return nil, nil
	})})

	user, err := client.AddPixivUserByRefreshToken(ctx, "input-refresh-token")
	if err != nil {
		t.Fatalf("AddPixivUserByRefreshToken() error = %v", err)
	}

	if user.PixivUserID != "12345" || user.Name != "Manually Added User" || user.AccessToken != "mock-access" || user.RefreshToken != "mock-refresh" {
		t.Fatalf("unexpected returned user: %+v", user)
	}

	// Verify persistence
	var dbUser model.PixezPixivUser
	if err := db.DB(ctx).Where("pixiv_user_id = ?", "12345").First(&dbUser).Error; err != nil {
		t.Fatalf("failed to find user in DB: %v", err)
	}

	if dbUser.Name != "Manually Added User" || dbUser.RefreshToken != "mock-refresh" || dbUser.IsPremium != 1 {
		t.Fatalf("unexpected persisted user: %+v", dbUser)
	}
}

func TestGetUserProfile(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()
	user := model.PixezPixivUser{
		PixivUserID:  "12345",
		Name:         "Pixiv User",
		Account:      "pixiv_user",
		AccessToken:  "mock-access",
		RefreshToken: "mock-refresh",
	}

	client := NewClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "app-api.pixiv.net" && req.URL.Path == "/v1/user/detail" {
			if req.Header.Get("Authorization") != "Bearer mock-access" {
				t.Fatalf("unexpected auth: %s", req.Header.Get("Authorization"))
			}
			return jsonResponse(http.StatusOK, `{
				"user": {
					"id": 12345,
					"name": "Pixiv User",
					"account": "pixiv_user"
				},
				"profile": {
					"total_illusts": 42
				}
			}`), nil
		}
		t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		return nil, nil
	})})

	data, err := client.GetUserProfile(ctx, user, "12345")
	if err != nil {
		t.Fatalf("GetUserProfile() error = %v", err)
	}

	if !strings.Contains(string(data), `"total_illusts": 42`) {
		t.Fatalf("unexpected profile response data: %s", string(data))
	}
}

func TestAddPixivUserByCode(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()
	client := NewClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "oauth.secure.pixiv.net" && req.URL.Path == "/auth/token" {
			body, _ := io.ReadAll(req.Body)
			bodyStr := string(body)
			if !strings.Contains(bodyStr, "grant_type=authorization_code") {
				t.Fatalf("unexpected grant_type: %s", bodyStr)
			}
			if !strings.Contains(bodyStr, "code=input-code") {
				t.Fatalf("unexpected code: %s", bodyStr)
			}
			if !strings.Contains(bodyStr, "code_verifier=input-verifier") {
				t.Fatalf("unexpected verifier: %s", bodyStr)
			}
			return jsonResponse(http.StatusOK, `{
				"response": {
					"access_token": "mock-access",
					"refresh_token": "mock-refresh",
					"user": {
						"id": "12345",
						"name": "Manually Added User By Code",
						"account": "manual_account_code",
						"mail_address": "manual_code@example.com",
						"profile_image_urls": {
							"px_170x170": "https://example.com/avatar_manual_code.png"
						},
						"is_premium": true,
						"x_restrict": 1,
						"is_mail_authorized": true
					}
				}
			}`), nil
		}
		t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		return nil, nil
	})})

	user, err := client.AddPixivUserByCode(ctx, "input-code", "input-verifier")
	if err != nil {
		t.Fatalf("AddPixivUserByCode() error = %v", err)
	}

	if user.PixivUserID != "12345" || user.Name != "Manually Added User By Code" || user.AccessToken != "mock-access" || user.RefreshToken != "mock-refresh" {
		t.Fatalf("unexpected returned user: %+v", user)
	}

	// Verify persistence in DB
	var dbUser model.PixezPixivUser
	if err := db.DB(ctx).Where("pixiv_user_id = ?", "12345").First(&dbUser).Error; err != nil {
		t.Fatalf("failed to find user in DB: %v", err)
	}
	if dbUser.Name != "Manually Added User By Code" || dbUser.AccessToken != "mock-access" || dbUser.RefreshToken != "mock-refresh" {
		t.Fatalf("persisted credentials incorrect: %+v", dbUser)
	}
}


