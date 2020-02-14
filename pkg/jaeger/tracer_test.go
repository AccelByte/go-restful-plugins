package jaeger

import (
	"context"
	"testing"

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
