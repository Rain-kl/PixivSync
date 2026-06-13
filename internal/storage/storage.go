// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"
	"fmt"
	"io"
)

const (
	defaultContentType = "application/octet-stream"
	storageDirPerm     = 0o750
	storageFilePerm    = 0o600
)

// Object describes a readable stored object.
type Object struct {
	CachePath     string
	Body          io.ReadCloser
	ContentLength int64
	ContentType   string
}

// Backend defines storage operations used by the upload domain.
type Backend interface {
	Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) (string, error)
	Get(ctx context.Context, key string) (*Object, error)
	Delete(ctx context.Context, key string) error
	Test(ctx context.Context) error
}

var (
	// IsEnabledFunc preserves the legacy S3 test hook while tests migrate to backend injection.
	IsEnabledFunc = func() bool { return false }
	mockBackend   Backend
)

// Active returns the configured active driver and backend.
func Active(ctx context.Context) (Driver, Backend, error) {
	if IsEnabledFunc() && mockBackend != nil {
		return DriverS3, mockBackend, nil
	}
	cfg, err := LoadConfig(ctx)
	if err != nil {
		return "", nil, err
	}
	backend, err := NewBackend(ctx, cfg, cfg.Driver)
	return cfg.Driver, backend, err
}

// ForDriver returns the active or pending backend for an upload record.
func ForDriver(ctx context.Context, driver Driver) (Backend, error) {
	if driver == DriverS3 && mockBackend != nil {
		return mockBackend, nil
	}
	cfg, err := LoadConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cfg.Driver == driver {
		return NewBackend(ctx, cfg, driver)
	}
	return nil, fmt.Errorf("storage configuration for driver %q is unavailable", driver)
}

type functionBackend struct {
	put    func(context.Context, string, io.Reader, int64, string) error
	get    func(context.Context, string) (*Object, error)
	delete func(context.Context, string) error
}

func (b *functionBackend) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) (string, error) {
	if err := b.put(ctx, key, body, size, contentType); err != nil {
		return "", err
	}
	return key, nil
}

func (b *functionBackend) Get(ctx context.Context, key string) (*Object, error) {
	return b.get(ctx, key)
}

func (b *functionBackend) Delete(ctx context.Context, key string) error {
	return b.delete(ctx, key)
}

func (b *functionBackend) Test(context.Context) error {
	return nil
}

// MockStorage replaces object operations for package tests and returns a restore function.
func MockStorage(
	put func(context.Context, string, io.Reader, int64, string) error,
	get func(context.Context, string) (*Object, error),
	deleteObject func(context.Context, string) error,
) func() {
	previous := mockBackend
	mockBackend = &functionBackend{put: put, get: get, delete: deleteObject}
	return func() {
		mockBackend = previous
	}
}

// NewBackend constructs a concrete backend from configuration.
func NewBackend(ctx context.Context, cfg Config, driver Driver) (Backend, error) {
	if driver == DriverS3 && mockBackend != nil {
		return mockBackend, nil
	}
	switch driver {
	case DriverLocal:
		return newLocalBackend(cfg.Local)
	case DriverS3:
		return newS3Backend(ctx, cfg.S3)
	case DriverR2:
		return newR2Backend(ctx, cfg.R2)
	case DriverMinIO:
		return newS3Backend(ctx, cfg.MinIO)
	case DriverOSS:
		return newOSSBackend(cfg.OSS)
	case DriverWebDAV:
		return newWebDAVBackend(cfg.WebDAV)
	default:
		return nil, fmt.Errorf("unsupported storage driver %q", driver)
	}
}
