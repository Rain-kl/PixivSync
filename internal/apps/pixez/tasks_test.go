// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package pixez

import (
	"encoding/json"
	"testing"

	pixezsvc "github.com/Rain-kl/Wavelet/internal/service/pixez"
)

func TestMirrorTaskPayloadValidation(t *testing.T) {
	handler := &MirrorTaskHandler{}
	if _, err := handler.ValidatePayload([]byte(`{"target_type":0,"target_id":0}`)); err == nil {
		t.Fatal("ValidatePayload() error = nil, want target_id validation error")
	}
	if _, err := handler.ValidatePayload([]byte(`{"target_type":3,"target_id":123}`)); err == nil {
		t.Fatal("ValidatePayload() error = nil, want target_type validation error")
	}
	payload, err := handler.ValidatePayload([]byte(`{"target_type":0,"target_id":123}`))
	if err != nil {
		t.Fatalf("ValidatePayload() error = %v", err)
	}
	var req mirrorPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("decode normalized payload failed: %v", err)
	}
	if req.TargetType != TargetTypeIllust || req.TargetID != 123 {
		t.Fatalf("req = %+v, want illust target_id 123", req)
	}
}

func TestExportBookmarksTaskPayloadValidation(t *testing.T) {
	handler := &ExportBookmarksTaskHandler{}
	payload, err := handler.ValidatePayload(nil)
	if err != nil {
		t.Fatalf("ValidatePayload(nil) error = %v", err)
	}
	var req exportBookmarksPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("decode normalized payload failed: %v", err)
	}
	if req.TargetType != nil {
		t.Fatalf("expected TargetType to be nil, got: %v", req.TargetType)
	}

	if _, err := handler.ValidatePayload([]byte(`{"target_type":3}`)); err == nil {
		t.Fatal("ValidatePayload() error = nil, want target_type validation error")
	}
	payload, err = handler.ValidatePayload([]byte(`{"target_type":1,"pixiv_user_id":" 100 "}`))
	if err != nil {
		t.Fatalf("ValidatePayload() error = %v", err)
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("decode normalized payload failed: %v", err)
	}
	if req.TargetType == nil || *req.TargetType != TargetTypeNovel || req.PixivUserID != "100" {
		t.Fatalf("req = %+v, want novel user_id 100", req)
	}
}

func TestAutoMirrorTaskPayloadValidation(t *testing.T) {
	handler := &AutoEnqueueBookmarkMirrorsTaskHandler{}
	payload, err := handler.ValidatePayload(nil)
	if err != nil {
		t.Fatalf("ValidatePayload(nil) error = %v", err)
	}
	var req autoMirrorPayload
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("decode default payload failed: %v", err)
	}
	if req.Limit != 50 {
		t.Fatalf("default limit = %d, want 50", req.Limit)
	}

	payload, err = handler.ValidatePayload([]byte(`{"target_type":0,"limit":999}`))
	if err != nil {
		t.Fatalf("ValidatePayload() error = %v", err)
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("decode capped payload failed: %v", err)
	}
	if req.TargetType == nil || *req.TargetType != TargetTypeIllust || req.Limit != 500 {
		t.Fatalf("normalized payload = %+v, want illust limit 500", req)
	}

	if _, err := handler.ValidatePayload([]byte(`{"target_type":3}`)); err == nil {
		t.Fatal("ValidatePayload() error = nil, want target_type validation error")
	}
}

func TestImportLegacyTaskPayloadDefaults(t *testing.T) {
	handler := &ImportLegacyServerTaskHandler{}

	// Test boolean true
	payload, err := handler.ValidatePayload([]byte(`{"dry_run":true}`))
	if err != nil {
		t.Fatalf("ValidatePayload(dry_run:true) error = %v", err)
	}
	var req pixezsvc.ImportLegacyRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("decode import payload failed: %v", err)
	}
	if req.SQLitePath != "server/pixez-sync.db" || req.MirrorDir != "server/data/mirror" || !req.DryRun {
		t.Fatalf("unexpected import defaults with true: %+v", req)
	}

	// Test boolean false
	payload, err = handler.ValidatePayload([]byte(`{"dry_run":false}`))
	if err != nil {
		t.Fatalf("ValidatePayload(dry_run:false) error = %v", err)
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("decode import payload failed: %v", err)
	}
	if req.DryRun {
		t.Fatalf("expected DryRun to be false when parsed from boolean false")
	}
}
