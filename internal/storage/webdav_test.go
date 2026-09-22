// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package storage

import (
	"bytes"
	"context"
	"io"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/webdav"
)

func TestWebDAVTargetPath(t *testing.T) {
	tests := []struct {
		name     string
		basePath string
		key      string
		expected string
	}{
		{
			name:     "no base path, relative key",
			basePath: "",
			key:      "uploads/2026/09/17/1.jpg",
			expected: "/uploads/2026/09/17/1.jpg",
		},
		{
			name:     "no base path, leading slash key",
			basePath: "",
			key:      "/uploads/2026/09/17/1.jpg",
			expected: "/uploads/2026/09/17/1.jpg",
		},
		{
			name:     "with base path, relative key (uploading)",
			basePath: "/DockerData/pixiv_sync_data_webdav",
			key:      "uploads/2026/09/17/105102954490499072.jpg",
			expected: "/DockerData/pixiv_sync_data_webdav/uploads/2026/09/17/105102954490499072.jpg",
		},
		{
			name:     "with base path, key with leading slash (uploading)",
			basePath: "/DockerData/pixiv_sync_data_webdav",
			key:      "/uploads/2026/09/17/105102954490499072.jpg",
			expected: "/DockerData/pixiv_sync_data_webdav/uploads/2026/09/17/105102954490499072.jpg",
		},
		{
			name:     "with base path, key already has base path (reading from legacy DB)",
			basePath: "/DockerData/pixiv_sync_data_webdav",
			key:      "/DockerData/pixiv_sync_data_webdav/uploads/2026/09/17/105102954490499072.jpg",
			expected: "/DockerData/pixiv_sync_data_webdav/uploads/2026/09/17/105102954490499072.jpg",
		},
		{
			name:     "with base path without leading slash, key already has base path",
			basePath: "DockerData/pixiv_sync_data_webdav",
			key:      "/DockerData/pixiv_sync_data_webdav/uploads/2026/09/17/105102954490499072.jpg",
			expected: "/DockerData/pixiv_sync_data_webdav/uploads/2026/09/17/105102954490499072.jpg",
		},
		{
			name:     "with base path with trailing slash, key already has base path",
			basePath: "/DockerData/pixiv_sync_data_webdav/",
			key:      "/DockerData/pixiv_sync_data_webdav/uploads/2026/09/17/105102954490499072.jpg",
			expected: "/DockerData/pixiv_sync_data_webdav/uploads/2026/09/17/105102954490499072.jpg",
		},
		{
			name:     "with base path, key already has double base path from previous bug",
			basePath: "/DockerData/pixiv_sync_data_webdav",
			key:      "/DockerData/pixiv_sync_data_webdav/DockerData/pixiv_sync_data_webdav/uploads/2026/09/17/105102954490499072.jpg",
			expected: "/DockerData/pixiv_sync_data_webdav/uploads/2026/09/17/105102954490499072.jpg",
		},
		{
			name:     "key with similar prefix name that is not a directory match",
			basePath: "/data",
			key:      "/data_backup/uploads/1.jpg",
			expected: "/data/data_backup/uploads/1.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, err := newWebDAVBackend(WebDAVConfig{
				Endpoint: "http://127.0.0.1:5005",
				BasePath: tt.basePath,
			})
			if err != nil {
				t.Fatalf("newWebDAVBackend failed: %v", err)
			}
			actual := backend.targetPath(tt.key)
			if actual != tt.expected {
				t.Errorf("targetPath(%q) = %q, want %q", tt.key, actual, tt.expected)
			}
		})
	}
}

func TestWebDAVRelKey(t *testing.T) {
	tests := []struct {
		name     string
		basePath string
		key      string
		expected string
	}{
		{
			name:     "relative key remains relative",
			basePath: "/DockerData/pixiv_sync_data_webdav",
			key:      "uploads/2026/09/17/105102954490499072.jpg",
			expected: "uploads/2026/09/17/105102954490499072.jpg",
		},
		{
			name:     "leading slash key is stripped to relative",
			basePath: "/DockerData/pixiv_sync_data_webdav",
			key:      "/uploads/2026/09/17/105102954490499072.jpg",
			expected: "uploads/2026/09/17/105102954490499072.jpg",
		},
		{
			name:     "legacy key with basePath is stripped to relative",
			basePath: "/DockerData/pixiv_sync_data_webdav",
			key:      "/DockerData/pixiv_sync_data_webdav/uploads/2026/09/17/105102954490499072.jpg",
			expected: "uploads/2026/09/17/105102954490499072.jpg",
		},
		{
			name:     "corrupted key with duplicate basePath is stripped to relative",
			basePath: "/DockerData/pixiv_sync_data_webdav",
			key:      "/DockerData/pixiv_sync_data_webdav/DockerData/pixiv_sync_data_webdav/uploads/2026/09/17/105102954490499072.jpg",
			expected: "uploads/2026/09/17/105102954490499072.jpg",
		},
		{
			name:     "empty basePath preserves relative key",
			basePath: "",
			key:      "/uploads/2026/09/17/105102954490499072.jpg",
			expected: "uploads/2026/09/17/105102954490499072.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, err := newWebDAVBackend(WebDAVConfig{
				Endpoint: "http://127.0.0.1:5005",
				BasePath: tt.basePath,
			})
			if err != nil {
				t.Fatalf("newWebDAVBackend failed: %v", err)
			}
			actual := backend.relKey(tt.key)
			if actual != tt.expected {
				t.Errorf("relKey(%q) = %q, want %q", tt.key, actual, tt.expected)
			}
		})
	}
}

