// Copyright 2019 AccelByte Inc
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

package util

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AccelByte/go-jose/jwt"
	"github.com/AccelByte/go-restful-plugins/v4/pkg/constant"
	"github.com/AccelByte/go-restful-plugins/v4/pkg/logger/event"
	"github.com/AccelByte/iam-go-sdk/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const testSpanTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"

// newTestRequest returns a request with a span in its context and an X-Ab-TraceID header, which
// ExtractDefault must not return as the trace ID.
func newTestRequest(t *testing.T) *http.Request {
	t.Helper()

	traceID, err := oteltrace.TraceIDFromHex(testSpanTraceID)
	require.NoError(t, err)
	spanID, err := oteltrace.SpanIDFromHex("00f067aa0ba902b7")
	require.NoError(t, err)
	ctx := oteltrace.ContextWithSpanContext(context.Background(), oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
	}))

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/namespace/abc/user/def", nil)
	req.Header.Set("X-Forwarded-For", "8.8.8.8")
	req.Header.Set("X-Ab-TraceID", "testTraceID")
	req.Header.Set(constant.SessionID, "testSesssionID")
	req.Header.Set(constant.FlightID, "testFlightID")

	return req
}

//nolint:dupl,funlen // most part of the test is identical
func TestExtractDefaultWithJWT(t *testing.T) {
	t.Parallel()

	ws := new(restful.WebService)

	var UserID, Namespace, traceID, sessionID, flightID string

	var ClientIDs []string

	ws.Filter(event.Log("test", "iam", ExtractDefault))
	ws.Route(
		ws.GET("/namespace/{namespace}/user/{id}").
			Param(restful.PathParameter("namespace", "namespace")).
			Param(restful.PathParameter("id", "user ID")).
			To(func(request *restful.Request, response *restful.Response) {
				request.SetAttribute("JWTClaims", &iam.JWTClaims{
					Namespace: "testNamespace",
					ClientID:  "testClientID",
					Claims: jwt.Claims{
						Subject: "testUserID",
					},
				})

				UserID, ClientIDs, Namespace, traceID, sessionID, flightID = ExtractDefault(request)
			}))

	container := restful.NewContainer()
	container.Add(ws)

	req := newTestRequest(t)

	resp := httptest.NewRecorder()
	container.ServeHTTP(resp, req)

	assert.Equal(t, "testUserID", UserID)
	assert.Equal(t, []string{"testClientID"}, ClientIDs)
	assert.Equal(t, "testNamespace", Namespace)
	assert.Equal(t, testSpanTraceID, traceID, "the OpenTelemetry trace ID, not X-Ab-TraceID")
	assert.Equal(t, "testSesssionID", sessionID)
	assert.Equal(t, "testFlightID", flightID)
}

func TestExtractDefaultWithoutJWT(t *testing.T) {
	t.Parallel()

	ws := new(restful.WebService)

	var UserID, Namespace, traceID, sessionID, flightID string

	var ClientIDs []string

	ws.Filter(event.Log("test", "iam", ExtractDefault))
	ws.Route(
		ws.GET("/namespace/{namespace}/user/{id}").
			Param(restful.PathParameter("namespace", "namespace")).
			Param(restful.PathParameter("id", "user ID")).
			To(func(request *restful.Request, response *restful.Response) {
				UserID, ClientIDs, Namespace, traceID, sessionID, flightID = ExtractDefault(request)
			}))

	container := restful.NewContainer()
	container.Add(ws)

	req := newTestRequest(t)

	resp := httptest.NewRecorder()
	container.ServeHTTP(resp, req)

	assert.Equal(t, "", UserID)
	assert.Equal(t, []string{}, ClientIDs)
	assert.Equal(t, "", Namespace)
	assert.Equal(t, testSpanTraceID, traceID, "the OpenTelemetry trace ID, not X-Ab-TraceID")
	assert.Equal(t, "testSesssionID", sessionID)
	assert.Equal(t, "testFlightID", flightID)
}
