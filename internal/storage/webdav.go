// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/Rain-kl/Wavelet/internal/httppool"
	"github.com/studio-b12/gowebdav"
)

type webDAVBackend struct {
	client   *gowebdav.Client
	basePath string
}

func newWebDAVBackend(cfg WebDAVConfig) (*webDAVBackend, error) {
	client := gowebdav.NewClient(strings.TrimRight(cfg.Endpoint, "/"), cfg.Username, cfg.Password)
	client.SetTransport(httppool.DefaultTransport())
	return &webDAVBackend{
		client:   client,
		basePath: strings.Trim(cfg.BasePath, "/"),
	}, nil
}

func (b *webDAVBackend) Put(_ context.Context, key string, body io.Reader, size int64, _ string) (PutResult, error) {
	key = b.key(key)
	if dir := path.Dir(key); dir != "." && dir != "/" {
		if err := b.client.MkdirAll(dir, storageDirPerm); err != nil {
			return PutResult{}, fmt.Errorf("create WebDAV directory: %w", err)
		}
	}
	if err := b.client.WriteStreamWithLength(key, body, size, storageFilePerm); err != nil {
		return PutResult{}, fmt.Errorf("put WebDAV object: %w", err)
	}
	return PutResult{Key: key}, nil
}

func (b *webDAVBackend) Get(_ context.Context, key string) (*Object, error) {
	key = b.key(key)
	info, err := b.client.Stat(key)
	if err != nil {
		return nil, fmt.Errorf("stat WebDAV object: %w", err)
	}
	body, err := b.client.ReadStream(key)
	if err != nil {
		return nil, fmt.Errorf("get WebDAV object: %w", err)
	}
	contentType := defaultContentType
	if typed, ok := info.(interface{ ContentType() string }); ok && typed.ContentType() != "" {
		contentType = typed.ContentType()
	}
	return &Object{Body: body, ContentLength: info.Size(), ContentType: contentType}, nil
}

func (b *webDAVBackend) Delete(_ context.Context, key string) error {
	if err := b.client.Remove(b.key(key)); err != nil {
		return fmt.Errorf("delete WebDAV object: %w", err)
	}
	return nil
}

func (b *webDAVBackend) Test(_ context.Context) error {
	if err := b.client.Connect(); err != nil {
		return fmt.Errorf("connect WebDAV: %w", err)
	}
	return nil
}

func (b *webDAVBackend) key(key string) string {
	return "/" + path.Join(b.basePath, strings.TrimLeft(key, "/"))
}
