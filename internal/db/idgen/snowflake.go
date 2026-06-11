// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

// Package idgen 提供分布式 ID 生成器
package idgen

import (
	"log"

	"github.com/Rain-kl/Wavelet/internal/config"
	"github.com/bwmarrin/snowflake"
)

// 2025-12-01 00:00:00 UTC 的毫秒时间戳
const epoch int64 = 1764547200000

var node *snowflake.Node

func init() {
	snowflake.Epoch = epoch

	nodeID := config.Config.App.NodeID
	var err error
	node, err = snowflake.NewNode(nodeID)
	if err != nil {
		log.Fatalf("[Snowflake] init failed: %v\n", err)
	}
	log.Printf("[Snowflake] initialized with node ID: %d, epoch: 2025-12-01\n", nodeID)
}

// NextUint64ID 生成下一个分布式唯一 ID
func NextUint64ID() uint64 {
	val := node.Generate().Int64()
	if val < 0 {
		return 0
	}
	return uint64(val)
}
