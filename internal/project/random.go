// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package project

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// RandomHexString generates a random string of the provided length.
func RandomHexString(len int) (string, error) {
	b := make([]byte, len)
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("failed to generate random: %w", err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(b[:])), nil
}

// RandomBase64String encodes a random base64 string of a given length.
func RandomBase64String(len int) (string, error) {
	b := make([]byte, len)
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("failed to generate random: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// RandomBytes returns a byte slice of random values of the given length.
func RandomBytes(length int) ([]byte, error) {
	buf := make([]byte, length)
	n, err := rand.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random: %w", err)
	}
	if n < length {
		return nil, fmt.Errorf("insufficient bytes read: %v, expected %v", n, length)
	}
	return buf, nil
}
