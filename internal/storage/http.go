// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

func getHTTPObject(ctx context.Context, baseURL, key string) (*Object, error) {
	objectURL, err := url.JoinPath(baseURL, key)
	if err != nil {
		return nil, fmt.Errorf("build CDN object URL: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, objectURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create CDN request: %w", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("get CDN object: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		_ = response.Body.Close()
		return nil, fmt.Errorf("get CDN object: unexpected status %d", response.StatusCode)
	}
	contentType := response.Header.Get("Content-Type")
	if contentType == "" {
		contentType = defaultContentType
	}
	return &Object{
		Body:          response.Body,
		ContentLength: response.ContentLength,
		ContentType:   contentType,
	}, nil
}
