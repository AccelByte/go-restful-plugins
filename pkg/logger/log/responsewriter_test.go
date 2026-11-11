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
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// go-restful's Response.Flush reaches the real writer through the interceptor, and the response is still logged.
//
//nolint:paralleltest // changes the package-level access log settings and logger outputs
func TestResponseWriterInterceptor_Flush(t *testing.T) {
	useFullAccessLog(t, 10<<10)
	accessLog, _ := captureLogs(t)

	ws := new(restful.WebService)
	ws.Route(ws.GET("/stream").
		Filter(AccessLog).
		To(func(_ *restful.Request, resp *restful.Response) {
			resp.Header().Set("Content-Type", "text/plain")
			_, _ = resp.Write([]byte("event1;"))
			resp.Flush()
		}))
	container := restful.NewContainer()
	container.Add(ws)

	rec := httptest.NewRecorder()
	container.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/stream", nil))

	assert.True(t, rec.Flushed, "the flush reached the real writer")
	assert.Contains(t, accessLog.String(), "response_body=AB[event1;]AB")
}

// A route behind AccessLog can take over the connection, as a WebSocket upgrade does.
//
//nolint:paralleltest // changes the logger outputs
func TestResponseWriterInterceptor_Hijack(t *testing.T) {
	captureLogs(t)

	ws := new(restful.WebService)
	ws.Route(ws.GET("/upgrade").
		Filter(AccessLog).
		To(func(_ *restful.Request, resp *restful.Response) {
			hijacker, ok := resp.ResponseWriter.(http.Hijacker)
			if !ok {
				resp.WriteHeader(http.StatusInternalServerError)

				return
			}
			conn, buf, err := hijacker.Hijack()
			if err != nil {
				resp.WriteHeader(http.StatusInternalServerError)

				return
			}
			defer conn.Close()
			_, _ = buf.WriteString("HTTP/1.1 200 OK\r\nContent-Length: 8\r\nConnection: close\r\n\r\nhijacked")
			_ = buf.Flush()
		}))
	container := restful.NewContainer()
	container.Add(ws)

	// served is closed once the request, AccessLog included, is done: the client gets the hijacked reply before that.
	served := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(served)
		container.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/upgrade", nil)
	require.NoError(t, err)
	resp, err := server.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "hijacked", string(body))

	select {
	case <-served:
	case <-time.After(5 * time.Second):
		t.Fatal("the request didn't finish")
	}
}

func TestResponseWriterInterceptor_HijackNotSupported(t *testing.T) {
	t.Parallel()

	w := &ResponseWriterInterceptor{ResponseWriter: httptest.NewRecorder()} // the recorder can't hijack

	_, _, err := w.Hijack()
	assert.ErrorIs(t, err, http.ErrNotSupported)
}

// http.ResponseController reaches the real writer through Unwrap.
func TestResponseWriterInterceptor_Unwrap(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	w := &ResponseWriterInterceptor{ResponseWriter: rec}

	assert.Same(t, rec, w.Unwrap())
	require.NoError(t, http.NewResponseController(w).Flush())
	assert.True(t, rec.Flushed)
}

// unwrapOnly is a writer between the interceptor and the real one that can only be unwrapped.
type unwrapOnly struct{ http.ResponseWriter }

func (u unwrapOnly) Unwrap() http.ResponseWriter { return u.ResponseWriter }

// Flush reaches the real writer through writers that can only be unwrapped.
func TestResponseWriterInterceptor_FlushThroughUnwrap(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	w := &ResponseWriterInterceptor{ResponseWriter: unwrapOnly{rec}}

	w.Flush()
	assert.True(t, rec.Flushed)
}

// flushFailer is a writer whose flush fails, as net/http's does once the client is gone.
type flushFailer struct {
	http.ResponseWriter
	err error
}

func (f flushFailer) FlushError() error { return f.err }

// A flush error from the writer underneath reaches http.ResponseController.
func TestResponseWriterInterceptor_FlushError(t *testing.T) {
	t.Parallel()

	errGone := errors.New("client gone")
	w := &ResponseWriterInterceptor{ResponseWriter: flushFailer{ResponseWriter: httptest.NewRecorder(), err: errGone}}

	assert.ErrorIs(t, http.NewResponseController(w).Flush(), errGone)
	assert.ErrorIs(t, w.FlushError(), errGone)
}
