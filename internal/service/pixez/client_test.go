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
