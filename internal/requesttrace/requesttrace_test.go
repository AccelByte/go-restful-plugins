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

package requesttrace

import (
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

// newRequest returns a request that carries an X-Ab-TraceID, which must never be returned.
func newRequest(t *testing.T, withSpan bool, headers map[string]string) *restful.Request {
	t.Helper()

	httpReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	httpReq.Header.Set(abTraceIDKey, testABTraceID)
	for key, value := range headers {
		httpReq.Header.Set(key, value)
	}
	if withSpan {
		httpReq = httpReq.WithContext(contextWithTestSpan(t, httpReq.Context()))
	}

	req := restful.NewRequest(httpReq)
	req.SetAttribute(abTraceIDKey, testABTraceID)

	return req
}

//nolint:paralleltest // sets the global OpenTelemetry propagator
func TestTraceID(t *testing.T) {
	traceContext := propagation.TraceContext{}
	noPropagator := propagation.NewCompositeTextMapPropagator()
	traceparent := map[string]string{"traceparent": testTraceparent}

	tests := []struct {
		name       string
		propagator propagation.TextMapPropagator
		withSpan   bool
		headers    map[string]string
		want       string
	}{
		{"span in the request context", traceContext, true, nil, testSpanTraceID},
		{"span wins over incoming trace headers", traceContext, true, traceparent, testSpanTraceID},
		{"no span, incoming trace context", traceContext, false, traceparent, testHeaderTraceID},
		{"no span, no trace headers", traceContext, false, nil, ""},
		{"no span, no global propagator", noPropagator, false, traceparent, ""},
		{"no span, invalid trace header", traceContext, false, map[string]string{"traceparent": "00-0-0-01"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := otel.GetTextMapPropagator()
			otel.SetTextMapPropagator(tt.propagator)
			t.Cleanup(func() { otel.SetTextMapPropagator(original) })

			assert.Equal(t, tt.want, TraceID(newRequest(t, tt.withSpan, tt.headers)))
		})
	}

	t.Run("no request", func(t *testing.T) {
		assert.Empty(t, TraceID(nil))
		assert.Empty(t, TraceID(&restful.Request{}))
	})
}
