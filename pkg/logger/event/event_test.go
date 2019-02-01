/*
 * Copyright 2018 AccelByte Inc
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

package event

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	iamAuth "github.com/AccelByte/go-restful-plugins/pkg/auth/iam"
	"github.com/AccelByte/iam-go-sdk"
	"github.com/dgrijalva/jwt-go"
	"github.com/emicklei/go-restful"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// nolint: dupl // most part of the test is identical
func TestInfoLog(t *testing.T) {
	ws := new(restful.WebService)
	ws.Filter(Log("test", ExtractNull))

	var evt *event

	ws.Route(
		ws.GET("/namespace/{namespace}/user/{id}").
			Param(restful.PathParameter("namespace", "namespace")).
			Param(restful.PathParameter("id", "user ID")).
			To(func(request *restful.Request, response *restful.Response) {
				TargetUser(request, request.PathParameter("id"), request.PathParameter("namespace"))
				AdditionalFields(request, map[string]interface{}{"test": "test"})
				Info(request, 99, "success")
				response.WriteHeader(http.StatusOK)

				evt = getEvent(request)
			}))

	container := restful.NewContainer()
	container.Add(ws)

	req := httptest.NewRequest(http.MethodGet, "/namespace/abc/user/def", nil)
	req.Header.Set("X-Forwarded-For", "8.8.8.8")

	resp := httptest.NewRecorder()
	container.ServeHTTP(resp, req)

	assert.Equal(t, "test", evt.Realm)
	assert.Equal(t, "abc", evt.TargetNamespace)
	assert.Equal(t, "def", evt.TargetUserID)
	assert.Equal(t, "8.8.8.8", evt.SourceIP)
	assert.Equal(t, 99, evt.ID)
	assert.Equal(t, []interface{}{"success"}, evt.message)
	assert.Equal(t, logrus.InfoLevel, evt.level)
	assert.Contains(t, evt.additionalFields, "test")
}

// nolint: dupl // most part of the test is identical
func TestWarnLog(t *testing.T) {
	ws := new(restful.WebService)
	ws.Filter(Log("test", ExtractNull))

	var evt *event

	ws.Route(
		ws.GET("/namespace/{namespace}/user/{id}").
			Param(restful.PathParameter("namespace", "namespace")).
			Param(restful.PathParameter("id", "user ID")).
			To(func(request *restful.Request, response *restful.Response) {
				TargetUser(request, request.PathParameter("id"), request.PathParameter("namespace"))
				AdditionalFields(request, map[string]interface{}{"test": "test"})
				Warn(request, 99, "success")
				response.WriteHeader(http.StatusOK)

				evt = getEvent(request)
			}))

	container := restful.NewContainer()
	container.Add(ws)

	req := httptest.NewRequest(http.MethodGet, "/namespace/abc/user/def", nil)
	req.Header.Set("X-Forwarded-For", "8.8.8.8")
	resp := httptest.NewRecorder()
	container.ServeHTTP(resp, req)

	assert.Equal(t, "test", evt.Realm)
	assert.Equal(t, "abc", evt.TargetNamespace)
	assert.Equal(t, "def", evt.TargetUserID)
	assert.Equal(t, "8.8.8.8", evt.SourceIP)
	assert.Equal(t, 99, evt.ID)
	assert.Equal(t, []interface{}{"success"}, evt.message)
	assert.Equal(t, logrus.WarnLevel, evt.level)
	assert.Contains(t, evt.additionalFields, "test")
}

// nolint: dupl // most part of the test is identical
func TestDebugLog(t *testing.T) {
	ws := new(restful.WebService)
	ws.Filter(Log("test", ExtractNull))

	var evt *event

	ws.Route(
		ws.GET("/namespace/{namespace}/user/{id}").
			Param(restful.PathParameter("namespace", "namespace")).
			Param(restful.PathParameter("id", "user ID")).
			To(func(request *restful.Request, response *restful.Response) {
				TargetUser(request, request.PathParameter("id"), request.PathParameter("namespace"))
				AdditionalFields(request, map[string]interface{}{"test": "test"})
				Debug(request, 99, "success")
				response.WriteHeader(http.StatusOK)

				evt = getEvent(request)
			}))

	container := restful.NewContainer()
	container.Add(ws)

	req := httptest.NewRequest(http.MethodGet, "/namespace/abc/user/def", nil)
	req.Header.Set("X-Forwarded-For", "8.8.8.8")
	resp := httptest.NewRecorder()
	container.ServeHTTP(resp, req)

	assert.Equal(t, "test", evt.Realm)
	assert.Equal(t, "abc", evt.TargetNamespace)
	assert.Equal(t, "def", evt.TargetUserID)
	assert.Equal(t, "8.8.8.8", evt.SourceIP)
	assert.Equal(t, 99, evt.ID)
	assert.Equal(t, []interface{}{"success"}, evt.message)
	assert.Equal(t, logrus.DebugLevel, evt.level)
	assert.Contains(t, evt.additionalFields, "test")
}

// nolint: dupl // most part of the test is identical
func TestErrorLog(t *testing.T) {
	ws := new(restful.WebService)
	ws.Filter(Log("test", ExtractNull))

	var evt *event

	ws.Route(
		ws.GET("/namespace/{namespace}/user/{id}").
			Param(restful.PathParameter("namespace", "namespace")).
			Param(restful.PathParameter("id", "user ID")).
			To(func(request *restful.Request, response *restful.Response) {
				TargetUser(request, request.PathParameter("id"), request.PathParameter("namespace"))
				AdditionalFields(request, map[string]interface{}{"test": "test"})
				Error(request, 99, "success")
				response.WriteHeader(http.StatusOK)

				evt = getEvent(request)
			}))

	container := restful.NewContainer()
	container.Add(ws)

	req := httptest.NewRequest(http.MethodGet, "/namespace/abc/user/def", nil)
	req.Header.Set("X-Forwarded-For", "8.8.8.8")
	resp := httptest.NewRecorder()
	container.ServeHTTP(resp, req)

	assert.Equal(t, "test", evt.Realm)
	assert.Equal(t, "abc", evt.TargetNamespace)
	assert.Equal(t, "def", evt.TargetUserID)
	assert.Equal(t, "8.8.8.8", evt.SourceIP)
	assert.Equal(t, 99, evt.ID)
	assert.Equal(t, []interface{}{"success"}, evt.message)
	assert.Equal(t, logrus.ErrorLevel, evt.level)
	assert.Contains(t, evt.additionalFields, "test")
}

// nolint: dupl // most part of the test is identical
func TestWithNoEventID(t *testing.T) {
	ws := new(restful.WebService)
	ws.Filter(Log("test", ExtractNull))

	var evt *event

	ws.Route(
		ws.GET("/namespace/{namespace}/user/{id}").
			Param(restful.PathParameter("namespace", "namespace")).
			Param(restful.PathParameter("id", "user ID")).
			To(func(request *restful.Request, response *restful.Response) {
				TargetUser(request, request.PathParameter("id"), request.PathParameter("namespace"))
				AdditionalFields(request, map[string]interface{}{"test": "test"})
				Info(request, 0, "success")
				response.WriteHeader(http.StatusOK)

				evt = getEvent(request)
			}))

	container := restful.NewContainer()
	container.Add(ws)

	req := httptest.NewRequest(http.MethodGet, "/namespace/abc/user/def", nil)
	req.Header.Set("X-Forwarded-For", "8.8.8.8")

	resp := httptest.NewRecorder()
	container.ServeHTTP(resp, req)

	assert.Equal(t, "test", evt.Realm)
	assert.Equal(t, "abc", evt.TargetNamespace)
	assert.Equal(t, "def", evt.TargetUserID)
	assert.Equal(t, "8.8.8.8", evt.SourceIP)
	assert.Equal(t, []interface{}{"success"}, evt.message)
	assert.Equal(t, logrus.InfoLevel, evt.level)
	assert.Contains(t, evt.additionalFields, "test")
}

// nolint: dupl // most part of the test is identical
func TestInfoLogWithJWTClaims(t *testing.T) {
	ws := new(restful.WebService)
	extract := func(req *restful.Request) (userID string, clientID string, namespace string) {
		claims := iamAuth.RetrieveJWTClaims(req)
		if claims != nil {
			return claims.Subject, claims.Audience, claims.Namespace
		}
		return "", "", ""
	}
	ws.Filter(Log("test", extract))

	var evt *event
	ws.Route(
		ws.GET("/namespace/{namespace}/user/{id}").
			Param(restful.PathParameter("namespace", "namespace")).
			Param(restful.PathParameter("id", "user ID")).
			To(func(request *restful.Request, response *restful.Response) {
				TargetUser(request, request.PathParameter("id"), request.PathParameter("namespace"))

				request.SetAttribute("JWTClaims", &iam.JWTClaims{
					Namespace: "testNamespace",
					StandardClaims: jwt.StandardClaims{
						Audience: "testClientID",
						Subject:  "testUserID",
					},
				})

				AdditionalFields(request, map[string]interface{}{"test": "test"})
				Info(request, 99, "success")
				response.WriteHeader(http.StatusOK)

				evt = getEvent(request)
			}))

	container := restful.NewContainer()
	container.Add(ws)

	req := httptest.NewRequest(http.MethodGet, "/namespace/abc/user/def", nil)
	req.Header.Set("X-Forwarded-For", "8.8.8.8")

	resp := httptest.NewRecorder()
	container.ServeHTTP(resp, req)

	assert.Equal(t, "test", evt.Realm)
	assert.Equal(t, "abc", evt.TargetNamespace)
	assert.Equal(t, "def", evt.TargetUserID)
	assert.Equal(t, "8.8.8.8", evt.SourceIP)
	assert.Equal(t, "testUserID", evt.UserID)
	assert.Equal(t, "testClientID", evt.ClientID)
	assert.Equal(t, "testNamespace", evt.Namespace)

	assert.Equal(t, 99, evt.ID)
	assert.Equal(t, []interface{}{"success"}, evt.message)
	assert.Equal(t, logrus.InfoLevel, evt.level)
	assert.Contains(t, evt.additionalFields, "test")
}

func TestFormatUTC(t *testing.T) {
	timeFormat := "2006-01-02T15:04:05.999Z07:00"
	timeSample := "2019-01-02T12:34:56.789+07:00"
	timeLogSample, _ := time.Parse(timeFormat, timeSample) // nolint: gosec // ignore error in test

	sampleLog := &logrus.Entry{
		Time: timeLogSample,
	}

	out, err := UTCFormatter{&logrus.TextFormatter{TimestampFormat: millisecondTimeFormat}}.Format(sampleLog)

	parts := strings.Split(string(out), " ")
	if len(parts) <= 0 {
		assert.FailNow(t, "log parts can't be zero")
	}
	var timeString string
	for _, part := range parts {
		fields := strings.Split(part, "=")
		if fields[0] == "time" {
			timeString = fields[1]
		}
	}

	assert.Nil(t, err, "error should be nil")
	assert.Equal(t, "\"2019-01-02T05:34:56.789Z\"", timeString, "time string is not equal")
}
