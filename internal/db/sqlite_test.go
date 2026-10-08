// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package db

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestConfigureSQLitePreservesDataAndSerializesConcurrentTransactions(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "test.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := database.Exec("CREATE TABLE counter (id INTEGER PRIMARY KEY, value INTEGER NOT NULL)").Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Exec("INSERT INTO counter VALUES (1, 10)").Error; err != nil {
		t.Fatal(err)
	}
	if err := configureSQLite(database); err != nil {
		t.Fatal(err)
	}
	var journalMode string
	if err := database.Raw("PRAGMA journal_mode").Scan(&journalMode).Error; err != nil || journalMode != "wal" {
		t.Fatalf("journal_mode=%q, err=%v", journalMode, err)
	}
	if sqlDB.Stats().MaxOpenConnections != 1 {
		t.Fatal("SQLite connection pool is not limited to one connection")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	const writers, increments = 20, 10
	errors := make(chan error, writers)
	var workers sync.WaitGroup
	for range writers {
		workers.Go(func() {
			for range increments {
				if err := database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
					var value int
					if err := tx.Raw("SELECT value FROM counter WHERE id = 1").Scan(&value).Error; err != nil {
						return err
					}
					return tx.Exec("UPDATE counter SET value = ? WHERE id = 1", value+1).Error
				}); err != nil {
					errors <- err
					return
				}
			}
		})
	}
	workers.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
	var value int
	if err := database.WithContext(ctx).Raw("SELECT value FROM counter WHERE id = 1").Scan(&value).Error; err != nil {
		t.Fatal(err)
	}
	if value != 10+writers*increments {
		t.Fatalf("existing data or concurrent increments lost: value=%d", value)
	}
}
