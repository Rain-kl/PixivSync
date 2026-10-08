// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package pixez

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/testhelper"
)

func setupArtistTest(t *testing.T) context.Context {
	t.Helper()
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	t.Cleanup(cleanup)
	t.Chdir(t.TempDir())
	ctx := context.Background()
	if err := db.DB(ctx).Create(&model.PixezPixivUser{PixivUserID: "100", AccessToken: "test-token"}).Error; err != nil {
		t.Fatal(err)
	}
	return ctx
}
func testNovel(id int64, title string) Novel {
	var novel Novel
	novel.ID = id
	novel.Title = title
	novel.User.ID = 7
	novel.User.Name = "作者"
	return novel
}
func seedArchivedNovel(t *testing.T, ctx context.Context, id int64, title string) {
	t.Helper()
	novel := testNovel(id, title)
	if err := db.DB(ctx).Create(&model.PixezMirrorNovel{NovelID: id, Status: model.PixezMirrorStatusSuccess, DetailJSON: mustJSON(NovelDetail{Novel: novel}), TextJSON: `{"text":"原始正文"}`, TotalCount: 1, SuccessCount: 1}).Error; err != nil {
		t.Fatal(err)
	}
}
func TestArtistBackfillPreservesSnapshotsAndSeparatesCategories(t *testing.T) {
	ctx := setupArtistTest(t)
	seedArchivedNovel(t, ctx, 99, "原始标题")
	var illustration Illust
	illustration.ID = 55
	illustration.User.ID = 7
	illustration.User.Name = "作者"
	illustration.Title = "插画"
	if err := db.DB(ctx).Create(&model.PixezMirrorIllust{IllustID: 55, DetailJSON: mustJSON(IllustDetail{Illust: illustration})}).Error; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := BackfillArtistMirrors(ctx); err != nil {
			t.Fatal(err)
		}
	}
	for _, target := range []string{"novel", "illust"} {
		rows, total, err := ListArtists(ctx, ArtistQuery{TargetType: target, Page: 1, PageSize: 24})
		if err != nil || total != 1 || len(rows) != 1 || rows[0].Subscription.Enabled {
			t.Fatalf("target=%s total=%d rows=%+v err=%v", target, total, rows, err)
		}
	}
	changed, err := SetArtistSubscription(ctx, 7, "novel", true)
	if err != nil || !changed {
		t.Fatal(changed, err)
	}
	subs, err := ArtistSubscriptions(ctx, 7)
	if err != nil || len(subs) != 1 || subs[0].TargetType != "novel" {
		t.Fatal(subs, err)
	}
	changed, err = SetArtistSubscription(ctx, 7, "novel", true)
	if err != nil || changed {
		t.Fatal("repeat enable was not idempotent", err)
	}
	row, err := GetMirrorNovel(ctx, 99)
	if err != nil || row.TextJSON != `{"text":"原始正文"}` || !strings.Contains(row.DetailJSON, "原始标题") {
		t.Fatal("archive overwritten", row, err)
	}
}

func TestArtistScanDuplicateExecutionAndCancellationDuringPage(t *testing.T) {
	ctx := setupArtistTest(t)
	if err := IndexNovel(ctx, testNovel(99, "目录"), true, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := SetArtistSubscription(ctx, 7, model.PixezMirrorTargetNovel, true); err != nil {
		t.Fatal(err)
	}
	run, _, err := CreateArtistRun(ctx, 7, model.PixezMirrorTargetNovel, true, "initial")
	if err != nil {
		t.Fatal(err)
	}
	requests, downloads := 0, 0
	client := NewClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		// A second delivery of the same task must not fetch or advance its cursor.
		if err := ScanArtistRun(ctx, nil, run.ID, nil); err != nil {
			t.Fatal(err)
		}
		if _, err := SetArtistSubscription(ctx, 7, model.PixezMirrorTargetNovel, false); err != nil {
			t.Fatal(err)
		}
		return jsonResponse(200, `{"novels":[{"id":100,"title":"新作品","user":{"id":7,"name":"作者"}}],"next_url":null}`), nil
	})})
	if err := ScanArtistRun(ctx, client, run.ID, func(context.Context, string, int64) (bool, error) { downloads++; return true, nil }); err != nil {
		t.Fatal(err)
	}
	done, err := GetArtistRun(ctx, run.ID)
	if err != nil || done.Status != "cancelled" || downloads != 0 || requests != 1 {
		t.Fatal(done, err, requests, downloads)
	}
	if err := FailArtistRun(ctx, run.ID, true); err != nil {
		t.Fatal(err)
	}
	done, err = GetArtistRun(ctx, run.ID)
	if err != nil || done.Status != "cancelled" {
		t.Fatal("failure cleanup replaced cancellation", done, err)
	}
}

