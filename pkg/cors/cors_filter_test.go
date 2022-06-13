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

package cors

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emicklei/go-restful/v3"
	"github.com/stretchr/testify/assert"
)

func createDummyRequest() *restful.Request {
	return &restful.Request{
		Request: &http.Request{
			Header: map[string][]string{},
		},
	}
}

func createDummyResponse() *restful.Response {
	httpWriter := httptest.NewRecorder()
	response := &restful.Response{
		ResponseWriter: httpWriter,
	}

	return response
}

func createDummyFilterChain() *restful.FilterChain {
	return &restful.FilterChain{
		Filters: make([]restful.FilterFunction, 0),
		Target: func(req *restful.Request, resp *restful.Response) {
			fmt.Println("[FilterChain] dummy target func invoked")
		},
	}
}

func TestIsOriginAllowed(t *testing.T) {
	// TEST 1: Single allowed domain
	cors := CrossOriginResourceSharing{
		AllowedDomains: []string{"https://www.example.io"},
	}
	assert.True(t, cors.isOriginAllowed("https://www.example.io"))
	assert.False(t, cors.isOriginAllowed("https://www.example.com"))
	assert.False(t, cors.isOriginAllowed("https://www.example.io.something"))
	assert.False(t, cors.isOriginAllowed("https://www.example.io.something.io"))

	// TEST 2: IP domain
	corsWithIPDomain := CrossOriginResourceSharing{
		AllowedDomains: []string{"127.0.0.1"},
	}
	assert.True(t, corsWithIPDomain.isOriginAllowed("127.0.0.1"))
	assert.False(t, corsWithIPDomain.isOriginAllowed("127.0.0.2"))
	assert.False(t, corsWithIPDomain.isOriginAllowed("127.0.0.1.1"))
	assert.False(t, corsWithIPDomain.isOriginAllowed("https://www.example.io"))
	assert.False(t, corsWithIPDomain.isOriginAllowed("https://www.example.com"))
	assert.False(t, corsWithIPDomain.isOriginAllowed("https://www.example.io.something"))
	assert.False(t, corsWithIPDomain.isOriginAllowed("https://www.example.io.something.io"))

	// TEST 3: Multiple allowed domains
	corsWithMultipleAllowedDomain := CrossOriginResourceSharing{
		AllowedDomains: []string{"https://www.example.io", "https://www.example.com", "127.0.0.1"},
	}
	assert.True(t, corsWithMultipleAllowedDomain.isOriginAllowed("https://www.example.io"))
	assert.True(t, corsWithMultipleAllowedDomain.isOriginAllowed("https://www.example.com"))
	assert.True(t, corsWithMultipleAllowedDomain.isOriginAllowed("127.0.0.1"))
	assert.False(t, corsWithMultipleAllowedDomain.isOriginAllowed("https://www.example.io.something"))
	assert.False(t, corsWithMultipleAllowedDomain.isOriginAllowed("https://www.example.io.something.io"))

	// TEST 4: Allowed domain is wildcard
	corsWithWildcardAllowedDomain := CrossOriginResourceSharing{
		AllowedDomains: []string{"*"},
	}
	assert.True(t, corsWithWildcardAllowedDomain.isOriginAllowed("https://www.example.io"))
	assert.True(t, corsWithWildcardAllowedDomain.isOriginAllowed("https://www.example.com"))
	assert.True(t, corsWithWildcardAllowedDomain.isOriginAllowed("https://www.example.io.something"))
	assert.True(t, corsWithWildcardAllowedDomain.isOriginAllowed("https://www.example.io.something.io"))

	// TEST 5: Allowed domains with regex
	corsWithRegex := CrossOriginResourceSharing{
		AllowedDomains: []string{"re:https://([a-z0-9]+[.])*example.io$", "https://www.example.com"},
	}
	assert.True(t, corsWithRegex.isOriginAllowed("https://www.example.io"))
	assert.True(t, corsWithRegex.isOriginAllowed("https://subdomain.example.io"))
	assert.True(t, corsWithRegex.isOriginAllowed("https://www.example.com"))
	assert.False(t, corsWithRegex.isOriginAllowed("https://subdomain.example.com"))
	assert.False(t, corsWithRegex.isOriginAllowed("https://www.example.net"))
	assert.False(t, corsWithRegex.isOriginAllowed("https://subdomain.example.io.something"))
	assert.False(t, corsWithRegex.isOriginAllowed("https://www.example.io.something.io"))
}

func TestIsValidAccessControlRequestMethod(t *testing.T) {
	cors := CrossOriginResourceSharing{
		AllowedMethods: []string{"GET", "POST"},
	}
	assert.True(t, cors.isValidAccessControlRequestMethod("GET"))
	assert.True(t, cors.isValidAccessControlRequestMethod("POST"))
	assert.False(t, cors.isValidAccessControlRequestMethod("DELETE"))
}

func TestIsValidAccessControlRequestHeader(t *testing.T) {
	cors := CrossOriginResourceSharing{
		AllowedHeaders: []string{"Content-Type", "Accept", "Device-Id"},
	}
	assert.True(t, cors.isValidAccessControlRequestHeader("Content-Type"))
	assert.True(t, cors.isValidAccessControlRequestHeader("Accept"))
	assert.False(t, cors.isValidAccessControlRequestHeader("Something"))
}

