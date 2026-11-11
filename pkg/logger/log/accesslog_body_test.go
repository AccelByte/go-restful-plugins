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
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/emicklei/go-restful/v3"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// useFullAccessLog turns full access logging on with maxBodySize for one test and restores the settings after it.
// Tests that use it change package-level settings, so they can't run in parallel.
func useFullAccessLog(t *testing.T, maxBodySize int) {
	t.Helper()

	enabled := FullAccessLogEnabled
	requestBody := FullAccessLogRequestBodyEnabled
	responseBody := FullAccessLogResponseBodyEnabled
	maxSize := FullAccessLogMaxBodySize
	t.Cleanup(func() {
		FullAccessLogEnabled = enabled
		FullAccessLogRequestBodyEnabled = requestBody
		FullAccessLogResponseBodyEnabled = responseBody
		FullAccessLogMaxBodySize = maxSize
	})

	FullAccessLogEnabled = true
	FullAccessLogRequestBodyEnabled = true
	FullAccessLogResponseBodyEnabled = true
	FullAccessLogMaxBodySize = maxBodySize
}

// newTestInterceptor returns an interceptor that collects the body when captureBody is set.
func newTestInterceptor(captureBody bool) *ResponseWriterInterceptor {
	return &ResponseWriterInterceptor{
		ResponseWriter: httptest.NewRecorder(),
		captureBody:    captureBody,
	}
}

//nolint:paralleltest // changes the package-level access log settings
func TestResponseWriterInterceptor_CollectsAllWrites(t *testing.T) {
	useFullAccessLog(t, 10<<10)

	w := newTestInterceptor(true)
	buf := []byte(`{"foo":`)
	_, _ = w.Write(buf)
	_, _ = w.Write([]byte(`"bar"}`))
	buf[0] = 'X' // a handler reusing its buffer doesn't change what was collected

	assert.Equal(t, `{"foo":"bar"}`, string(w.data))
	assert.Equal(t, len(`{"foo":"bar"}`), w.size)
}

//nolint:paralleltest // changes the package-level access log settings
func TestResponseWriterInterceptor_StopsCollectingAtMaxBodySize(t *testing.T) {
	useFullAccessLog(t, 8)

	w := newTestInterceptor(true)
	_, _ = w.Write([]byte("12345"))
	_, _ = w.Write([]byte("67890"))

	assert.Empty(t, w.data, "a body larger than the limit is dropped, since it is logged as \""+bodyTooLarge+"\"")
	assert.Equal(t, 10, w.size)

	body, sizeKB := getResponseBody(w, "text/plain", "")
	assert.Equal(t, bodyTooLarge, body)
	assert.Equal(t, float32(10)/kb, sizeKB, "the size counts every byte written")
}

func TestResponseWriterInterceptor_DoesNotCollectWhenNotLogged(t *testing.T) {
	t.Parallel()

	w := newTestInterceptor(false)
	_, _ = w.Write([]byte("12345"))

	assert.Nil(t, w.data)
	assert.Equal(t, 5, w.size)
}

// captureLogs sends the access log and the global logrus output to buffers for one test.
func captureLogs(t *testing.T) (accessLog, otherLogs *bytes.Buffer) {
	t.Helper()

	accessLog, otherLogs = new(bytes.Buffer), new(bytes.Buffer)
	accessOut, stdOut := fullAccessLogLogger.Out, logrus.StandardLogger().Out
	fullAccessLogLogger.Out = accessLog
	logrus.SetOutput(otherLogs)
	t.Cleanup(func() {
		fullAccessLogLogger.Out = accessOut
		logrus.SetOutput(stdOut)
	})

	return accessLog, otherLogs
}

// A response written in several chunks is logged in full and masked.
//
//nolint:paralleltest // changes the package-level access log settings and logger outputs
func TestAccessLog_LogsWholeResponseWrittenInChunks(t *testing.T) {
	useFullAccessLog(t, 10<<10)
	accessLog, _ := captureLogs(t)

	ws := new(restful.WebService)
	ws.Route(ws.GET("/test").
		Filter(AccessLog).
		Filter(Attribute(Option{MaskedResponseFields: "password"})).
		To(func(_ *restful.Request, resp *restful.Response) {
			resp.Header().Set("Content-Type", mimeTypeJSON)
			resp.WriteHeader(http.StatusOK)
			_, _ = resp.Write([]byte(`{"user":"u1",`))
			_, _ = resp.Write([]byte(`"password":"hunter2"}`))
		}))
	container := restful.NewContainer()
	container.Add(ws)

	container.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil))

	assert.Contains(t, accessLog.String(), `response_body=AB[{"user":"u1","password":"******"}]AB`)
	assert.NotContains(t, accessLog.String(), "hunter2")
}

// failingWriter reports an invalid byte count, with err, as go-restful's CompressingResponseWriter does once it is
// closed.
type failingWriter struct {
	http.ResponseWriter
	n   int
	err error
}

func (f failingWriter) Write(_ []byte) (int, error) { return f.n, f.err }

