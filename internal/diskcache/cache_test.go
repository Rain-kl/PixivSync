// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package diskcache

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/testhelper"
)

func TestDiskCacheBasic(t *testing.T) {
	testDir := "uploads/test_diskcache_basic"
	defer func() { _ = os.RemoveAll(testDir) }()
	_ = os.RemoveAll(testDir)

	c := New(testDir)
	defer func() { _ = c.Clear() }()

	key := "key1"
	val := []byte("value1")

	// Get non-existent
	_, err := c.Get(key)
	if err != ErrCacheMiss {
		t.Fatalf("expected ErrCacheMiss, got %v", err)
	}

	// Set & Get
	err = c.Set(key, val, 10*time.Second)
	if err != nil {
		t.Fatalf("failed to set cache: %v", err)
	}

	got, err := c.Get(key)
	if err != nil {
		t.Fatalf("failed to get cache: %v", err)
	}

	if !bytes.Equal(got, val) {
		t.Errorf("expected %s, got %s", val, got)
	}

	// Delete
	err = c.Delete(key)
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	_, err = c.Get(key)
	if err != ErrCacheMiss {
		t.Errorf("expected ErrCacheMiss after delete, got %v", err)
	}
}

func TestDiskCacheTTL(t *testing.T) {
	testDir := "uploads/test_diskcache_ttl"
	defer func() { _ = os.RemoveAll(testDir) }()
	_ = os.RemoveAll(testDir)

	c := New(testDir)
	defer func() { _ = c.Clear() }()

	key := "ttlkey"
	val := []byte("ttlval")

	// Set with 200ms TTL
	err := c.Set(key, val, 200*time.Millisecond)
	if err != nil {
		t.Fatalf("failed to set: %v", err)
	}

	// Immediate Get should succeed
	got, err := c.Get(key)
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if !bytes.Equal(got, val) {
		t.Errorf("expected %s, got %s", val, got)
	}

	// Sleep 250ms to expire
	time.Sleep(250 * time.Millisecond)

	// Get should fail with cache miss
	_, err = c.Get(key)
	if err != ErrCacheMiss {
		t.Errorf("expected ErrCacheMiss after TTL expiration, got %v", err)
	}
}

func TestDiskCacheLRUEviction(t *testing.T) {
	testDir := "uploads/test_diskcache_lru"
	defer func() { _ = os.RemoveAll(testDir) }()
	_ = os.RemoveAll(testDir)

	c := New(testDir)
	defer func() { _ = c.Clear() }()

	// Force a very small max size of 20 bytes for testing (8 bytes header + payload)
	// So 2 items of 2 bytes payload = 2 * (8 + 2) = 20 bytes max.
	c.maxSize = 20
	c.lruEnabled = true

	// Write item 1: 8 + 2 = 10 bytes
	err := c.Set("k1", []byte("v1"), 0)
	if err != nil {
		t.Fatalf("failed to set k1: %v", err)
	}

	// Write item 2: 8 + 2 = 10 bytes
	err = c.Set("k2", []byte("v2"), 0)
	if err != nil {
		t.Fatalf("failed to set k2: %v", err)
	}

	// Both should exist
	if _, err := c.Get("k1"); err != nil {
		t.Errorf("k1 should exist: %v", err)
	}
	if _, err := c.Get("k2"); err != nil {
		t.Errorf("k2 should exist: %v", err)
	}

	// Write item 3: 8 + 2 = 10 bytes -> total size would be 30, exceeding 20.
	// This should evict the oldest item. Since k1 was accessed, but then k2 was accessed,
	// wait, let's access k1 again to make it the most recently used, so k2 becomes oldest!
	_, _ = c.Get("k1") // k1 is now MRU, k2 is LRU

	err = c.Set("k3", []byte("v3"), 0)
	if err != nil {
		t.Fatalf("failed to set k3: %v", err)
	}

	// k2 should be evicted, k1 and k3 should exist
	_, err = c.Get("k2")
	if err != ErrCacheMiss {
		t.Errorf("expected k2 to be evicted, got error %v", err)
	}

	if _, err := c.Get("k1"); err != nil {
		t.Errorf("k1 should still exist: %v", err)
	}

	if _, err := c.Get("k3"); err != nil {
		t.Errorf("k3 should exist: %v", err)
	}
}

func TestDiskCacheReloadConfig(t *testing.T) {
	dbConn, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	testDir := "uploads/test_diskcache_reload"
	defer func() { _ = os.RemoveAll(testDir) }()
	_ = os.RemoveAll(testDir)

	c := New(testDir)
	defer func() { _ = c.Clear() }()

	// Update DB config values
	dbConn.Model(&model.SystemConfig{}).Where("key = ?", model.ConfigKeyDiskCacheMaxSizeMB).Update("value", "250")
	dbConn.Model(&model.SystemConfig{}).Where("key = ?", model.ConfigKeyDiskCacheTTLMinutes).Update("value", "120")
	dbConn.Model(&model.SystemConfig{}).Where("key = ?", model.ConfigKeyDiskCacheLRUEnabled).Update("value", "false")

	// Invalidate Redis config cache to force DB reload
	if db.Redis != nil {
		db.Redis.Del(context.Background(), db.PrefixedKey(model.SystemConfigRedisHashKey))
	}

	// Reload config
	c.ReloadConfig(context.Background())

	status := c.Status()
	if status.MaxSizeMB != 250 {
		t.Errorf("expected MaxSizeMB to be 250, got %d", status.MaxSizeMB)
	}
	if status.TTLMinutes != 120 {
		t.Errorf("expected TTLMinutes to be 120, got %d", status.TTLMinutes)
	}
	if status.LRUEnabled != false {
		t.Errorf("expected LRUEnabled to be false, got %t", status.LRUEnabled)
	}
}
