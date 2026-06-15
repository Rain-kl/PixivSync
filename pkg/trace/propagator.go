// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

// Package trace 提供 OpenTelemetry 链路追踪封装工具
package trace

import "go.opentelemetry.io/otel/propagation"

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}
