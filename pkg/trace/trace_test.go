/*
 * Copyright 2019 AccelByte Inc
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

package trace

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/assert"
)

// nolint: dupl // most part of the test is identical
func TestFilterWithTraceID(t *testing.T) {
	ws := new(restful.WebService)
	ws.Filter(Filter())

	var traceID string

	ws.Route(
		ws.GET("/namespace/{namespace}/user/{id}").
			Param(restful.PathParameter("namespace", "namespace")).
			Param(restful.PathParameter("id", "user ID")).
			To(func(request *restful.Request, response *restful.Response) {
				traceID = request.HeaderParameter(TraceIDKey)
			}))

	container := restful.NewContainer()
	container.Add(ws)

	req := httptest.NewRequest(http.MethodGet, "/namespace/abc/user/def", nil)
	req.Header.Set("X-Forwarded-For", "8.8.8.8")
	req.Header.Set(TraceIDKey, "123456789")

	resp := httptest.NewRecorder()
	container.ServeHTTP(resp, req)

	assert.Equal(t, "123456789", traceID)
}

// nolint: dupl // most part of the test is identical
func TestFilterWithoutTraceID(t *testing.T) {
	ws := new(restful.WebService)
	ws.Filter(Filter())

	var traceID string

	ws.Route(
		ws.GET("/namespace/{namespace}/user/{id}").
			Param(restful.PathParameter("namespace", "namespace")).
			Param(restful.PathParameter("id", "user ID")).
			To(func(request *restful.Request, response *restful.Response) {
				traceID = request.HeaderParameter(TraceIDKey)
			}))

	container := restful.NewContainer()
	container.Add(ws)

	req := httptest.NewRequest(http.MethodGet, "/namespace/abc/user/def", nil)
	req.Header.Set("X-Forwarded-For", "8.8.8.8")

	resp := httptest.NewRecorder()
	container.ServeHTTP(resp, req)

	assert.NotNil(t, traceID)
	traceIDSplited := strings.Split(traceID, "-")
	traceUnix, err := strconv.ParseInt(traceIDSplited[0], 16, 64)
	assert.Nil(t, err)
	traceTime := time.Unix(traceUnix, 0)
	assert.Nil(t, validateIAMUUID(traceIDSplited[1]))
	assert.WithinDuration(t, time.Now().UTC(), traceTime, time.Second*2)
}

func validateIAMUUID(u string) error {
	notIAMFormat := strings.Contains(u, "-")
	if notIAMFormat {
		return errors.New("IAM's UUID doesn't contain dash (-)")
	}
	_, err := uuid.Parse(u)
	return err
}
