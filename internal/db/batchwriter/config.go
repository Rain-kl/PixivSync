// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package batchwriter

import (
	"fmt"
	"time"
)

const (
	defaultQueueSize    = 10_000
	defaultMaxBatchSize = 1_000
	defaultFlushEvery   = time.Second
)

// Config controls queue capacity and flush thresholds for a Writer instance.
type Config struct {
	// Name identifies the writer in logs and diagnostics. Optional.
	Name string

	// QueueSize is the buffered channel capacity.
	QueueSize int

	// MaxBatchSize triggers a flush when the in-memory batch reaches this count.
	MaxBatchSize int

	// FlushInterval triggers a time-based flush even when the batch is smaller.
	FlushInterval time.Duration
}

// DefaultConfig returns production-friendly defaults aligned with audit log batching.
func DefaultConfig() Config {
	return Config{
		QueueSize:     defaultQueueSize,
		MaxBatchSize:  defaultMaxBatchSize,
		FlushInterval: defaultFlushEvery,
	}
}

func (c Config) validate() error {
	if c.QueueSize <= 0 {
		return fmt.Errorf("batchwriter: queue size must be positive")
	}
	if c.MaxBatchSize <= 0 {
		return fmt.Errorf("batchwriter: max batch size must be positive")
	}
	if c.FlushInterval <= 0 {
		return fmt.Errorf("batchwriter: flush interval must be positive")
	}
	return nil
}
