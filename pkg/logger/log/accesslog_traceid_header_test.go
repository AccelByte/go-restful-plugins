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
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AccelByte/go-restful-plugins/v4/pkg/constant"
	"github.com/emicklei/go-restful/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// testSpanFilter puts the test span on the request, like observability-go-sdk's request filter does.
func testSpanFilter(t *testing.T) restful.FilterFunction {
	t.Helper()

	spanContext := oteltrace.SpanContextFromContext(contextWithTestSpan(t, context.Background()))

	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		req.Request = req.Request.WithContext(oteltrace.ContextWithSpanContext(req.Request.Context(), spanContext))
		chain.ProcessFilter(req, resp)
	}
}

func newTraceIDHeaderContainer(handler restful.RouteFunction, filters ...restful.FilterFunction) *restful.Container {
	ws := new(restful.WebService)
	ws.Route(ws.GET("/test").Produces(restful.MIME_JSON).To(handler))
	container := restful.NewContainer()
	container.Add(ws)
	for _, filter := range filters {
		container.Filter(filter)
	}

	return container
}

func writeOK(_ *restful.Request, resp *restful.Response) { resp.WriteHeader(http.StatusOK) }

// The X-Trace-Id response header carries the same trace ID as the access log's trace_id, and is left out when
// there is none.
//
//nolint:paralleltest // swaps the logger outputs and sets the global propagator
func TestAccessLog_TraceIDHeader(t *testing.T) {
	useTextMapPropagator(t, propagation.TraceContext{})
	out, _ := captureLogs(t)
	spanFilter := testSpanFilter(t)

	tests := []struct {
		name      string
		container *restful.Container
		headers   map[string]string
		want      string
	}{
		{"span filter before AccessLog", newTraceIDHeaderContainer(writeOK, spanFilter, AccessLog), nil, testSpanTraceID},
		{"span filter after AccessLog", newTraceIDHeaderContainer(writeOK, AccessLog, spanFilter), nil, testSpanTraceID},
		{"no span, incoming trace context", newTraceIDHeaderContainer(writeOK, AccessLog), map[string]string{"traceparent": testTraceparent}, testHeaderTraceID},
		{"no span, no trace headers", newTraceIDHeaderContainer(writeOK, AccessLog), nil, ""},
		{"incoming X-Trace-Id is ignored", newTraceIDHeaderContainer(writeOK, AccessLog), map[string]string{constant.TraceIDHeader: testABTraceID}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out.Reset()

			httpReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
			httpReq.Header.Set(abTraceIDKey, testABTraceID)
			for key, value := range tt.headers {
				httpReq.Header.Set(key, value)
			}
			rec := httptest.NewRecorder()
			tt.container.ServeHTTP(rec, httpReq)

			assert.Contains(t, out.String(), " trace_id="+tt.want+" namespace=")
			if tt.want == "" {
				assert.NotContains(t, rec.Header(), constant.TraceIDHeader)
			} else {
				assert.Equal(t, tt.want, rec.Header().Get(constant.TraceIDHeader))
			}
		})
	}
}

// The header is set however the handler writes its response, including with a span filter that runs after
// AccessLog.
//
//nolint:paralleltest // swaps the logger outputs
func TestAccessLog_TraceIDHeaderForEachWayOfWriting(t *testing.T) {
	captureLogs(t)
	spanFilter := testSpanFilter(t)

	tests := []struct {
		name    string
		handler restful.RouteFunction
		status  int
	}{
		{"WriteHeader", writeOK, http.StatusOK},
		{"Write without WriteHeader", func(_ *restful.Request, resp *restful.Response) {
			_, _ = resp.Write([]byte("body"))
		}, http.StatusOK},
		{"WriteEntity", func(_ *restful.Request, resp *restful.Response) {
			_ = resp.WriteEntity(map[string]string{"key": "value"})
		}, http.StatusOK},
		{"error response", func(_ *restful.Request, resp *restful.Response) {
			_ = resp.WriteErrorString(http.StatusInternalServerError, "failed")
		}, http.StatusInternalServerError},
		{"nothing written", func(_ *restful.Request, _ *restful.Response) {}, http.StatusOK},
		{"flush first", func(_ *restful.Request, resp *restful.Response) {
			resp.Flush()
			_, _ = resp.Write([]byte("event"))
		}, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// A real server, so that headers set after the response was sent can't show up in a recorder.
			server := httptest.NewServer(newTraceIDHeaderContainer(tt.handler, AccessLog, spanFilter))
			t.Cleanup(server.Close)

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/test", nil)
			require.NoError(t, err)
			resp, err := server.Client().Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			_, _ = io.Copy(io.Discard, resp.Body)

			assert.Equal(t, tt.status, resp.StatusCode)
			assert.Equal(t, testSpanTraceID, resp.Header.Get(constant.TraceIDHeader))
		})
	}
}

