/*
Copyright 2025 linux.do
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

// Package db 提供数据库连接与基础设施
package db

import (
	"context"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/Rain-kl/Wavelet/internal/config"
)

const (
	clickhouseMaxExecTime        = 60 // ClickHouse 最大执行时间（秒）
	clickhouseReadTimeoutFactor  = 2  // ReadTimeout 为 DialTimeout 的倍数
)

var (
	// ChConn ClickHouse 连接实例
	ChConn driver.Conn
)

func init() {
	if !config.Config.ClickHouse.Enabled {
		return
	}

	cfg := config.Config.ClickHouse
	var err error

	// 配置 ClickHouse 连接
	ChConn, err = clickhouse.Open(&clickhouse.Options{
		Addr: cfg.Hosts,
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.Username,
			Password: cfg.Password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": clickhouseMaxExecTime,
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		DialTimeout:     time.Duration(cfg.DialTimeout) * time.Second,
		MaxOpenConns:    cfg.MaxOpenConn,
		MaxIdleConns:    cfg.MaxIdleConn,
		ConnMaxLifetime: time.Duration(cfg.ConnMaxLifetime) * time.Second,
		ReadTimeout:     time.Duration(cfg.DialTimeout*clickhouseReadTimeoutFactor) * time.Second,
		BlockBufferSize: cfg.BlockBufferSize,
	})

	if err != nil {
		log.Fatalf("[ClickHouse] init connection failed: %v\n", err)
	}

	// 测试连接
	if err = ChConn.Ping(context.Background()); err != nil {
		log.Fatalf("[ClickHouse] ping failed: %v\n", err)
	}

	log.Println("[ClickHouse] connection established successfully")
}
