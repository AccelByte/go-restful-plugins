// Copyright 2026 AccelByte Inc
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

package log

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/emicklei/go-restful/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// trackingBody records how much of it was read.
type trackingBody struct {
	io.Reader
	read int
}

func (b *trackingBody) Read(p []byte) (int, error) {
	n, err := b.Reader.Read(p)
	b.read += n

	return n, err
}

func (b *trackingBody) Close() error { return nil }

// newBodyRequest returns a request with body and the given Content-Length (-1 when unknown, e.g. chunked).
func newBodyRequest(body io.ReadCloser, contentLength int64) *restful.Request {
	httpReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/test", nil)
	httpReq.Header.Set("Content-Type", mimeTypeJSON)
	httpReq.Body = body
	httpReq.ContentLength = contentLength

	return restful.NewRequest(httpReq)
}

// A body that isn't logged and has a known length isn't read; its size is its Content-Length.
//
//nolint:paralleltest // changes the package-level access log settings
func TestGetRequestBody_NotLoggedWithKnownLengthIsNotRead(t *testing.T) {
	useFullAccessLog(t, 10<<10)
	FullAccessLogEnabled = false

	content := `{"a":1}`
	body := &trackingBody{Reader: strings.NewReader(content)}
	req := newBodyRequest(body, int64(len(content)))

	logged, sizeKB := getRequestBody(req, mimeTypeJSON, "")

	assert.Equal(t, "-", logged)
	assert.Equal(t, float32(len(content))/kb, sizeKB)
	assert.Zero(t, body.read, "the body isn't read")
	assert.Same(t, body, req.Request.Body, "the handler gets the original body")
}

// An empty body that isn't logged is logged as empty, as before.
//
//nolint:paralleltest // changes the package-level access log settings
func TestGetRequestBody_NotLoggedEmptyBody(t *testing.T) {
	useFullAccessLog(t, 10<<10)
	FullAccessLogRequestBodyEnabled = false

	logged, sizeKB := getRequestBody(newBodyRequest(http.NoBody, 0), mimeTypeJSON, "")

	assert.Empty(t, logged)
	assert.Zero(t, sizeKB)
}

// A body of unknown length, e.g. a chunked one, is still read to measure it, and the handler gets the same bytes.
//
//nolint:paralleltest // changes the package-level access log settings
func TestGetRequestBody_NotLoggedWithUnknownLengthIsMeasured(t *testing.T) {
	useFullAccessLog(t, 10<<10)
	FullAccessLogEnabled = false

	content := `{"a":1}`
	req := newBodyRequest(io.NopCloser(strings.NewReader(content)), -1)

	logged, sizeKB := getRequestBody(req, mimeTypeJSON, "")
	assert.Equal(t, "-", logged)
	assert.Equal(t, float32(len(content))/kb, sizeKB)

	got, err := io.ReadAll(req.Request.Body)
	require.NoError(t, err)
	assert.Equal(t, content, string(got))
}

// End to end, with request bodies not logged: the access log line is the same for a body of known and of unknown
// length, and the handler reads the whole body in both cases.
//
//nolint:paralleltest // changes the package-level access log settings and logger outputs
func TestAccessLog_RequestBodyNotLogged(t *testing.T) {
	useFullAccessLog(t, 10<<10)
	FullAccessLogEnabled = false
	accessLog, _ := captureLogs(t)

	ws := new(restful.WebService).Consumes(restful.MIME_JSON)
	ws.Route(ws.POST("/test").Filter(AccessLog).To(func(req *restful.Request, resp *restful.Response) {
		var in map[string]string
		if err := req.ReadEntity(&in); err != nil || in["user"] != "u1" {
			resp.WriteHeader(http.StatusBadRequest)

			return
		}
		resp.WriteHeader(http.StatusOK)
	}))
	container := restful.NewContainer()
	container.Add(ws)

	content := `{"user":"u1"}`
	for _, contentLength := range []int64{int64(len(content)), -1} {
		accessLog.Reset()

		httpReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/test", strings.NewReader(content))
		httpReq.Header.Set("Content-Type", mimeTypeJSON)
		httpReq.ContentLength = contentLength
		rec := httptest.NewRecorder()
		container.ServeHTTP(rec, httpReq)

		assert.Equal(t, http.StatusOK, rec.Code, "Content-Length %d: the handler got the whole body", contentLength)
		assert.Contains(t, accessLog.String(), "request_body=AB[-]AB req_body_size_kb=0.013",
			"Content-Length %d", contentLength)
	}
}
