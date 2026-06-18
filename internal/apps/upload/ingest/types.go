// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package ingest provides the programmatic upload domain service for Wavelet.
package ingest

import (
	"io"

	"github.com/Rain-kl/Wavelet/internal/model"
)

// Policy controls how ingest handles hash collisions and record creation.
type Policy int

const (
	// PolicyCreate always stores a new object and creates a new upload record.
	PolicyCreate Policy = iota

	// PolicyDedupNewRecord reuses an existing object path on hash match but creates a new record and stats delta.
	PolicyDedupNewRecord

	// PolicyResolveExisting returns an existing upload on hash match without creating a record or stats delta.
	PolicyResolveExisting
)

// ObjectKeyFn builds the storage object key for a new upload.
type ObjectKeyFn func(id uint64, ext string) string

// Request describes a programmatic file ingest operation.
type Request struct {
	UserID uint64
	Type   string

	AccessMode *int
	Status     model.UploadStatus

	Reader    io.Reader
	Size      int64
	FileName  string
	MimeType  string
	Extension string
	Hash      string

	Metadata model.UploadMetadata
	Policy   Policy

	ObjectKeyFn ObjectKeyFn

	// SkipExtensionCheck bypasses the configured upload extension whitelist.
	SkipExtensionCheck bool
}

// Result reports the outcome of an ingest operation.
type Result struct {
	Upload   model.Upload
	Created  bool
	Stored   bool
	Resolved bool
}