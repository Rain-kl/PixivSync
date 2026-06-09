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

package logger

import (
	"context"
	"fmt"
	"log"

	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *otelzap.Logger

// ringBufferCapacity 环形缓冲区容量
const ringBufferCapacity = 5000

// GlobalRingBuffer 全局日志环形缓冲区，供 Admin 日志查询和 WebSocket 推送使用
var GlobalRingBuffer *LogRingBuffer

func init() {
	logWriter, err := GetLogWriter()
	if err != nil {
		log.Fatalf("[Logger] get log writer err: %v\n", err)
	}

	// 初始化 ring buffer（保留最近 5000 行日志）
	GlobalRingBuffer = NewLogRingBuffer(ringBufferCapacity)

	// 使用 multi writer 同时写入原始输出和 ring buffer
	multiWriter := zapcore.NewMultiWriteSyncer(
		logWriter,
		zapcore.AddSync(GlobalRingBuffer),
	)

	zapLogger := zap.New(
		zapcore.NewCore(getEncoder(), multiWriter, getLogLevel()),
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	)
	logger = otelzap.New(
		zapLogger,
		otelzap.WithMinLevel(zapLogger.Level()),
	)

	fmt.Printf("[Logger] %s\n", logger.Level())
}

// DebugF 输出 Debug 级别日志
func DebugF(ctx context.Context, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	logger.Ctx(ctx).Debug(msg, getTraceIDFields(ctx)...)
}

// InfoF 输出 Info 级别日志
func InfoF(ctx context.Context, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	logger.Ctx(ctx).Info(msg, getTraceIDFields(ctx)...)
}

// WarnF 输出 Warn 级别日志
func WarnF(ctx context.Context, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	logger.Ctx(ctx).Warn(msg, getTraceIDFields(ctx)...)
}

// ErrorF 输出 Error 级别日志
func ErrorF(ctx context.Context, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	logger.Ctx(ctx).Error(msg, getTraceIDFields(ctx)...)
}
