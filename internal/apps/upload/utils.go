/*
Copyright 2025 linux.do
Modified by Arctel.net, 2026

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
