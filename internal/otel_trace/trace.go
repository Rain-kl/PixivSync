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
