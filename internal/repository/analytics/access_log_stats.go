// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package analytics

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	analyticsmodel "github.com/Rain-kl/Wavelet/internal/model/analytics"
)

const hoursInDay = 24

// DailyTrend is a single day's access count.
type DailyTrend struct {
	Date  string
	Count uint64
}

// BrowserShare is a browser group's share of access logs.
type BrowserShare struct {
	Browser string
	Count   uint64
}

// TopUser is an active user ranked by access count.
type TopUser struct {
	UserID uint64
	Count  uint64
}

// GetDailyTrend returns per-day access counts for the last days days (inclusive of today).
func GetDailyTrend(ctx context.Context, days int) ([]DailyTrend, error) {
	if days < 1 {
		days = 7
	}

	ch := db.ChDB(ctx)
	if ch == nil {
		return nil, fmt.Errorf("clickhouse gorm connection is not initialized")
	}

	startTime := time.Now().AddDate(0, 0, -(days - 1)).Truncate(hoursInDay * time.Hour)
	tableName := analyticsmodel.UserAccessLog{}.TableName()

	query := fmt.Sprintf(`
		SELECT toDate(created_at) AS date, count() AS count
		FROM %s
		WHERE created_at >= ?
		GROUP BY date
		ORDER BY date ASC
	`, tableName)

	type trendRow struct {
		Date  time.Time
		Count uint64
	}

	var rows []trendRow
	if err := ch.Raw(query, startTime).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("get daily trend: %w", err)
	}

	trendMap := make(map[string]uint64, days)
	for i := 0; i < days; i++ {
		dateStr := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		trendMap[dateStr] = 0
	}
	for _, row := range rows {
		dateStr := row.Date.Format("2006-01-02")
		trendMap[dateStr] = row.Count
	}

	result := make([]DailyTrend, 0, days)
	for i := days - 1; i >= 0; i-- {
		dateStr := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		result = append(result, DailyTrend{
			Date:  dateStr,
			Count: trendMap[dateStr],
		})
	}
	return result, nil
}

// GetBrowserDistribution returns browser-grouped access counts since startTime.
func GetBrowserDistribution(ctx context.Context, startTime time.Time) ([]BrowserShare, error) {
	ch := db.ChDB(ctx)
	if ch == nil {
		return nil, fmt.Errorf("clickhouse gorm connection is not initialized")
	}

	tableName := analyticsmodel.UserAccessLog{}.TableName()
	query := fmt.Sprintf(`
		SELECT user_agent, count() AS count
		FROM %s
		WHERE created_at >= ?
		GROUP BY user_agent
	`, tableName)

	type uaRow struct {
		UserAgent string
		Count     uint64
	}

	var rows []uaRow
	if err := ch.Raw(query, startTime).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("get browser distribution: %w", err)
	}

	browserCounts := make(map[string]uint64)
	for _, row := range rows {
		browser := ParseBrowserName(row.UserAgent)
		browserCounts[browser] += row.Count
	}

	result := make([]BrowserShare, 0, len(browserCounts))
	for browser, count := range browserCounts {
		result = append(result, BrowserShare{
			Browser: browser,
			Count:   count,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})
	return result, nil
}

// GetTopActiveUsers returns the most active users since startTime.
func GetTopActiveUsers(ctx context.Context, startTime time.Time, limit int) ([]TopUser, error) {
	if limit < 1 {
		limit = 10
	}

	ch := db.ChDB(ctx)
	if ch == nil {
		return nil, fmt.Errorf("clickhouse gorm connection is not initialized")
	}

	tableName := analyticsmodel.UserAccessLog{}.TableName()
	query := fmt.Sprintf(`
		SELECT user_id, count() AS count
		FROM %s
		WHERE created_at >= ? AND user_id > 0
		GROUP BY user_id
		ORDER BY count DESC
		LIMIT ?
	`, tableName)

	var users []TopUser
	if err := ch.Raw(query, startTime, limit).Scan(&users).Error; err != nil {
		return nil, fmt.Errorf("get top active users: %w", err)
	}
	return users, nil
}