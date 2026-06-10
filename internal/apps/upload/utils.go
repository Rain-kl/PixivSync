// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package upload

import (
	"errors"
	"fmt"
	"strings"
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
