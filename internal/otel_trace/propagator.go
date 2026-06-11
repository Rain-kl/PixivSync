// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

// Package otel_trace 提供 OpenTelemetry 链路追踪封装工具
package otel_trace

import "go.opentelemetry.io/otel/propagation"

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}