func TestArtistScanRejectsMissingResponseWithoutMarkingWorksMissing(t *testing.T) {
	ctx := setupArtistTest(t)
	if err := IndexNovel(ctx, testNovel(99, "保留"), true, ""); err != nil {
		t.Fatal(err)
	}
	run, _, err := CreateArtistRun(ctx, 7, model.PixezMirrorTargetNovel, false, "refresh")
	if err != nil {
		t.Fatal(err)
	}
	client := NewClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"error":{"message":"unavailable"}}`), nil
	})})
	if err := ScanArtistRun(ctx, client, run.ID, nil); err == nil {
		t.Fatal("missing response was accepted as empty directory")
	}
	var work model.PixezArtistWork
	if err := db.DB(ctx).First(&work).Error; err != nil || work.NotReturned {
		t.Fatal(work, err)
	}
}
func TestArtistScanSkipsCompleteArchivesAndKeepsMissingWorks(t *testing.T) {
	ctx := setupArtistTest(t)
	seedArchivedNovel(t, ctx, 99, "原始标题")
	seedArchivedNovel(t, ctx, 101, "已隐藏作品")
	if err := BackfillArtistMirrors(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := SetArtistSubscription(ctx, 7, "novel", true); err != nil {
		t.Fatal(err)
	}
	run, created, err := CreateArtistRun(ctx, 7, "novel", true, "initial")
	if err != nil || !created {
		t.Fatal(run, err)
	}
	calls := 0
	client := NewClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.Query().Get("offset") == "30" {
			return jsonResponse(200, `{"novels":[],"next_url":null}`), nil
		}
		payload := struct {
			Novels  []Novel `json:"novels"`
			NextURL string  `json:"next_url"`
		}{[]Novel{testNovel(99, "覆盖标题"), testNovel(100, "新作品")}, "https://" + pixivAPIHost + "/v1/user/novels?user_id=7&offset=30"}
		raw, e := json.Marshal(payload)
		if e != nil {
			return nil, e
		}
		return jsonResponse(200, string(raw)), nil
	})})
	var enqueued []int64
	if err = ScanArtistRun(ctx, client, run.ID, func(_ context.Context, target string, id int64) (bool, error) {
		if target != "novel" {
			t.Fatal(target)
		}
		enqueued = append(enqueued, id)
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	if calls != 2 || len(enqueued) != 1 || enqueued[0] != 100 {
		t.Fatal(calls, enqueued)
	}
	done, err := GetArtistRun(ctx, run.ID)
	if err != nil || done.Status != "success" || done.DiscoveredCount != 2 || done.SkippedCount != 1 || done.QueuedCount != 1 {
		t.Fatal(done, err)
	}
	subs, err := ArtistSubscriptions(ctx, 7)
	if err != nil || subs[0].LastSuccessAt == nil || subs[0].NextDueAt == nil || time.Until(*subs[0].NextDueAt) < 23*time.Hour {
		t.Fatal(subs, err)
	}
	var hidden model.PixezArtistWork
	if err = db.DB(ctx).Where("target_id = ?", 101).First(&hidden).Error; err != nil || !hidden.NotReturned {
		t.Fatal(hidden, err)
	}
	row, err := GetMirrorNovel(ctx, 99)
	if err != nil || !strings.Contains(row.DetailJSON, "原始标题") {
		t.Fatal("old detail replaced", err)
	}
	if _, err = SetArtistSubscription(ctx, 7, "novel", false); err != nil {
		t.Fatal(err)
	}
	row, err = GetMirrorNovel(ctx, 101)
	if err != nil || !novelComplete(row) {
		t.Fatal("cancel removed hidden archive", err)
	}
}
func TestArtistCancellationAndUntrustedPaginationDoNotFetchOrErase(t *testing.T) {
	ctx := setupArtistTest(t)
	seedArchivedNovel(t, ctx, 99, "保留")
	if err := BackfillArtistMirrors(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := SetArtistSubscription(ctx, 7, "novel", true); err != nil {
		t.Fatal(err)
	}
	run, _, err := CreateArtistRun(ctx, 7, "novel", true, "initial")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = SetArtistSubscription(ctx, 7, "novel", false); err != nil {
		t.Fatal(err)
	}
	calls := 0
	client := NewClient(&http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		calls++
		return jsonResponse(200, `{"novels":[],"next_url":"https://evil.example/v1/user/novels?user_id=7"}`), nil
	})})
	if err = ScanArtistRun(ctx, client, run.ID, nil); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("cancelled subscription made network calls")
	}
	refresh, _, err := CreateArtistRun(ctx, 7, "novel", false, "refresh")
	if err != nil {
		t.Fatal(err)
	}
	if err = ScanArtistRun(ctx, client, refresh.ID, nil); err == nil {
		t.Fatal("untrusted pagination accepted")
	}
	if calls != 1 {
		t.Fatal("credentials sent to pagination host", calls)
	}
	var work model.PixezArtistWork
	if err = db.DB(ctx).Where("target_id = ?", 99).First(&work).Error; err != nil || work.NotReturned {
		t.Fatal("incomplete scan marked work absent", err)
	}
}
func TestPartialIllustRetryOnlyFillsMissingPage(t *testing.T) {
	ctx := setupArtistTest(t)
	seedMirrorTestUser(t, ctx)
	var detail IllustDetail
	detail.Illust.ID = 123
	detail.Illust.User.ID = 7
	detail.Illust.Title = "原始插画"
	detail.Illust.PageCount = 2
	if err := json.Unmarshal([]byte(`[{"image_urls":{"original":"https://i.pximg.net/123_p0.jpg"}},{"image_urls":{"original":"https://i.pximg.net/123_p1.jpg"}}]`), &detail.Illust.MetaPages); err != nil {
		t.Fatal(err)
	}
	detailCalls := 0
	page0Calls := 0
	page1Calls := 0
	client := NewClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == pixivAPIHost {
			detailCalls++
			if detailCalls > 1 {
				return nil, errors.New("original now hidden")
			}
			return jsonResponse(200, mustJSON(detail)), nil
		}
		if strings.Contains(req.URL.Path, "p0") {
			page0Calls++
		} else {
			page1Calls++
			if page1Calls == 1 {
				return nil, errors.New("temporary download failure")
			}
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"image/jpeg"}}, Body: io.NopCloser(strings.NewReader(req.URL.Path))}, nil
	})})
	if _, err := EnsureMirrorIllustQueued(ctx, 123, "first"); err != nil {
		t.Fatal(err)
	}
	if err := ProcessMirrorIllust(ctx, client, "first", 123); err == nil {
		t.Fatal("partial download was reported complete")
	}
	first, err := GetMirrorIllust(ctx, 123)
	if err != nil || first.SuccessCount != 1 {
		t.Fatal(first, err)
	}
	if _, err = EnsureMirrorIllustQueued(ctx, 123, "retry"); err != nil {
		t.Fatal(err)
	}
	if err = ProcessMirrorIllust(ctx, client, "retry", 123); err != nil {
		t.Fatal(err)
	}
	final, err := GetMirrorIllust(ctx, 123)
	if err != nil || !illustComplete(ctx, final) {
		t.Fatal(final, err)
	}
	if detailCalls != 1 || page0Calls != 1 || page1Calls != 2 {
		t.Fatal("retry fetched archived content", detailCalls, page0Calls, page1Calls)
	}
	if final.DetailJSON != first.DetailJSON {
		t.Fatal("snapshot overwritten")
	}
	if _, err = EnsureMirrorIllustQueued(ctx, 123, "again"); err != nil {
		t.Fatal(err)
	}
	if err = ProcessMirrorIllust(ctx, client, "again", 123); err != nil {
		t.Fatal(err)
	}
	if detailCalls != 1 || page0Calls != 1 || page1Calls != 2 {
		t.Fatal("complete mirror downloaded again")
	}
}
func TestArchivedNovelAndStaleWorkerCannotOverwriteSnapshots(t *testing.T) {
	ctx := setupArtistTest(t)
	seedArchivedNovel(t, ctx, 99, "保留")
	client := NewClient(&http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		t.Fatal("complete novel accessed upstream")
		return nil, nil
	})})
	if _, err := EnsureMirrorNovelQueued(ctx, 99, "new"); err != nil {
		t.Fatal(err)
	}
	if err := ProcessMirrorNovel(ctx, client, "new", 99); err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureMirrorNovelQueued(ctx, 100, "worker"); err != nil {
		t.Fatal(err)
	}
	err := runProtectedMirror(ctx, "novel", 100, "worker", func(leaseCtx context.Context) error {
		if e := db.DB(leaseCtx).Model(&model.PixezMirrorNovel{}).Where("novel_id = ?", 100).Updates(map[string]any{"execution_token": "another-owner", "detail_json": "protected"}).Error; e != nil {
			return e
		}
		return updateMirrorNovel(leaseCtx, 100, map[string]any{"detail_json": "overwritten"})
	})
	if err == nil {
		t.Fatal("stale worker wrote archive")
	}
	row, e := GetMirrorNovel(ctx, 100)
	if e != nil || row.DetailJSON != "protected" || row.ExecutionToken != "another-owner" {
		t.Fatal(row, e)
	}
}
