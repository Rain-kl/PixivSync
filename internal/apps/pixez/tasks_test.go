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
	"encoding/json"
	"testing"

	pixezsvc "github.com/Rain-kl/Wavelet/internal/service/pixez"
)

func TestMirrorTaskPayloadValidation(t *testing.T) {
	illustHandler := &MirrorIllustTaskHandler{}
	if _, err := illustHandler.ValidatePayload([]byte(`{"illust_id":0}`)); err == nil {
		t.Fatal("ValidatePayload() error = nil, want illust_id validation error")
	}
	payload, err := illustHandler.ValidatePayload([]byte(`{"illust_id":123}`))
	if err != nil {
		t.Fatalf("ValidatePayload() error = %v", err)
	}
	var illustReq mirrorIllustPayload
	if err := json.Unmarshal(payload, &illustReq); err != nil {
		t.Fatalf("decode normalized payload failed: %v", err)
	}
	if illustReq.IllustID != 123 {
		t.Fatalf("illust_id = %d, want 123", illustReq.IllustID)
	}

	novelHandler := &MirrorNovelTaskHandler{}
	if _, err := novelHandler.ValidatePayload([]byte(`{"novel_id":-1}`)); err == nil {
		t.Fatal("ValidatePayload() error = nil, want novel_id validation error")
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

	payload, err = handler.ValidatePayload([]byte(`{"target_type":"illust","limit":999}`))
	if err != nil {
		t.Fatalf("ValidatePayload() error = %v", err)
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("decode capped payload failed: %v", err)
	}
	if req.TargetType != "illust" || req.Limit != 500 {
		t.Fatalf("normalized payload = %+v, want illust limit 500", req)
	}

	if _, err := handler.ValidatePayload([]byte(`{"target_type":"bad"}`)); err == nil {
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

	// Test string "true"
	payload, err = handler.ValidatePayload([]byte(`{"dry_run":"true"}`))
	if err != nil {
		t.Fatalf("ValidatePayload(dry_run:\"true\") error = %v", err)
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("decode import payload failed: %v", err)
	}
	if !req.DryRun {
		t.Fatalf("expected DryRun to be true when parsed from string \"true\"")
	}

	// Test string "false"
	payload, err = handler.ValidatePayload([]byte(`{"dry_run":"false"}`))
	if err != nil {
		t.Fatalf("ValidatePayload(dry_run:\"false\") error = %v", err)
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("decode import payload failed: %v", err)
	}
	if req.DryRun {
		t.Fatalf("expected DryRun to be false when parsed from string \"false\"")
	}
}
