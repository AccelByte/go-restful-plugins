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

package jaeger

import (
	"context"
	"testing"

	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	jaegerclientgo "github.com/uber/jaeger-client-go"
)

func TestGetSpanFromRestfulContextWithoutSpan(t *testing.T) {
	t.Parallel()

	logrus.SetLevel(logrus.DebugLevel)

	closer := InitGlobalTracer("", "", "test", "")
	defer closer.Close()

	span := GetSpanFromRestfulContext(context.Background())
	require.NotNil(t, span)

	require.NotNil(t, span.Context())
	require.IsType(t, jaegerclientgo.SpanContext{}, span.Context())
	require.NotNil(t, span.Context().(jaegerclientgo.SpanContext))
	require.NotEmpty(t, span.Context().(jaegerclientgo.SpanContext).TraceID())
	assert.NotEmpty(t, span.Context().(jaegerclientgo.SpanContext).TraceID().String())
}

func TestGetSpanFromRestfulContextWithSpan(t *testing.T) {
	t.Parallel()

	logrus.SetLevel(logrus.DebugLevel)

	closer := InitGlobalTracer("", "", "test", "")
	defer closer.Close()

	expectedSpan, _ := StartSpanFromContext(context.Background(), "test")
	ctx := context.WithValue(context.Background(), spanContextKey, expectedSpan)

	span := GetSpanFromRestfulContext(ctx)
	require.NotNil(t, span)

	require.NotNil(t, span.Context())
	require.IsType(t, jaegerclientgo.SpanContext{}, span.Context())
	require.NotNil(t, span.Context().(jaegerclientgo.SpanContext))
	require.NotEmpty(t, span.Context().(jaegerclientgo.SpanContext).TraceID())

	assert.Equal(t,
		expectedSpan.Context().(jaegerclientgo.SpanContext).TraceID().String(),
		span.Context().(jaegerclientgo.SpanContext).TraceID().String(),
	)
}

func TestChildSpanFromRemoteSpan(t *testing.T) {
	t.Parallel()

	logrus.SetLevel(logrus.DebugLevel)

	closer := InitGlobalTracer("", "", "test", "")
	defer closer.Close()

	// limit access to a shared library
	globalTracerAccessMutex.Lock()
	expectedSpan, _ := opentracing.StartSpanFromContext(context.Background(), "test")
	globalTracerAccessMutex.Unlock()

	spanContextStr := expectedSpan.Context().(jaegerclientgo.SpanContext).String()

	span, _ := ChildSpanFromRemoteSpan(context.Background(), "test", spanContextStr)

	assert.Equal(t,
		expectedSpan.Context().(jaegerclientgo.SpanContext).TraceID().String(),
		span.Context().(jaegerclientgo.SpanContext).TraceID().String(),
	)

	assert.Equal(t,
		expectedSpan.Context().(jaegerclientgo.SpanContext).SpanID().String(),
		span.Context().(jaegerclientgo.SpanContext).ParentID().String(),
	)
}

func TestChildSpanFromRemoteSpan_EmptySpanContextString(t *testing.T) {
	t.Parallel()

	logrus.SetLevel(logrus.DebugLevel)

	closer := InitGlobalTracer("", "", "test", "")
	defer closer.Close()

	scope, _ := ChildSpanFromRemoteSpan(context.Background(), "test", "")

	assert.NotEmpty(t,
		scope.Context().(jaegerclientgo.SpanContext).TraceID().String(),
	)

	assert.NotEmpty(t,
		scope.Context().(jaegerclientgo.SpanContext).ParentID().String(),
	)
}

func TestGetSpanContextString_NotEmptySpanContext(t *testing.T) {
	t.Parallel()

	logrus.SetLevel(logrus.DebugLevel)

	closer := InitGlobalTracer("", "", "test", "")
	defer closer.Close()

	span := GetSpanFromRestfulContext(context.Background())
	require.NotNil(t, span)

	require.NotNil(t, span.Context())
	require.IsType(t, jaegerclientgo.SpanContext{}, span.Context())
	require.NotNil(t, span.Context().(jaegerclientgo.SpanContext))
	require.NotEmpty(t, span.Context().(jaegerclientgo.SpanContext).TraceID())
	assert.NotEmpty(t, span.Context().(jaegerclientgo.SpanContext).TraceID().String())

	spanContextString := GetSpanContextString(span)
	assert.NotEmpty(t, spanContextString)
}

func TestGetSpanContextString_EmptySpanContext(t *testing.T) {
	t.Parallel()

	logrus.SetLevel(logrus.DebugLevel)

	closer := InitGlobalTracer("", "", "test", "")
	defer closer.Close()

	span := opentracing.Span(nil)
	require.Nil(t, span)

	spanContextString := GetSpanContextString(span)
	assert.Empty(t, spanContextString)
}