func TestPreflightRequest(t *testing.T) {
	cors := CrossOriginResourceSharing{
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type", "Accept", "Device-Id"},
	}

	// TEST 1: Success
	req1 := createDummyRequest()
	req1.Request.Header.Set("Access-Control-Request-Method", "GET")
	req1.Request.Header.Set("Access-Control-Request-Headers", "Content-Type,Accept")
	resp1 := createDummyResponse()
	cors.doPreflightRequest(req1, resp1)

	assert.NotEmpty(t, resp1.Header().Get("Access-Control-Allow-Methods"))
	assert.NotEmpty(t, resp1.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "GET,POST", resp1.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type,Accept", resp1.Header().Get("Access-Control-Allow-Headers"))

	// TEST 2: Request Method not allowed
	req2 := createDummyRequest()
	req2.Request.Header.Set("Access-Control-Request-Method", "DELETE")
	req2.Request.Header.Set("Access-Control-Request-Headers", "Content-Type,Accept")
	resp2 := createDummyResponse()
	cors.doPreflightRequest(req2, resp2)

	assert.Empty(t, resp2.Header().Get("Access-Control-Allow-Methods"))
	assert.Empty(t, resp2.Header().Get("Access-Control-Allow-Headers"))

	// TEST 3: Request Header not allowed
	req3 := createDummyRequest()
	req3.Request.Header.Set("Access-Control-Request-Method", "GET")
	req3.Request.Header.Set("Access-Control-Request-Headers", "Something")
	resp3 := createDummyResponse()
	cors.doPreflightRequest(req3, resp3)

	assert.Empty(t, resp3.Header().Get("Access-Control-Allow-Methods"))
	assert.Empty(t, resp3.Header().Get("Access-Control-Allow-Headers"))
}

func TestPreflightRequest_MaxAgeConfigured(t *testing.T) {
	corsWithMaxAge := CrossOriginResourceSharing{
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type", "Accept", "Device-Id"},
		MaxAge:         3600,
	}

	// TEST 1: Success
	req1 := createDummyRequest()
	req1.Request.Header.Set("Access-Control-Request-Method", "GET")
	req1.Request.Header.Set("Access-Control-Request-Headers", "Content-Type,Accept")
	resp1 := createDummyResponse()

	corsWithMaxAge.doPreflightRequest(req1, resp1)

	assert.NotEmpty(t, resp1.Header().Get("Access-Control-Allow-Methods"))
	assert.NotEmpty(t, resp1.Header().Get("Access-Control-Allow-Headers"))
	assert.NotEmpty(t, resp1.Header().Get("Access-Control-Max-Age"))
	assert.Equal(t, "GET,POST", resp1.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type,Accept", resp1.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "3600", resp1.Header().Get("Access-Control-Max-Age"))
}

func TestSetOptionHeaders(t *testing.T) {
	cors := CrossOriginResourceSharing{
		ExposeHeaders:  []string{"Authorization", "AB-Session"},
		CookiesAllowed: true,
	}

	// TEST 1: Success
	req1 := createDummyRequest()
	req1.Request.Header.Set("Origin", "https://www.example.io")
	resp1 := createDummyResponse()
	cors.setOptionsHeaders(req1, resp1)

	assert.NotEmpty(t, resp1.Header().Get("Access-Control-Allow-Origin"))
	assert.NotEmpty(t, resp1.Header().Get("Access-Control-Expose-Headers"))
	assert.NotEmpty(t, resp1.Header().Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "https://www.example.io", resp1.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "Authorization,AB-Session", resp1.Header().Get("Access-Control-Expose-Headers"))
	assert.Equal(t, "true", resp1.Header().Get("Access-Control-Allow-Credentials"))
}

func TestFilter_ActualRequest(t *testing.T) {
	cors := CrossOriginResourceSharing{
		AllowedDomains: []string{"https://www.example.io"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type", "Accept", "Device-Id"},
		CookiesAllowed: true,
		MaxAge:         3600,
	}
	filterChain := createDummyFilterChain()

	// TEST 1: Origin allowed
	req1 := createDummyRequest()
	req1.Request.Header.Set("Origin", "https://www.example.io")
	req1.Request.Method = "GET"
	resp1 := createDummyResponse()
	cors.Filter(req1, resp1, filterChain)

	assert.NotEmpty(t, resp1.Header().Get("Access-Control-Allow-Origin"))
	assert.NotEmpty(t, resp1.Header().Get("Access-Control-Allow-Credentials"))
	assert.Empty(t, resp1.Header().Get("Access-Control-Allow-Methods"))
	assert.Empty(t, resp1.Header().Get("Access-Control-Allow-Headers"))
	assert.Empty(t, resp1.Header().Get("Access-Control-Max-Age"))

	assert.Equal(t, "https://www.example.io", resp1.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", resp1.Header().Get("Access-Control-Allow-Credentials"))

	// TEST 2: Origin is not allowed
	req2 := createDummyRequest()
	req2.Request.Header.Set("Origin", "https://wrong.accelbyte.io")
	req2.Request.Method = "GET"
	resp2 := createDummyResponse()
	cors.Filter(req2, resp2, filterChain)

	assert.Empty(t, resp2.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, resp2.Header().Get("Access-Control-Allow-Credentials"))
	assert.Empty(t, resp2.Header().Get("Access-Control-Allow-Methods"))
	assert.Empty(t, resp2.Header().Get("Access-Control-Allow-Headers"))
	assert.Empty(t, resp2.Header().Get("Access-Control-Max-Age"))

	// TEST 3: Origin is not exist in request header
	req3 := createDummyRequest()
	req3.Request.Header.Set("Origin", "")
	req3.Request.Method = "GET"
	resp3 := createDummyResponse()
	cors.Filter(req3, resp3, filterChain)

	assert.Empty(t, resp3.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, resp3.Header().Get("Access-Control-Allow-Credentials"))
	assert.Empty(t, resp3.Header().Get("Access-Control-Allow-Methods"))
	assert.Empty(t, resp3.Header().Get("Access-Control-Allow-Headers"))
	assert.Empty(t, resp3.Header().Get("Access-Control-Max-Age"))
}

func TestFilter_PreflightRequest(t *testing.T) {
	cors := CrossOriginResourceSharing{
		AllowedDomains: []string{"https://www.example.io"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type", "Accept", "Device-Id"},
		CookiesAllowed: true,
		MaxAge:         3600,
	}
	filterChain := createDummyFilterChain()

	// TEST 1: Origin allowed
	req1 := createDummyRequest()
	req1.Request.Header.Set("Origin", "https://www.example.io")
	req1.Request.Header.Set("Access-Control-Request-Method", "GET")
	req1.Request.Header.Set("Access-Control-Request-Headers", "Content-Type,Accept")
	req1.Request.Method = "OPTIONS"
	resp1 := createDummyResponse()
	cors.Filter(req1, resp1, filterChain)

	assert.NotEmpty(t, resp1.Header().Get("Access-Control-Allow-Origin"))
	assert.NotEmpty(t, resp1.Header().Get("Access-Control-Allow-Credentials"))
	assert.NotEmpty(t, resp1.Header().Get("Access-Control-Allow-Methods"))
	assert.NotEmpty(t, resp1.Header().Get("Access-Control-Allow-Headers"))
	assert.NotEmpty(t, resp1.Header().Get("Access-Control-Max-Age"))

	assert.Equal(t, "https://www.example.io", resp1.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", resp1.Header().Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "GET,POST", resp1.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type,Accept", resp1.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "3600", resp1.Header().Get("Access-Control-Max-Age"))

	// TEST 2: Origin not allowed
	req2 := createDummyRequest()
	req2.Request.Header.Set("Origin", "https://wrong.accelbyte.io")
	req2.Request.Header.Set("Access-Control-Request-Method", "GET")
	req2.Request.Header.Set("Access-Control-Request-Headers", "Content-Type,Accept")
	req2.Request.Method = "OPTIONS"
	resp2 := createDummyResponse()
	cors.Filter(req2, resp2, filterChain)

	assert.Empty(t, resp2.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, resp2.Header().Get("Access-Control-Allow-Credentials"))
	assert.Empty(t, resp2.Header().Get("Access-Control-Allow-Methods"))
	assert.Empty(t, resp2.Header().Get("Access-Control-Allow-Headers"))
	assert.Empty(t, resp2.Header().Get("Access-Control-Max-Age"))

	// TEST 3: Origin is not exist in request header
	req3 := createDummyRequest()
	req3.Request.Header.Set("Origin", "")
	req3.Request.Method = "GET"
	resp3 := createDummyResponse()
	cors.Filter(req3, resp3, filterChain)

	assert.Empty(t, resp3.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, resp3.Header().Get("Access-Control-Allow-Credentials"))
	assert.Empty(t, resp3.Header().Get("Access-Control-Allow-Methods"))
	assert.Empty(t, resp3.Header().Get("Access-Control-Allow-Headers"))
	assert.Empty(t, resp3.Header().Get("Access-Control-Max-Age"))

	// TEST 4: Preflight Request but not specified "Access-Control-Request-Method" in its header,
	//         so it will consider as Actual Request
	req4 := createDummyRequest()
	req4.Request.Header.Set("Origin", "https://www.example.io")
	req4.Request.Method = "OPTIONS"
	resp4 := createDummyResponse()
	cors.Filter(req4, resp4, filterChain)

	assert.NotEmpty(t, resp4.Header().Get("Access-Control-Allow-Origin"))
	assert.NotEmpty(t, resp4.Header().Get("Access-Control-Allow-Credentials"))
	assert.Empty(t, resp4.Header().Get("Access-Control-Allow-Methods"))
	assert.Empty(t, resp4.Header().Get("Access-Control-Allow-Headers"))
	assert.Empty(t, resp4.Header().Get("Access-Control-Max-Age"))

	assert.Equal(t, "https://www.example.io", resp4.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", resp4.Header().Get("Access-Control-Allow-Credentials"))
}
