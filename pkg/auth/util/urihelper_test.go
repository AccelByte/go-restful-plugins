// Copyright 2021 AccelByte Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// nolint:paralleltest
func TestGetDomain(t *testing.T) {
	assert.Equal(t, "http://example.net", GetDomain("http://example.net"))
	assert.Equal(t, "http://example.net", GetDomain("http://example.net/"))
	assert.Equal(t, "http://example.net", GetDomain("http://example.net/path"))
	assert.Equal(t, "http://example.net", GetDomain("http://example.net/path/path"))

	assert.Equal(t, "http://www.example.net", GetDomain("http://www.example.net"))
	assert.Equal(t, "http://www.example.net", GetDomain("http://www.example.net/"))
	assert.Equal(t, "http://www.example.net", GetDomain("http://www.example.net/path"))
	assert.Equal(t, "http://www.example.net", GetDomain("http://www.example.net/path/path"))

	assert.Equal(t, "https://www.example.net", GetDomain("https://www.example.net"))
	assert.Equal(t, "https://www.example.net", GetDomain("https://www.example.net/"))
	assert.Equal(t, "https://www.example.net", GetDomain("https://www.example.net/path"))
	assert.Equal(t, "https://www.example.net", GetDomain("https://www.example.net/path/path"))

	assert.Equal(t, "https://api.subdomain.example.net", GetDomain("https://api.subdomain.example.net"))
	assert.Equal(t, "https://api.subdomain.example.net", GetDomain("https://api.subdomain.example.net/"))
	assert.Equal(t, "https://api.subdomain.example.net", GetDomain("https://api.subdomain.example.net/path"))
	assert.Equal(t, "https://api.subdomain.example.net", GetDomain("https://api.subdomain.example.net/path/path"))

	assert.Equal(t, "www.example.net", GetDomain("www.example.net"))
	assert.Equal(t, "www.example.net", GetDomain("www.example.net/"))
	assert.Equal(t, "www.example.net", GetDomain("www.example.net/path"))
	assert.Equal(t, "www.example.net", GetDomain("www.example.net/path/path"))

	assert.Equal(t, "http://127.0.0.1", GetDomain("http://127.0.0.1"))
	assert.Equal(t, "http://127.0.0.1", GetDomain("http://127.0.0.1/"))
	assert.Equal(t, "http://127.0.0.1", GetDomain("http://127.0.0.1/path"))
	assert.Equal(t, "http://127.0.0.1", GetDomain("http://127.0.0.1/path/path"))

	assert.Equal(t, "http://127.0.0.1:8080", GetDomain("http://127.0.0.1:8080"))
	assert.Equal(t, "http://127.0.0.1:8080", GetDomain("http://127.0.0.1:8080/"))
	assert.Equal(t, "http://127.0.0.1:8080", GetDomain("http://127.0.0.1:8080/path"))
	assert.Equal(t, "http://127.0.0.1:8080", GetDomain("http://127.0.0.1:8080/path/path"))

	assert.Equal(t, "127.0.0.1", GetDomain("127.0.0.1"))
	assert.Equal(t, "127.0.0.1", GetDomain("127.0.0.1/"))
	assert.Equal(t, "127.0.0.1", GetDomain("127.0.0.1/path"))
	assert.Equal(t, "127.0.0.1", GetDomain("127.0.0.1/path/path"))

	assert.Equal(t, "127.0.0.1:8080", GetDomain("127.0.0.1:8080"))
	assert.Equal(t, "127.0.0.1:8080", GetDomain("127.0.0.1:8080/"))
	assert.Equal(t, "127.0.0.1:8080", GetDomain("127.0.0.1:8080/path"))
	assert.Equal(t, "127.0.0.1:8080", GetDomain("127.0.0.1:8080/path/path"))

	assert.Equal(t, "", GetDomain(""))
}
