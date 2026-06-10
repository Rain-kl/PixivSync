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
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/testhelper"
)

func TestExportIllustBookmarksMarksPlaceholdersAndMissingRowsRemoved(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()
	user := model.PixezPixivUser{
		PixivUserID:  "100",
		Name:         "Pixiv User",
		Account:      "pixiv_user",
		AccessToken:  "pixiv-token",
		RefreshToken: "pixiv-refresh",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := db.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("seed pixiv user failed: %v", err)
	}

	active := newTestIllust(9, "still active")
	missing := newTestIllust(11, "missing")
	for _, row := range []model.PixezBookmarkIllust{
		{
			PixivUserID:     user.PixivUserID,
			Restrict:        restrictPublic,
			IllustID:        active.ID,
			IllustJSON:      mustJSON(active),
			LastExportRunID: "previous",
			LastSeenAt:      time.Now().Add(-time.Hour),
			Removed:         false,
			CreatedAt:       time.Now().Add(-time.Hour),
			UpdatedAt:       time.Now().Add(-time.Hour),
		},
		{
			PixivUserID:     user.PixivUserID,
			Restrict:        restrictPublic,
			IllustID:        missing.ID,
			IllustJSON:      mustJSON(missing),
			LastExportRunID: "previous",
			LastSeenAt:      time.Now().Add(-time.Hour),
			Removed:         false,
			CreatedAt:       time.Now().Add(-time.Hour),
			UpdatedAt:       time.Now().Add(-time.Hour),
		},
	} {
		if err := db.DB(ctx).Create(&row).Error; err != nil {
			t.Fatalf("seed bookmark row failed: %v", err)
		}
	}

	placeholder := newTestIllust(10, "hidden")
	placeholder.ImageUrls.SquareMedium = "https://i.pximg.net/c/360x360/limit_unknown_360.png"

	client := NewClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != pixivAPIHost {
			t.Fatalf("unexpected host: %s", req.URL.Host)
		}
		switch req.URL.Query().Get("restrict") {
		case restrictPublic:
			return jsonResponse(http.StatusOK, `{"illusts":[`+mustJSON(active)+`,`+mustJSON(placeholder)+`],"next_url":""}`), nil
		case restrictPrivate:
			return jsonResponse(http.StatusOK, `{"illusts":[],"next_url":""}`), nil
		default:
			t.Fatalf("unexpected restrict: %s", req.URL.RawQuery)
			return nil, nil
		}
	})})

	summary, err := ExportIllustBookmarks(ctx, client, user.PixivUserID)
	if err != nil {
		t.Fatalf("ExportIllustBookmarks() error = %v", err)
	}
	if summary.RunCount != 2 || summary.TotalCount != 2 || summary.RemovedCount != 2 || summary.NewCount != 0 || summary.UpdatedCount != 0 {
		t.Fatalf("unexpected export summary: %+v", summary)
	}

	var rows []model.PixezBookmarkIllust
	if err := db.DB(ctx).Where("pixiv_user_id = ?", user.PixivUserID).Order("illust_id asc").Find(&rows).Error; err != nil {
		t.Fatalf("load bookmark rows failed: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("row count = %d, want 3", len(rows))
	}
	statusByID := make(map[int64]bool, len(rows))
	for _, row := range rows {
		statusByID[row.IllustID] = row.Removed
	}
	if statusByID[9] {
		t.Fatalf("active bookmark was marked removed: %+v", rows)
	}
	if !statusByID[10] || !statusByID[11] {
		t.Fatalf("placeholder and missing rows should be removed: %+v", rows)
	}

	var runs []model.PixezBookmarkExportRun
	if err := db.DB(ctx).Order("restrict asc").Find(&runs).Error; err != nil {
		t.Fatalf("load export runs failed: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("run count = %d, want 2", len(runs))
	}
	for _, run := range runs {
		if run.Status != model.PixezBookmarkExportStatusSuccess || run.FinishedAt == nil {
			t.Fatalf("run not completed successfully: %+v", run)
		}
		if strings.TrimSpace(run.ID) == "" {
			t.Fatalf("run ID is empty: %+v", run)
		}
	}
}

func newTestIllust(id int64, title string) Illust {
	illust := Illust{ID: id, Title: title, Type: "illust", Visible: true, PageCount: 1}
	illust.User.ID = 200
	illust.User.Name = "artist"
	illust.ImageUrls.SquareMedium = "https://i.pximg.net/c/360x360/img-master/img/2026/01/01/00/00/00/test.jpg"
	illust.MetaSinglePage.OriginalImageURL = "https://i.pximg.net/img-original/img/2026/01/01/00/00/00/test.jpg"
	return illust
}
