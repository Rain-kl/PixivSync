// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"testing"
	"time"

	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/repository"
	"github.com/Rain-kl/Wavelet/internal/testhelper"
)

func TestListVisibleSystemConfigsUsesStoreCache(t *testing.T) {
	dbConn, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	repository.ResetSystemConfigRAMCacheForTest()
	if err := repository.InvalidateVisibleSystemConfigsCache(ctx); err != nil {
		t.Fatalf("InvalidateVisibleSystemConfigsCache() error = %v", err)
	}

	// Warm cache
	if _, err := repository.ListVisibleSystemConfigs(ctx); err != nil {
		t.Fatalf("ListVisibleSystemConfigs() warm cache error = %v", err)
	}

	// Directly insert a new config in DB (bypassing caching layer)
	if err := dbConn.Create(&model.SystemConfig{
		Key:         "cache_probe_public_key",
		Value:       "cache_probe_public_value",
		Type:        "system",
		Visibility:  model.ConfigVisibilityVisible,
		Description: "cache probe",
	}).Error; err != nil {
		t.Fatalf("Create(cache_probe_public_key) error = %v", err)
	}

	// Cached call: shouldn't return the new key yet
	cached, err := repository.ListVisibleSystemConfigs(ctx)
	if err != nil {
		t.Fatalf("ListVisibleSystemConfigs() cached call error = %v", err)
	}
	for _, item := range cached {
		if item.Key == "cache_probe_public_key" {
			t.Fatal("cached visible config list should be stale before invalidation")
		}
	}

	// Invalidate: triggers reload
	if err := repository.InvalidateVisibleSystemConfigsCache(ctx); err != nil {
		t.Fatalf("InvalidateVisibleSystemConfigsCache() error = %v", err)
	}

	// Wait for Pub/Sub delivery in test environment
	time.Sleep(100 * time.Millisecond)

	// Refreshed call: should return the new key
	refreshed, err := repository.ListVisibleSystemConfigs(ctx)
	if err != nil {
		t.Fatalf("ListVisibleSystemConfigs() refreshed call error = %v", err)
	}

	var found bool
	for _, item := range refreshed {
		if item.Key == "cache_probe_public_key" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("refreshed visible config list should include newly created public config")
	}
}
