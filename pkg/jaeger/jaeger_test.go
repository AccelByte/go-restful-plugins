/*
 * Copyright 2019-2020 AccelByte Inc
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package jaeger

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emicklei/go-restful"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	jaegerclientgo "github.com/uber/jaeger-client-go"
)

func TestJaegerFilterWithZipkinMultipleHeaders(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	closer := InitGlobalTracer("", "", "test", "")
	defer closer.Close()

	ws := new(restful.WebService)
	ws.Filter(Filter())

	var span opentracing.Span

	traceID := "80f198ee56343ba864fe8b2a57d3eff7"
	parentSpanID := "05e3ac9a4f6e3b90"
	spanID := "e457b5a2e4d86bd1"
	sampled := "1"

	ws.Route(
		ws.GET("/namespace/{namespace}/user/{id}").
			Param(restful.PathParameter("namespace", "namespace")).
			Param(restful.PathParameter("id", "user ID")).
			To(func(request *restful.Request, response *restful.Response) {
				span = GetSpanFromRestfulContext(request.Request.Context())
			}))

	container := restful.NewContainer()
	container.Add(ws)

	req := httptest.NewRequest(http.MethodGet, "/namespace/abc/user/def", nil)
	req.Header.Set("X-Forwarded-For", "8.8.8.8")

	// more info about headers https://github.com/openzipkin/b3-propagation/blob/master/README.md#multiple-headers
	req.Header.Set("X-B3-TraceId", traceID)
	req.Header.Set("X-B3-ParentSpanId", parentSpanID)
	req.Header.Set("X-B3-SpanId", spanID)
	req.Header.Set("X-B3-Sampled", sampled)

	resp := httptest.NewRecorder()
	container.ServeHTTP(resp, req)

	require.NotNil(t, span)

	expectedTraceID := traceID
	assert.Equal(t, expectedTraceID, span.Context().(jaegerclientgo.SpanContext).TraceID().String())

	// as we create a new span - header span-id should be mapped into parent-span-id
	expectedParentID := spanID
	assert.Equal(t, expectedParentID, span.Context().(jaegerclientgo.SpanContext).ParentID().String())

	// as we create a new span - span-id should be new
	assert.NotEmpty(t, span.Context().(jaegerclientgo.SpanContext).ParentID().String())

	expectedSampled := true
	assert.Equal(t, expectedSampled, span.Context().(jaegerclientgo.SpanContext).IsSampled())
}

func TestJaegerFilterWithMissedZipkinHeaders(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	closer := InitGlobalTracer("", "", "test", "")
	defer closer.Close()

	ws := new(restful.WebService)
	ws.Filter(Filter())

	var span opentracing.Span

	ws.Route(
		ws.GET("/namespace/{namespace}/user/{id}").
			Param(restful.PathParameter("namespace", "namespace")).
			Param(restful.PathParameter("id", "user ID")).
			To(func(request *restful.Request, response *restful.Response) {
				span = GetSpanFromRestfulContext(request.Request.Context())
			}))

	container := restful.NewContainer()
	container.Add(ws)

	req := httptest.NewRequest(http.MethodGet, "/namespace/abc/user/def", nil)
	req.Header.Set("X-Forwarded-For", "8.8.8.8")

	resp := httptest.NewRecorder()
	container.ServeHTTP(resp, req)

	require.NotNil(t, span)

	assert.NotEmpty(t, span.Context().(jaegerclientgo.SpanContext).TraceID().String())
	assert.NotEmpty(t, span.Context().(jaegerclientgo.SpanContext).ParentID().String())
	assert.NotEmpty(t, span.Context().(jaegerclientgo.SpanContext).ParentID().String())
	assert.NotEmpty(t, span.Context().(jaegerclientgo.SpanContext).IsSampled())
}
