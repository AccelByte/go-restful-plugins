// Copyright 2022 AccelByte Inc
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

package log

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/emicklei/go-restful/v3"
	"github.com/stretchr/testify/assert"
)

func TestGetRequestBody(t *testing.T) {
	t.Parallel()

	FullAccessLogEnabled = true
	FullAccessLogRequestBodyEnabled = true

	requestBody1, requestBodySize1 := getRequestBody(createDummyRequest("", ""), "", "")
	assert.Equal(t, "", requestBody1)
	assert.Equal(t, float32(0), requestBodySize1)

	requestBody2, requestBodySize2 := getRequestBody(createDummyRequest("{\"foo\":\"bar\"}", "application/json"), "application/json", "")
	assert.Equal(t, "{\"foo\":\"bar\"}", requestBody2)
	assert.Equal(t, float32(len("{\"foo\":\"bar\"}"))/kb, requestBodySize2)

	// uncompleted json
	requestBody3, requestBodySize3 := getRequestBody(createDummyRequest("{\"foo\":\"bar\"", "application/json"), "application/json", "")
	assert.Equal(t, "{\"foo\":\"bar\"", requestBody3)
	assert.Equal(t, float32(len("{\"foo\":\"bar\""))/kb, requestBodySize3)

	requestBody4, requestBodySize4 := getRequestBody(createDummyRequest("foo=bar&foo2=bar2", "application/x-www-form-urlencoded"), "application/x-www-form-urlencoded", "")
	assert.Equal(t, "foo=bar&foo2=bar2", requestBody4)
	assert.Equal(t, float32(len("foo=bar&foo2=bar2"))/kb, requestBodySize4)

	requestBody5, requestBodySize5 := getRequestBody(createDummyRequest("test test test", "text/plain"), "text/plain", "")
	assert.Equal(t, "test test test", requestBody5)
	assert.Equal(t, float32(len("test test test"))/kb, requestBodySize5)

	requestBody6, requestBodySize6 := getRequestBody(createDummyRequest("test test test", "unidentified-type"), "unidentified-type", "")
	assert.Equal(t, "", requestBody6)
	assert.Equal(t, float32(0), requestBodySize6)
}

func TestGetResponseBody(t *testing.T) {
	t.Parallel()

	FullAccessLogEnabled = true
	FullAccessLogRequestBodyEnabled = true

	responseBody1, responseBodySize1 := getResponseBody(createDummyResponse("", ""), "", "")
	assert.Equal(t, "", responseBody1)
	assert.Equal(t, float32(0), responseBodySize1)

	responseBody2, responseBodySize2 := getResponseBody(createDummyResponse("{\"foo\":\"bar\"}", "application/json"), "application/json", "")
	assert.Equal(t, "{\"foo\":\"bar\"}", responseBody2)
	assert.Equal(t, float32(len("{\"foo\":\"bar\"}"))/kb, responseBodySize2)

	// uncompleted json
	responseBody3, responseBodySize3 := getResponseBody(createDummyResponse("{\"foo\":\"bar\"", "application/json"), "application/json", "")
	assert.Equal(t, "{\"foo\":\"bar\"", responseBody3)
	assert.Equal(t, float32(len("{\"foo\":\"bar\""))/kb, responseBodySize3)

	responseBody4, responseBodySize4 := getResponseBody(createDummyResponse("foo=bar&foo2=bar2", "application/x-www-form-urlencoded"), "application/x-www-form-urlencoded", "")
	assert.Equal(t, "foo=bar&foo2=bar2", responseBody4)
	assert.Equal(t, float32(len("foo=bar&foo2=bar2"))/kb, responseBodySize4)

	responseBody5, responseBodySize5 := getResponseBody(createDummyResponse("test test test", "text/plain"), "text/plain", "")
	assert.Equal(t, "test test test", responseBody5)
	assert.Equal(t, float32(len("test test test"))/kb, responseBodySize5)

	responseBody6, responseBodySize6 := getResponseBody(createDummyResponse("test test test", "unidentified-type"), "unidentified-type", "")
	assert.Equal(t, "", responseBody6)
	assert.Equal(t, float32(0), responseBodySize6)
}

// nolint:paralleltest
func TestGetRequestBodyThatExceedMaxThreshold(t *testing.T) {
	FullAccessLogMaxBodySize = 1024 // 1KB
	FullAccessLogEnabled = true
	FullAccessLogRequestBodyEnabled = true

	largeData := `test test test test test test test test test test test test test test test test test test 
test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test 
test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test 
test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test 
test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test 
test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test 
test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test 
test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test 
test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test 
test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test 
test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test 
test test test test test test test test`

	requestBody, requestBodySize := getRequestBody(createDummyRequest(largeData, "text/plain"), "text/plain", "")
	assert.Equal(t, "data too large", requestBody)
	assert.Equal(t, float32(len(largeData))/kb, requestBodySize)
}

// nolint:paralleltest
func TestGetResponseBodyThatExceedMaxThreshold(t *testing.T) {
	FullAccessLogMaxBodySize = 1024 // 1KB
	FullAccessLogEnabled = true
	FullAccessLogRequestBodyEnabled = true

	largeData := `test test test test test test test test test test test test test test test test test test 
test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test 
test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test 
test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test 
test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test 
test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test 
test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test 
test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test 
test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test 
test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test 
test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test test 
test test test test test test test test`

	responseBody, responseBodySize := getResponseBody(createDummyResponse(largeData, "text/plain"), "text/plain", "")
	assert.Equal(t, "data too large", responseBody)
	assert.Equal(t, float32(len(largeData))/kb, responseBodySize)
}

func createDummyRequest(content string, contentType string) *restful.Request {
	request := &restful.Request{}
	request.Request = &http.Request{}
	request.Request.Header = map[string][]string{}
	request.Request.Header.Set("Content-Type", contentType)
	request.Request.Body = http.NoBody
	if content != "" {
		request.Request.Body = ioutil.NopCloser(strings.NewReader(content))
	}
	return request
}

func createDummyResponse(content string, contentType string) *ResponseWriterInterceptor {
	httpWriter := httptest.NewRecorder()
	httpWriter.Header().Set("Content-Type", contentType)
	response := &restful.Response{
		ResponseWriter: httpWriter,
	}

	return &ResponseWriterInterceptor{
		ResponseWriter: response.ResponseWriter,
		data:           []byte(content),
	}
}
