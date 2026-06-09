/*
Copyright 2026 Arctel.net

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

package cap

import (
	"fmt"
	"strings"
)

// fnv1a returns the 32-bit FNV-1a hash of a string
//
//nolint:mnd // FNV-1a 算法位移常量
func fnv1a(str string) uint32 {
	var hash uint32 = 2166136261
	for i := 0; i < len(str); i++ {
		hash ^= uint32(str[i])
		hash += (hash << 1) + (hash << 4) + (hash << 7) + (hash << 8) + (hash << 24)
	}
	return hash
}

// fnv1aResume resumes FNV-1a hashing from a given state
//
//nolint:mnd // FNV-1a 算法位移常量
func fnv1aResume(state uint32, str string) uint32 {
	h := state
	for i := 0; i < len(str); i++ {
		h ^= uint32(str[i])
		h += (h << 1) + (h << 4) + (h << 7) + (h << 8) + (h << 24)
	}
	return h
}

// prngFromHash generates a hex string of specified length using an initial hash state
//
//nolint:mnd // xorshift 算法位移常量
func prngFromHash(initialHash uint32, length int) string {
	state := initialHash
	var result strings.Builder
	for result.Len() < length {
		state ^= state << 13
		state ^= state >> 17
		state ^= state << 5
		hexStr := fmt.Sprintf("%08x", state)
		result.WriteString(hexStr)
	}
	return result.String()[:length]
}