// An invalid byte count from the wrapped writer doesn't panic, isn't counted, and is passed on to the caller as is.
//
//nolint:paralleltest // changes the package-level access log settings
func TestResponseWriterInterceptor_InvalidWrittenCount(t *testing.T) {
	useFullAccessLog(t, 10<<10)

	for _, n := range []int{-1, 100} {
		w := newTestInterceptor(true)
		w.ResponseWriter = failingWriter{ResponseWriter: httptest.NewRecorder(), n: n, err: errors.New("closed")}

		var written int
		assert.NotPanics(t, func() { written, _ = w.Write([]byte("abc")) }, "n=%d", n)
		assert.Equal(t, n, written, "n=%d: the caller gets the wrapped writer's count unchanged", n)
		assert.Equal(t, max(min(n, 3), 0), w.size, "n=%d", n)
		assert.Equal(t, max(min(n, 3), 0), len(w.data), "n=%d", n)
	}
}

// An empty response is logged as empty, without a minify warning, also with a negative size limit. With full access
// logging off it is logged as "-", as before.
//
//nolint:paralleltest // changes the package-level access log settings and logger outputs
func TestGetResponseBody_EmptyBody(t *testing.T) {
	useFullAccessLog(t, 10<<10)
	_, otherLogs := captureLogs(t)

	body, sizeKB := getResponseBody(newTestInterceptor(true), mimeTypeJSON, "")
	assert.Empty(t, body)
	assert.Zero(t, sizeKB)
	assert.Empty(t, otherLogs.String(), "an empty JSON body isn't a minify failure")

	FullAccessLogMaxBodySize = -1
	body, _ = getResponseBody(newTestInterceptor(true), mimeTypeJSON, "")
	assert.Empty(t, body, "not \""+bodyTooLarge+"\"")

	FullAccessLogEnabled = false
	body, _ = getResponseBody(newTestInterceptor(false), mimeTypeJSON, "")
	assert.Equal(t, "-", body)
}

// With go-restful's content encoding on, the logged body is the uncompressed one.
//
//nolint:paralleltest // changes the package-level access log settings and logger outputs
func TestAccessLog_LogsUncompressedBodyWithContentEncoding(t *testing.T) {
	useFullAccessLog(t, 10<<10)
	accessLog, _ := captureLogs(t)

	ws := new(restful.WebService)
	ws.Route(ws.GET("/test").
		Filter(AccessLog).
		To(func(_ *restful.Request, resp *restful.Response) {
			resp.Header().Set("Content-Type", mimeTypeJSON)
			resp.WriteHeader(http.StatusOK)
			_, _ = resp.Write([]byte(`{"a":`))
			_, _ = resp.Write([]byte(`1}`))
		}))
	container := restful.NewContainer()
	container.EnableContentEncoding(true)
	container.Add(ws)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	container.ServeHTTP(rec, req)

	assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"), "the response was compressed")
	assert.Contains(t, accessLog.String(), `response_body=AB[{"a":1}]AB`)
}

// AccessLog registered on both the container and the WebService wraps the writer once, so the body is copied once.
// Both filters still write their access log line, each with the whole body.
//
//nolint:paralleltest // changes the package-level access log settings and logger outputs
func TestAccessLog_RegisteredTwiceCopiesBodyOnce(t *testing.T) {
	useFullAccessLog(t, 10<<10)
	accessLog, _ := captureLogs(t)

	var nested bool

	ws := new(restful.WebService)
	ws.Filter(AccessLog)
	ws.Route(ws.GET("/test").To(func(_ *restful.Request, resp *restful.Response) {
		interceptor, _ := resp.ResponseWriter.(*ResponseWriterInterceptor)
		_, nested = interceptor.ResponseWriter.(*ResponseWriterInterceptor)

		resp.Header().Set("Content-Type", mimeTypeJSON)
		resp.WriteHeader(http.StatusOK)
		_, _ = resp.Write([]byte(`{"a":`))
		_, _ = resp.Write([]byte(`1}`))
	}))
	container := restful.NewContainer()
	container.Filter(AccessLog)
	container.Add(ws)

	container.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil))

	assert.False(t, nested, "the writer is wrapped once")
	assert.Equal(t, 2, strings.Count(accessLog.String(), `response_body=AB[{"a":1}]AB`))
}

// An interceptor that other code put in front of AccessLog isn't reused, since it doesn't collect the body.
//
//nolint:paralleltest // changes the package-level access log settings and logger outputs
func TestAccessLog_DoesNotReuseForeignInterceptor(t *testing.T) {
	useFullAccessLog(t, 10<<10)
	accessLog, _ := captureLogs(t)

	foreignWrapper := func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		resp.ResponseWriter = &ResponseWriterInterceptor{ResponseWriter: resp.ResponseWriter}
		chain.ProcessFilter(req, resp)
	}

	ws := new(restful.WebService)
	ws.Route(ws.GET("/test").
		Filter(foreignWrapper).
		Filter(AccessLog).
		To(func(_ *restful.Request, resp *restful.Response) {
			resp.Header().Set("Content-Type", mimeTypeJSON)
			resp.WriteHeader(http.StatusOK)
			_, _ = resp.Write([]byte(`{"a":1}`))
		}))
	container := restful.NewContainer()
	container.Add(ws)

	container.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil))

	assert.Contains(t, accessLog.String(), `response_body=AB[{"a":1}]AB`)
}
