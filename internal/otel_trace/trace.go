// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package otel_trace

import (
	"context"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// Tracer 全局 OpenTelemetry Tracer 实例
var Tracer trace.Tracer
var shutdownFuncs []func(context.Context) error

func init() {
	// 初始化 Propagator
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	// 初始化 Trace Provider
	tracerProvider, err := newTracerProvider()
	if err != nil {
		log.Fatalf("[Trace] init trace provider failed: %v", err)
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	// 初始化 Tracer
	Tracer = tracerProvider.Tracer("github.com/Rain-kl/Wavelet")
}

// Shutdown 关闭所有 Trace Provider
func Shutdown(ctx context.Context) {
	for _, fn := range shutdownFuncs {
		_ = fn(ctx)
	}
	shutdownFuncs = nil
}

// Start 创建一个新的 Trace Span
func Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return Tracer.Start(ctx, name, opts...)
}