func TestWebDAVBackendRoundTrip(t *testing.T) {
	wdHandler := &webdav.Handler{
		FileSystem: webdav.NewMemFS(),
		LockSystem: webdav.NewMemLS(),
	}
	server := httptest.NewServer(wdHandler)
	defer server.Close()

	ctx := context.Background()
	basePath := "/DockerData/pixiv_sync_data_webdav"
	backend, err := newWebDAVBackend(WebDAVConfig{
		Endpoint: server.URL,
		BasePath: basePath,
	})
	if err != nil {
		t.Fatalf("newWebDAVBackend failed: %v", err)
	}

	// 1. Test connection
	if err := backend.Test(ctx); err != nil {
		t.Fatalf("backend.Test failed: %v", err)
	}

	// 2. Put object using relative key (typical upload flow)
	origContent := []byte("test image content 12345")
	objectKey := "uploads/2026/09/17/105102954490499072.jpg"
	putRes, err := backend.Put(ctx, objectKey, bytes.NewReader(origContent), int64(len(origContent)), "image/jpeg")
	if err != nil {
		t.Fatalf("backend.Put failed: %v", err)
	}

	// PutResult.Key MUST be the pure logical relative key, decoupled from basePath
	expectedLogicalKey := "uploads/2026/09/17/105102954490499072.jpg"
	if putRes.Key != expectedLogicalKey {
		t.Errorf("putRes.Key = %q, want %q", putRes.Key, expectedLogicalKey)
	}

	// 3. Get object using logical key (new standard upload flow)
	obj, err := backend.Get(ctx, putRes.Key)
	if err != nil {
		t.Fatalf("backend.Get with putRes.Key failed: %v", err)
	}
	defer obj.Body.Close()

	bodyBytes, err := io.ReadAll(obj.Body)
	if err != nil {
		t.Fatalf("read body failed: %v", err)
	}
	if !bytes.Equal(bodyBytes, origContent) {
		t.Errorf("read content = %q, want %q", string(bodyBytes), string(origContent))
	}

	// 4. Get object using legacy key containing basePath (existing DB records from before fix)
	legacyKey := "/DockerData/pixiv_sync_data_webdav/uploads/2026/09/17/105102954490499072.jpg"
	obj2, err := backend.Get(ctx, legacyKey)
	if err != nil {
		t.Fatalf("backend.Get with legacyKey failed: %v", err)
	}
	defer obj2.Body.Close()
	bodyBytes2, err := io.ReadAll(obj2.Body)
	if err != nil {
		t.Fatalf("read body failed: %v", err)
	}
	if !bytes.Equal(bodyBytes2, origContent) {
		t.Errorf("read content = %q, want %q", string(bodyBytes2), string(origContent))
	}

	// 5. Get object using accidental double basePath (defensive recovery for corrupted DB records)
	doubledKey := "/DockerData/pixiv_sync_data_webdav/DockerData/pixiv_sync_data_webdav/uploads/2026/09/17/105102954490499072.jpg"
	obj3, err := backend.Get(ctx, doubledKey)
	if err != nil {
		t.Fatalf("backend.Get with doubledKey failed: %v", err)
	}
	defer obj3.Body.Close()
	bodyBytes3, err := io.ReadAll(obj3.Body)
	if err != nil {
		t.Fatalf("read body failed: %v", err)
	}
	if !bytes.Equal(bodyBytes3, origContent) {
		t.Errorf("read content = %q, want %q", string(bodyBytes3), string(origContent))
	}

	// 6. Delete object using logical key
	if err := backend.Delete(ctx, putRes.Key); err != nil {
		t.Fatalf("backend.Delete failed: %v", err)
	}

	// 7. Verify object is deleted
	_, err = backend.Get(ctx, putRes.Key)
	if err == nil {
		t.Fatalf("expected error after delete, got nil")
	}

	// 8. Put again, and delete using legacy key format
	_, err = backend.Put(ctx, objectKey, bytes.NewReader(origContent), int64(len(origContent)), "image/jpeg")
	if err != nil {
		t.Fatalf("second backend.Put failed: %v", err)
	}
	if err := backend.Delete(ctx, legacyKey); err != nil {
		t.Fatalf("backend.Delete with legacyKey failed: %v", err)
	}
	_, err = backend.Get(ctx, objectKey)
	if err == nil {
		t.Fatalf("expected error after delete with legacyKey, got nil")
	}
}


