// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package analytics provides ClickHouse data access for analytics tables.
package analytics

import (
	"context"
	"fmt"

	"github.com/Rain-kl/Wavelet/internal/db"
	analyticsmodel "github.com/Rain-kl/Wavelet/internal/model/analytics"
	"gorm.io/gorm"
)

// CountAccessLogs returns the number of access logs matching filter.
func CountAccessLogs(ctx context.Context, filter AccessLogFilter) (uint64, error) {
	ch := db.ChDB(ctx)
	if ch == nil {
		return 0, fmt.Errorf("clickhouse gorm connection is not initialized")
	}

	var count int64
	query := applyFilter(ch.Model(&analyticsmodel.UserAccessLog{}), filter)
	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count access logs: %w", err)
	}
	return safeUint64Count(count), nil
}

// ListAccessLogs returns paginated access logs and the total match count.
func ListAccessLogs(ctx context.Context, filter AccessLogFilter, page, pageSize int) ([]analyticsmodel.UserAccessLog, uint64, error) {
	ch := db.ChDB(ctx)
	if ch == nil {
		return nil, 0, fmt.Errorf("clickhouse gorm connection is not initialized")
	}

	if filter.UserIDs != nil && len(filter.UserIDs) == 0 {
		return []analyticsmodel.UserAccessLog{}, 0, nil
	}

	var total int64
	baseQuery := applyFilter(ch.Model(&analyticsmodel.UserAccessLog{}), filter)
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count access logs: %w", err)
	}
	if total == 0 {
		return []analyticsmodel.UserAccessLog{}, 0, nil
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var logs []analyticsmodel.UserAccessLog
	err := applyFilter(ch.Model(&analyticsmodel.UserAccessLog{}), filter).
		Order("created_at DESC, id DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&logs).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list access logs: %w", err)
	}

	return logs, safeUint64Count(total), nil
}

func safeUint64Count(count int64) uint64 {
	if count < 0 {
		return 0
	}
	return uint64(count)
}

func applyFilter(query *gorm.DB, filter AccessLogFilter) *gorm.DB {
	if filter.UserIDs != nil {
		if len(filter.UserIDs) == 0 {
			return query.Where("1 = 0")
		}
		query = query.Where("user_id IN ?", filter.UserIDs)
	}
	if filter.Path != "" {
		query = query.Where("path LIKE ?", "%"+filter.Path+"%")
	}
	if filter.StartTime != nil {
		query = query.Where("created_at >= ?", *filter.StartTime)
	}
	if filter.EndTime != nil {
		query = query.Where("created_at <= ?", *filter.EndTime)
	}
	return query
}