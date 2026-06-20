// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package analytics

import "time"

// AccessLogFilter scopes ClickHouse user access log queries.
type AccessLogFilter struct {
	// UserIDs filters by user IDs. nil means no user filter; an empty slice means no matches.
	UserIDs []uint64
	Path    string
	// StartTime filters created_at >= StartTime when non-nil.
	StartTime *time.Time
	// EndTime filters created_at <= EndTime when non-nil.
	EndTime *time.Time
}