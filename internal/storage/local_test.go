// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package storage

import (
	"bytes"
	"context"
	"io"
	"testing"
)

func TestLocalBackendRoundTrip(t *testing.T) {
	backend, err := newLocalBackend(LocalConfig{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("newLocalBackend() returned error: %v", err)
	}
	ctx := context.Background()
	const key = "uploads/2026/06/13/test.txt"
	const content = "wavelet storage"

	storedResult, err := backend.Put(ctx, key, bytes.NewBufferString(content), int64(len(content)), "text/plain")
	if err != nil {
		t.Fatalf("Put(%q) returned error: %v", key, err)
	}
	if storedResult.Key != key {
		t.Errorf("Put(%q) key = %q, want %q", key, storedResult.Key, key)
	}

	object, err := backend.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get(%q) returned error: %v", key, err)
	}
	got, err := io.ReadAll(object.Body)
	if err != nil {
		t.Fatalf("ReadAll(Get(%q)) returned error: %v", key, err)
	}
	if err := object.Body.Close(); err != nil {
		t.Fatalf("Close(Get(%q)) returned error: %v", key, err)
	}
	if string(got) != content {
		t.Errorf("Get(%q) content = %q, want %q", key, got, content)
	}

	if err := backend.Delete(ctx, key); err != nil {
		t.Fatalf("Delete(%q) returned error: %v", key, err)
	}
	if _, err := backend.Get(ctx, key); err == nil {
		t.Errorf("Get(%q) after Delete() returned nil error", key)
	}

	// Test key with leading slash
	slashKey := "/uploads/2026/06/13/slash_test.txt"
	putSlashRes, err := backend.Put(ctx, slashKey, bytes.NewBufferString(content), int64(len(content)), "text/plain")
	if err != nil {
		t.Fatalf("Put(%q) returned error: %v", slashKey, err)
	}
	if putSlashRes.Key != slashKey {
		t.Errorf("Put(%q) key = %q, want %q", slashKey, putSlashRes.Key, slashKey)
	}
	objSlash, err := backend.Get(ctx, slashKey)
	if err != nil {
		t.Fatalf("Get(%q) returned error: %v", slashKey, err)
	}
	_ = objSlash.Body.Close()
}

