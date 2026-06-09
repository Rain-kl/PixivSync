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
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// ParentBasedErrorAwareSampler 创建父级感知的概率采样器
// - 如果父 Span 已采样，则子 Span 也采样
// - 如果父 Span 未采样，则子 Span 也不采样
// - 如果是根 Span，按 samplingRate 概率采样
func ParentBasedErrorAwareSampler(samplingRate float64) sdktrace.Sampler {
	return sdktrace.ParentBased(
		sdktrace.TraceIDRatioBased(samplingRate),
	)
}
