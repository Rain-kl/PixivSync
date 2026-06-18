// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package ingest

import (
	"errors"

	"github.com/Rain-kl/Wavelet/internal/apps/upload/shared"
)

// ErrForbidden indicates the caller is not allowed to mutate the upload record.
var ErrForbidden = errors.New("upload forbidden")

// ErrStorageReadOnly indicates the storage backend is in migration read-only mode.
var ErrStorageReadOnly = errors.New(shared.ErrStorageReadOnly)