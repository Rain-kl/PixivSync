// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package util provides upload media helpers and image utilities.
package util

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/gif"  // Register GIF decoder for image.Decode
	_ "image/jpeg" // Register JPEG decoder for image.Decode
	_ "image/png"  // Register PNG decoder for image.Decode
	"io"
	"strings"

	"github.com/Rain-kl/Wavelet/internal/apps/upload/shared"
	"github.com/deepteams/webp"
	_ "golang.org/x/image/webp" // Register WebP decoder for image.Decode
)

// ValidateS3Key validates an S3 object key for safety.
func ValidateS3Key(key string) error {
	if key == "" {
		return errors.New(shared.ErrS3KeyRequired)
	}

	if len(key) > shared.MaxS3KeyLength {
		return fmt.Errorf(shared.ErrS3KeyTooLongFormat, shared.MaxS3KeyLength)
	}

	if strings.HasPrefix(key, "/") {
		return errors.New(shared.ErrS3KeyStartsWithSlash)
	}

	if strings.Contains(key, "\x00") {
		return errors.New(shared.ErrS3KeyContainsNullBytes)
	}

	return nil
}

// CompressImageToWebP decodes an image from srcReader and encodes it into WebP format
// using the specified quality (low -> 60, medium -> 75, high -> 85).
func CompressImageToWebP(srcReader io.Reader, quality string) ([]byte, error) {
	img, format, err := image.Decode(srcReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image (format: %s): %w", format, err)
	}

	var qualityScore float32
	switch strings.ToLower(quality) {
	case shared.ImageQualityLow:
		qualityScore = 60
	case shared.ImageQualityMedium:
		qualityScore = 75
	case shared.ImageQualityHigh, "":
		qualityScore = 85
	default:
		qualityScore = 85
	}

	var buf bytes.Buffer
	err = webp.Encode(&buf, img, &webp.EncoderOptions{
		Quality: qualityScore,
		Method:  4,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode WebP: %w", err)
	}

	return buf.Bytes(), nil
}