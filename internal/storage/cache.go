// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rain-kl/Wavelet/internal/config"
	"github.com/Rain-kl/Wavelet/internal/logger"
	"github.com/Rain-kl/Wavelet/internal/otel_trace"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/singleflight"
)

var localCacheEnabled = false
var localCacheDir = ""
var cacheFilePath = "%s/%s"
var cacheMetaFilePath = "%s/%s.meta"
var group singleflight.Group

type metaInfo struct {
	ContentType   string `json:"content_type"`
	ContentLength int64  `json:"content_length"`
}

// cacheDirPerm 缓存目录权限
const cacheDirPerm = 0755

func init() {
	cfg := config.Config.S3.LocalCache
	localCacheEnabled = cfg.Enabled && cfg.CacheDir != ""
	localCacheDir = strings.TrimSuffix(cfg.CacheDir, "/")
	if localCacheEnabled {
		if err := os.MkdirAll(cfg.CacheDir, cacheDirPerm); err != nil {
			log.Fatalf("[Storage] failed to create local cache directory: %v\n", err)
		}
	}
}

// GetObjectViaCache 通过本地缓存获取对象，缓存未命中时从 S3/CDN 拉取
func GetObjectViaCache(ctx context.Context, key string) (*ObjectInfo, error) {
	// 没有开启本地缓存
	if !localCacheEnabled {
		return GetObjectViaProxy(ctx, key)
	}

	// 初始化 Trace
	ctx, span := otel_trace.Start(ctx, "S3.GetObjectViaCache", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	// 检查本地缓存
	key = strings.TrimPrefix(key, "/")
	localPath := fmt.Sprintf(cacheFilePath, localCacheDir, key)
	metaPath := fmt.Sprintf(cacheMetaFilePath, localCacheDir, key)
	objInfo, err := getLocalCacheFile(ctx, localPath, metaPath)
	if err != nil {
		return nil, err
	}
	if objInfo != nil {
		return objInfo, nil
	}

	// 使用 singleflight 确保同一时间只有一个请求会触发 CDN 获取和本地缓存保存
	_, err, _ = group.Do(key, func() (interface{}, error) {
		ctx := context.WithoutCancel(ctx)

		// 没有缓存，通过 CDN 获取
		objInfo, err := GetObjectViaProxy(ctx, key)
		if err != nil {
			return nil, err
		}

		// 保存到本地
		if err := saveToLocalCache(ctx, localPath, metaPath, objInfo); err != nil {
			return nil, err
		}

		return nil, nil
	})
	if err != nil {
		logger.ErrorF(ctx, "Failed to get object via singleflight for key %s: %v", key, err)
		return nil, LocalCacheError{}
	}

	return GetObjectViaCache(ctx, key)
}

func getLocalCacheFile(ctx context.Context, localPath, metaPath string) (*ObjectInfo, error) {
	_, span := otel_trace.Start(ctx, "S3.GetLocalCacheFile", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	// 尝试打开本地缓存文件
	file, err := os.Open(localPath) //nolint:gosec // localPath is internally managed cache path
	if err == nil {
		defer func() { _ = file.Close() }()
	}

	// 文件不存在
	if err != nil && os.IsNotExist(err) {
		return nil, nil
	}

	// 判断是否为其他异常
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// 读取元信息
	metaData, err := os.ReadFile(metaPath) //nolint:gosec // metaPath is internally managed cache path

	// 文件不存在
	if err != nil && os.IsNotExist(err) {
		return nil, nil
	}

	// 判断是否为其他异常
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// 解析元信息
	meta := &metaInfo{}
	if err := json.Unmarshal(metaData, meta); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return &ObjectInfo{CachePath: localPath, ContentLength: meta.ContentLength, ContentType: meta.ContentType}, nil
}

func saveToLocalCache(ctx context.Context, localPath, metaPath string, objInfo *ObjectInfo) error {
	_, span := otel_trace.Start(ctx, "S3.SaveToLocalCache", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	// 创建目录
	localDir := filepath.Dir(localPath)
	if err := os.MkdirAll(localDir, cacheDirPerm); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	// 创建文件
	if err := saveFile(localPath, objInfo.Body); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	// 创建元信息文件
	meta := &metaInfo{ContentType: objInfo.ContentType, ContentLength: objInfo.ContentLength}
	metaData, err := json.Marshal(meta)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if err := saveFile(metaPath, bytes.NewReader(metaData)); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}

func saveFile(localPath string, data io.Reader) error {
	// 创建临时文件
	tempFile, err := os.CreateTemp(filepath.Dir(localPath), "cache_temp_*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tempFile.Name()) }()

	// 将内容写入临时文件
	if _, err := tempFile.ReadFrom(data); err != nil {
		return err
	}

	// 确保数据写入磁盘
	if err := tempFile.Sync(); err != nil {
		return err
	}

	// 关闭临时文件
	if err := tempFile.Close(); err != nil {
		return err
	}

	// 将临时文件重命名为最终文件
	if err := os.Rename(tempFile.Name(), localPath); err != nil {
		return err
	}

	return nil
}
