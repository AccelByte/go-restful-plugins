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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emicklei/go-restful/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	testSpanTraceID   = "4bf92f3577b34da6a3ce929d0e0e4736"
	testHeaderTraceID = "0af7651916cd43dd8448eb211c80319c"
	testTraceparent   = "00-" + testHeaderTraceID + "-b7ad6b7169203331-01"
	testABTraceID     = "0123456789abcdef0123456789abcdef"
	abTraceIDKey      = "X-Ab-TraceID"
)

func contextWithTestSpan(t *testing.T, ctx context.Context) context.Context {
	t.Helper()

	traceID, err := oteltrace.TraceIDFromHex(testSpanTraceID)
	require.NoError(t, err)
	spanID, err := oteltrace.SpanIDFromHex("00f067aa0ba902b7")
	require.NoError(t, err)

	return oteltrace.ContextWithSpanContext(ctx, oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
	}))
}

// useTextMapPropagator sets the global propagator for one test, so tests that use it can't run in parallel.
func useTextMapPropagator(t *testing.T, propagator propagation.TextMapPropagator) {
	t.Helper()

	original := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagator)
	t.Cleanup(func() { otel.SetTextMapPropagator(original) })
}

// TestAccessLogTraceID runs AccessLog in a container and checks the trace_id written to the access log line.
//
//nolint:paralleltest // swaps the output of the package-level access log logger and sets the global propagator
func TestAccessLogTraceID(t *testing.T) {
	useTextMapPropagator(t, propagation.TraceContext{})

	var out bytes.Buffer
	originalOut := fullAccessLogLogger.Out
	fullAccessLogLogger.Out = &out
	t.Cleanup(func() { fullAccessLogLogger.Out = originalOut })

	// spanFilter puts a span on the request like observability-go-sdk's request filter does. The span
	// context is built here, in the parent test, so that the filter makes no assertions in the subtests.
	spanContext := oteltrace.SpanContextFromContext(contextWithTestSpan(t, context.Background()))
	spanFilter := func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		req.Request = req.Request.WithContext(oteltrace.ContextWithSpanContext(req.Request.Context(), spanContext))
		chain.ProcessFilter(req, resp)
	}

	newContainer := func(filters ...restful.FilterFunction) *restful.Container {
		ws := new(restful.WebService)
		ws.Route(ws.GET("/test").To(func(_ *restful.Request, resp *restful.Response) {
			resp.WriteHeader(http.StatusOK)
		}))
		container := restful.NewContainer()
		container.Add(ws)
		for _, filter := range filters {
			container.Filter(filter)
		}

		return container
	}

	tests := []struct {
		name      string
		container *restful.Container
		headers   map[string]string
		want      string
	}{
		{"span filter before AccessLog", newContainer(spanFilter, AccessLog), nil, testSpanTraceID},
		{"span filter after AccessLog", newContainer(AccessLog, spanFilter), nil, testSpanTraceID},
		{"no span, incoming trace context", newContainer(AccessLog), map[string]string{"traceparent": testTraceparent}, testHeaderTraceID},
		{"no span, no trace headers", newContainer(AccessLog), nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out.Reset()

			httpReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
			httpReq.Header.Set(abTraceIDKey, testABTraceID)
			for key, value := range tt.headers {
				httpReq.Header.Set(key, value)
			}
			tt.container.ServeHTTP(httptest.NewRecorder(), httpReq)

			line := out.String()
			// The field keeps its place, so parsers matching "trace_id=... namespace=" still match.
			assert.Contains(t, line, " trace_id="+tt.want+" namespace=")
			assert.NotContains(t, line, testABTraceID)
		})
	}
}
