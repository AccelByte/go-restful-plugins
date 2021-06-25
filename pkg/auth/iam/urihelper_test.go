// Copyright (c) 2021 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package iam

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDomain(t *testing.T) {
	assert.Equal(t, "http://example.net", getDomain("http://example.net"))
	assert.Equal(t, "http://example.net", getDomain("http://example.net/"))
	assert.Equal(t, "http://example.net", getDomain("http://example.net/path"))
	assert.Equal(t, "http://example.net", getDomain("http://example.net/path/path"))

	assert.Equal(t, "http://www.example.net", getDomain("http://www.example.net"))
	assert.Equal(t, "http://www.example.net", getDomain("http://www.example.net/"))
	assert.Equal(t, "http://www.example.net", getDomain("http://www.example.net/path"))
	assert.Equal(t, "http://www.example.net", getDomain("http://www.example.net/path/path"))

	assert.Equal(t, "https://www.example.net", getDomain("https://www.example.net"))
	assert.Equal(t, "https://www.example.net", getDomain("https://www.example.net/"))
	assert.Equal(t, "https://www.example.net", getDomain("https://www.example.net/path"))
	assert.Equal(t, "https://www.example.net", getDomain("https://www.example.net/path/path"))

	assert.Equal(t, "https://api.subdomain.example.net", getDomain("https://api.subdomain.example.net"))
	assert.Equal(t, "https://api.subdomain.example.net", getDomain("https://api.subdomain.example.net/"))
	assert.Equal(t, "https://api.subdomain.example.net", getDomain("https://api.subdomain.example.net/path"))
	assert.Equal(t, "https://api.subdomain.example.net", getDomain("https://api.subdomain.example.net/path/path"))

	assert.Equal(t, "www.example.net", getDomain("www.example.net"))
	assert.Equal(t, "www.example.net", getDomain("www.example.net/"))
	assert.Equal(t, "www.example.net", getDomain("www.example.net/path"))
	assert.Equal(t, "www.example.net", getDomain("www.example.net/path/path"))

	assert.Equal(t, "http://127.0.0.1", getDomain("http://127.0.0.1"))
	assert.Equal(t, "http://127.0.0.1", getDomain("http://127.0.0.1/"))
	assert.Equal(t, "http://127.0.0.1", getDomain("http://127.0.0.1/path"))
	assert.Equal(t, "http://127.0.0.1", getDomain("http://127.0.0.1/path/path"))

	assert.Equal(t, "http://127.0.0.1:8080", getDomain("http://127.0.0.1:8080"))
	assert.Equal(t, "http://127.0.0.1:8080", getDomain("http://127.0.0.1:8080/"))
	assert.Equal(t, "http://127.0.0.1:8080", getDomain("http://127.0.0.1:8080/path"))
	assert.Equal(t, "http://127.0.0.1:8080", getDomain("http://127.0.0.1:8080/path/path"))

	assert.Equal(t, "127.0.0.1", getDomain("127.0.0.1"))
	assert.Equal(t, "127.0.0.1", getDomain("127.0.0.1/"))
	assert.Equal(t, "127.0.0.1", getDomain("127.0.0.1/path"))
	assert.Equal(t, "127.0.0.1", getDomain("127.0.0.1/path/path"))

	assert.Equal(t, "127.0.0.1:8080", getDomain("127.0.0.1:8080"))
	assert.Equal(t, "127.0.0.1:8080", getDomain("127.0.0.1:8080/"))
	assert.Equal(t, "127.0.0.1:8080", getDomain("127.0.0.1:8080/path"))
	assert.Equal(t, "127.0.0.1:8080", getDomain("127.0.0.1:8080/path/path"))

	assert.Equal(t, "", getDomain(""))
}
