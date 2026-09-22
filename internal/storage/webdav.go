// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/Rain-kl/Wavelet/pkg/httppool"
	"github.com/studio-b12/gowebdav"
)

type contextTransport struct {
	ctx    context.Context
	parent http.RoundTripper
}

func (t *contextTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.parent.RoundTrip(req.WithContext(t.ctx))
}

type webDAVBackend struct {
	endpoint string
	username string
	password string
	basePath string
}

func newWebDAVBackend(cfg WebDAVConfig) (*webDAVBackend, error) {
	basePath := strings.Trim(path.Clean("/"+cfg.BasePath), "/")
	if basePath == "." {
		basePath = ""
	}
	return &webDAVBackend{
		endpoint: strings.TrimRight(cfg.Endpoint, "/"),
		username: cfg.Username,
		password: cfg.Password,
		basePath: basePath,
	}, nil
}

func (b *webDAVBackend) newClient(ctx context.Context) *gowebdav.Client {
	client := gowebdav.NewClient(b.endpoint, b.username, b.password)
	client.SetTransport(&contextTransport{
		ctx:    ctx,
		parent: httppool.DefaultTransport(),
	})
	return client
}

func (b *webDAVBackend) Put(ctx context.Context, key string, body io.Reader, size int64, _ string) (PutResult, error) {
	target := b.targetPath(key)
	client := b.newClient(ctx)
	if dir := path.Dir(target); dir != "." && dir != "/" {
		if err := client.MkdirAll(dir, storageDirPerm); err != nil {
			return PutResult{}, fmt.Errorf("create WebDAV directory: %w", err)
		}
	}
	if err := client.WriteStreamWithLength(target, body, size, storageFilePerm); err != nil {
		return PutResult{}, fmt.Errorf("put WebDAV object: %w", err)
	}
	return PutResult{Key: b.relKey(key)}, nil
}

func (b *webDAVBackend) Get(ctx context.Context, key string) (*Object, error) {
	target := b.targetPath(key)
	client := b.newClient(ctx)
	info, err := client.Stat(target)
	if err != nil {
		return nil, fmt.Errorf("stat WebDAV object: %w", err)
	}
	body, err := client.ReadStream(target)
	if err != nil {
		return nil, fmt.Errorf("get WebDAV object: %w", err)
	}
	contentType := defaultContentType
	if typed, ok := info.(interface{ ContentType() string }); ok && typed.ContentType() != "" {
		contentType = typed.ContentType()
	}
	return &Object{Body: body, ContentLength: info.Size(), ContentType: contentType}, nil
}

func (b *webDAVBackend) Delete(ctx context.Context, key string) error {
	client := b.newClient(ctx)
	if err := client.Remove(b.targetPath(key)); err != nil {
		return fmt.Errorf("delete WebDAV object: %w", err)
	}
	return nil
}

func (b *webDAVBackend) Test(ctx context.Context) error {
	client := b.newClient(ctx)
	if err := client.Connect(); err != nil {
		return fmt.Errorf("connect WebDAV: %w", err)
	}
	return nil
}

// relKey extracts the clean, normalized, relative logical key (e.g. "uploads/2026/09/17/xxx.jpg")
// to be persisted in the database, stripping any driver-specific basePath and leading slashes.
func (b *webDAVBackend) relKey(key string) string {
	cleanKey := strings.Trim(path.Clean("/"+strings.ReplaceAll(key, "\\", "/")), "/")
	if cleanKey == "." {
		return ""
	}
	if b.basePath != "" {
		for cleanKey == b.basePath || strings.HasPrefix(cleanKey, b.basePath+"/") {
			cleanKey = strings.TrimPrefix(cleanKey, b.basePath)
			cleanKey = strings.TrimPrefix(cleanKey, "/")
		}
	}
	return cleanKey
}

// targetPath resolves any key (relative, legacy with basePath, or corrupted with duplicate basePath)
// into the absolute path used to access the object on the WebDAV server.
func (b *webDAVBackend) targetPath(key string) string {
	rel := b.relKey(key)
	if b.basePath == "" {
		if rel == "" {
			return "/"
		}
		return "/" + rel
	}
	if rel == "" {
		return "/" + b.basePath
	}
	return "/" + b.basePath + "/" + rel
}
