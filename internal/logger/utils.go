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
	"os"
	"path/filepath"
	"sync"

	"github.com/Rain-kl/Wavelet/internal/config"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logWriter         zapcore.WriteSyncer
	initLogWriterOnce sync.Once
	initLogWriterErr  error
)

// GetLogWriter 获取日志输出写入器
func GetLogWriter() (zapcore.WriteSyncer, error) {
	initLogWriterOnce.Do(func() {
		logWriter, initLogWriterErr = initWriter()
	})

	return logWriter, initLogWriterErr
}

// logDirPerm 日志目录权限
const logDirPerm = 0750

func initWriter() (zapcore.WriteSyncer, error) {
	logConfig := config.Config.Log

	if logConfig.Output == "file" {
		// 初始化日志目录
		logPath := logConfig.FilePath
		logDir := filepath.Dir(logPath)
		if err := os.MkdirAll(logDir, logDirPerm); err != nil {
			return nil, fmt.Errorf(errCreateLogFileDirFailed, err)
		}

		// 配置日志轮转
		logOutput := &lumberjack.Logger{
			Filename:   logPath,
			MaxSize:    logConfig.MaxSize,
			MaxBackups: logConfig.MaxBackups,
			MaxAge:     logConfig.MaxAge,
			Compress:   logConfig.Compress,
		}

		return zapcore.AddSync(logOutput), nil
	}

	return zapcore.AddSync(os.Stdout), nil
}

// getEncoder 获取日志编码器
func getEncoder() zapcore.Encoder {
	// 编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	if config.Config.Log.Format == "json" {
		return zapcore.NewJSONEncoder(encoderConfig)
	}
	return zapcore.NewConsoleEncoder(encoderConfig)
}

// getLogLevel 获取日志级别
func getLogLevel() zapcore.Level {
	level := config.Config.Log.Level

	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		log.Fatalf("[Logger] invalid log level: %s\n", level)
		return zapcore.InfoLevel
	}
}

func getTraceIDFields(ctx context.Context) []zap.Field {
	span := trace.SpanFromContext(ctx)
	spanContext := span.SpanContext()
	return []zap.Field{
		zap.String("traceID", spanContext.TraceID().String()),
		zap.String("spanID", spanContext.SpanID().String()),
	}
}