// A value the handler set itself is kept, and AccessLog registered twice sets the header once.
//
//nolint:paralleltest // swaps the logger outputs
func TestAccessLog_TraceIDHeaderSetOnce(t *testing.T) {
	captureLogs(t)
	spanFilter := testSpanFilter(t)

	t.Run("handler's own value is kept", func(t *testing.T) {
		container := newTraceIDHeaderContainer(func(_ *restful.Request, resp *restful.Response) {
			resp.Header().Set(constant.TraceIDHeader, "set-by-handler")
			resp.WriteHeader(http.StatusOK)
		}, spanFilter, AccessLog)

		rec := httptest.NewRecorder()
		container.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil))

		assert.Equal(t, []string{"set-by-handler"}, rec.Header().Values(constant.TraceIDHeader))
	})

	t.Run("AccessLog on the container and on the route", func(t *testing.T) {
		ws := new(restful.WebService)
		ws.Route(ws.GET("/test").Filter(AccessLog).To(writeOK))
		container := restful.NewContainer()
		container.Add(ws)
		container.Filter(spanFilter)
		container.Filter(AccessLog)

		rec := httptest.NewRecorder()
		container.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil))

		assert.Equal(t, []string{testSpanTraceID}, rec.Header().Values(constant.TraceIDHeader))
	})
}

// hijackRecorder is a recorder that can be hijacked, to check what the interceptor does around a hijack.
type hijackRecorder struct {
	*httptest.ResponseRecorder
}

func (h hijackRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	server, client := net.Pipe()
	_ = client.Close()

	return server, bufio.NewReadWriter(bufio.NewReader(server), bufio.NewWriter(server)), nil
}

// After a hijack the handler owns the connection, so the header is no longer touched.
func TestResponseWriterInterceptor_NoTraceIDHeaderAfterHijack(t *testing.T) {
	t.Parallel()

	req := restful.NewRequest(httptest.NewRequestWithContext(
		contextWithTestSpan(t, context.Background()), http.MethodGet, "/upgrade", nil))
	rec := hijackRecorder{httptest.NewRecorder()}
	w := &ResponseWriterInterceptor{ResponseWriter: rec, req: req}

	conn, _, err := w.Hijack()
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	w.setTraceIDHeader()

	assert.NotContains(t, rec.Header(), constant.TraceIDHeader)
}

// An interceptor that other code created has no request, and sets no header.
func TestResponseWriterInterceptor_NoTraceIDHeaderWithoutRequest(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	w := &ResponseWriterInterceptor{ResponseWriter: rec}
	w.WriteHeader(http.StatusOK)

	assert.NotContains(t, rec.Header(), constant.TraceIDHeader)
}

// A WebSocket-style upgrade behind AccessLog still works and the hijacked reply is the handler's own.
//
//nolint:paralleltest // swaps the logger outputs
func TestAccessLog_TraceIDHeaderNotOnHijackedReply(t *testing.T) {
	captureLogs(t)

	container := newTraceIDHeaderContainer(func(_ *restful.Request, resp *restful.Response) {
		conn, buf, err := resp.ResponseWriter.(http.Hijacker).Hijack()
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)

			return
		}
		defer conn.Close()
		_, _ = buf.WriteString("HTTP/1.1 200 OK\r\nContent-Length: 8\r\nConnection: close\r\n\r\nhijacked")
		_ = buf.Flush()
	}, AccessLog, testSpanFilter(t))

	served := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(served)
		container.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/test", nil)
	require.NoError(t, err)
	resp, err := server.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "hijacked", string(body))
	assert.Empty(t, resp.Header.Get(constant.TraceIDHeader))

	select {
	case <-served:
	case <-time.After(5 * time.Second):
		t.Fatal("the request didn't finish")
	}
}

