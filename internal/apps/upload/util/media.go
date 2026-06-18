// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"strings"

	"github.com/Rain-kl/Wavelet/internal/apps/upload/shared"
)

// IsImageExtension reports whether ext is a common image format.
func IsImageExtension(ext string) bool {
	for _, imgExt := range []string{"jpg", "jpeg", "png", "webp", "gif"} {
		if ext == imgExt {
			return true
		}
	}
	return false
}

// IsArchiveExtension reports whether ext is a common archive format.
func IsArchiveExtension(ext string) bool {
	for _, e := range []string{"zip", "rar", "7z", "tar", "gz", "tgz", "bz2", "xz"} {
		if ext == e {
			return true
		}
	}
	return false
}

// IsDocumentExtension reports whether ext is a common document format.
func IsDocumentExtension(ext string) bool {
	for _, e := range []string{"pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx", "txt", "md", "csv", "json", "yaml", "yml", "xml"} {
		if ext == e {
			return true
		}
	}
	return false
}

// NormalizeImageQuality normalizes the requested image quality query parameter.
func NormalizeImageQuality(quality string) string {
	switch strings.ToLower(quality) {
	case shared.ImageQualityLow, shared.ImageQualityMedium, shared.ImageQualityHigh:
		return strings.ToLower(quality)
	default:
		return shared.ImageQualityOrigin
	}
}