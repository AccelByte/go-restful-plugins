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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AccelByte/go-jose/jwt"
	"github.com/AccelByte/go-restful-plugins/v4/pkg/logger/event"
	"github.com/AccelByte/iam-go-sdk"
	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/assert"
)

//nolint: dupl,funlen // most part of the test is identical
func TestExtractDefaultWithJWT(t *testing.T) {
	t.Parallel()

	ws := new(restful.WebService)

	var UserID, Namespace, traceID, sessionID string

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

				UserID, ClientIDs, Namespace, traceID, sessionID = ExtractDefault(request)
			}))

	container := restful.NewContainer()
	container.Add(ws)

	req := httptest.NewRequest(http.MethodGet, "/namespace/abc/user/def", nil)
	req.Header.Set("X-Forwarded-For", "8.8.8.8")
	req.Header.Set(traceIDKey, "testTraceID")
	req.Header.Set(sessionIDKey, "testSesssionID")

	resp := httptest.NewRecorder()
	container.ServeHTTP(resp, req)

	assert.Equal(t, "testUserID", UserID)
	assert.Equal(t, []string{"testClientID"}, ClientIDs)
	assert.Equal(t, "testNamespace", Namespace)
	assert.Equal(t, "testTraceID", traceID)
	assert.Equal(t, "testSesssionID", sessionID)
}

func TestExtractDefaultWithoutJWT(t *testing.T) {
	t.Parallel()

	ws := new(restful.WebService)

	var UserID, Namespace, traceID, sessionID string

	var ClientIDs []string

	ws.Filter(event.Log("test", "iam", ExtractDefault))
	ws.Route(
		ws.GET("/namespace/{namespace}/user/{id}").
			Param(restful.PathParameter("namespace", "namespace")).
			Param(restful.PathParameter("id", "user ID")).
			To(func(request *restful.Request, response *restful.Response) {
				UserID, ClientIDs, Namespace, traceID, sessionID = ExtractDefault(request)
			}))

	container := restful.NewContainer()
	container.Add(ws)

	req := httptest.NewRequest(http.MethodGet, "/namespace/abc/user/def", nil)
	req.Header.Set("X-Forwarded-For", "8.8.8.8")
	req.Header.Set(traceIDKey, "testTraceID")
	req.Header.Set(sessionIDKey, "testSesssionID")

	resp := httptest.NewRecorder()
	container.ServeHTTP(resp, req)

	assert.Equal(t, "", UserID)
	assert.Equal(t, []string{}, ClientIDs)
	assert.Equal(t, "", Namespace)
	assert.Equal(t, "testTraceID", traceID)
	assert.Equal(t, "testSesssionID", sessionID)
}
