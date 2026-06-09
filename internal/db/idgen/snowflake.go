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
	return uint64(node.Generate().Int64())
}
