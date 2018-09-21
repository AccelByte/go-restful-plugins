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
	"testing"

	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/assert"
)

func TestLog(t *testing.T) {
	ws := new(restful.WebService)
	ws.Filter(Log("test"))
	ws.Route(
		ws.GET("/namespace/{namespace}/user/{id}").
			Param(restful.PathParameter("namespace", "namespace")).
			Param(restful.PathParameter("id", "user ID")).
			To(func(request *restful.Request, response *restful.Response) {
				TargetUser(request, request.PathParameter("id"), request.PathParameter("namespace"))
				AdditionalFields(request, map[string]interface{}{"test": "test"})
				Info(request, 99, "success")
				response.WriteHeader(http.StatusOK)

				evt := getEvent(request)

				assert.Equal(t, evt.TargetNamespace, "abc")
				assert.Equal(t, evt.TargetUserID, "def")
				assert.Equal(t, evt.ID, 99)
				assert.Contains(t, evt.additionalFields, "test")
			}))

	container := restful.NewContainer()
	container.Add(ws)

	req := httptest.NewRequest(http.MethodGet, "/namespace/abc/user/def", nil)
	resp := httptest.NewRecorder()
	container.ServeHTTP(resp, req)
}
