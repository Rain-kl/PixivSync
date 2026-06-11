// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package upload

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

	"github.com/deepteams/webp"
	_ "golang.org/x/image/webp" // Register WebP decoder for image.Decode
)

const maxS3KeyLength = 1024

// ValidateS3Key validates an S3 object key for safety.
func ValidateS3Key(key string) error {
	if key == "" {
		return errors.New(ErrS3KeyRequired)
	}

	if len(key) > maxS3KeyLength {
		return fmt.Errorf(ErrS3KeyTooLongFormat, maxS3KeyLength)
	}

	if strings.HasPrefix(key, "/") {
		return errors.New(ErrS3KeyStartsWithSlash)
	}

	if strings.Contains(key, "\x00") {
		return errors.New(ErrS3KeyContainsNullBytes)
	}

	return nil
}

// CompressImageToWebP decodes an image from srcReader and encodes it into WebP format
// using the specified quality (low -> 60, medium -> 75, high -> 85).
func CompressImageToWebP(srcReader io.Reader, quality string) ([]byte, error) {
	// Decode the image
	img, format, err := image.Decode(srcReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image (format: %s): %w", format, err)
	}

	// Determine quality
	var qualityScore float32
	switch strings.ToLower(quality) {
	case imageQualityLow:
		qualityScore = 60
	case imageQualityMedium:
		qualityScore = 75
	case imageQualityHigh, "":
		qualityScore = 85
	default:
		qualityScore = 85
	}

	// Encode to WebP
	var buf bytes.Buffer
	err = webp.Encode(&buf, img, &webp.EncoderOptions{
		Quality: qualityScore,
		Method:  4, // Default method
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode WebP: %w", err)
	}

	return buf.Bytes(), nil
}