// serveTraceIDHeader serves one GET /test through a real server and returns the response headers and body. It
// returns only once the server is done with the request, AccessLog included: after a hijack the client has its
// reply before that, and the test's cleanup must not swap the logger outputs while AccessLog still writes.
func serveTraceIDHeader(t *testing.T, container *restful.Container, headers map[string]string) (http.Header, string) {
	t.Helper()

	served := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(served)
		container.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	defer func() {
		select {
		case <-served:
		case <-time.After(5 * time.Second):
			t.Fatal("the request didn't finish")
		}
	}()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/test", nil)
	require.NoError(t, err)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := server.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp.Header, string(body)
}

// With go-restful's content encoding on, the writer underneath is a CompressingResponseWriter, which go-restful
// closes after the chain: the header is still sent, also when the handler writes nothing.
//
//nolint:paralleltest // swaps the logger outputs
func TestAccessLog_TraceIDHeaderWithContentEncoding(t *testing.T) {
	captureLogs(t)
	spanFilter := testSpanFilter(t)

	tests := []struct {
		name    string
		handler restful.RouteFunction
	}{
		{"WriteEntity", func(_ *restful.Request, resp *restful.Response) {
			_ = resp.WriteEntity(map[string]string{"key": "value"})
		}},
		{"nothing written", func(_ *restful.Request, _ *restful.Response) {}},
		{"flush first", func(_ *restful.Request, resp *restful.Response) {
			resp.Flush()
			_, _ = resp.Write([]byte("event"))
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := newTraceIDHeaderContainer(tt.handler, AccessLog, spanFilter)
			container.EnableContentEncoding(true)

			header, _ := serveTraceIDHeader(t, container, map[string]string{"Accept-Encoding": "gzip"})

			assert.Equal(t, "gzip", header.Get("Content-Encoding"))
			assert.Equal(t, testSpanTraceID, header.Get(constant.TraceIDHeader))
		})
	}
}

// When the hijack fails, as it does on HTTP/2, the error response the handler writes instead gets the header.
//
//nolint:paralleltest // swaps the logger outputs
func TestAccessLog_TraceIDHeaderAfterFailedHijack(t *testing.T) {
	captureLogs(t)

	container := newTraceIDHeaderContainer(func(_ *restful.Request, resp *restful.Response) {
		if _, _, err := http.NewResponseController(resp.ResponseWriter).Hijack(); err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
		}
	}, testSpanFilter(t), AccessLog)

	rec := httptest.NewRecorder() // the recorder can't hijack
	container.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, testSpanTraceID, rec.Header().Get(constant.TraceIDHeader))
}

// go-restful's Response.Hijack goes to the original writer, past the interceptor, so the interceptor can't see it.
// This checks that such a hijack still works behind AccessLog and that the handler's own reply goes out unchanged.
//
//nolint:paralleltest // swaps the logger outputs
func TestAccessLog_TraceIDHeaderNotOnRestfulHijack(t *testing.T) {
	captureLogs(t)

	container := newTraceIDHeaderContainer(func(_ *restful.Request, resp *restful.Response) {
		conn, buf, err := resp.Hijack()
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)

			return
		}
		defer conn.Close()
		_, _ = buf.WriteString("HTTP/1.1 200 OK\r\nContent-Length: 8\r\nConnection: close\r\n\r\nhijacked")
		_ = buf.Flush()
	}, AccessLog, testSpanFilter(t))

	header, body := serveTraceIDHeader(t, container, nil)

	assert.Equal(t, "hijacked", body)
	assert.Empty(t, header.Get(constant.TraceIDHeader))
}

// An interceptor that other code put in front of AccessLog isn't reused: AccessLog wraps it in its own, and the
// header is set once.
//
//nolint:paralleltest // swaps the logger outputs
func TestAccessLog_TraceIDHeaderWithForeignInterceptor(t *testing.T) {
	captureLogs(t)

	foreign := func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		resp.ResponseWriter = &ResponseWriterInterceptor{ResponseWriter: resp.ResponseWriter}
		chain.ProcessFilter(req, resp)
	}
	container := newTraceIDHeaderContainer(writeOK, testSpanFilter(t), foreign, AccessLog)

	rec := httptest.NewRecorder()
	container.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil))

	assert.Equal(t, []string{testSpanTraceID}, rec.Header().Values(constant.TraceIDHeader))
}
