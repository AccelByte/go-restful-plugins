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
	logrus.SetLevel(logrus.DebugLevel)

	closer := InitGlobalTracer("", "", "test", "")
	defer closer.Close()

	expectedSpan, _ := opentracing.StartSpanFromContext(context.Background(), "test")

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
