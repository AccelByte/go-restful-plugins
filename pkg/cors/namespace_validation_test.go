// Copyright 2026 AccelByte Inc
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
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// invalidNamespaces are values a request can carry that aren't AGS namespaces.
var invalidNamespaces = []string{
	"",
	"-", // what the rate limit service sets in x-ab-rl-ns when it found no namespace
	"a?x=1",
	"a#b",
	"a/../b",
	"../admin",
	"a b",
	"a_b",
	"a-b-c",
	"-a",
	"a-",
	strings.Repeat("a", 257),
}

func TestIsValidNamespace(t *testing.T) {
	t.Parallel()

	for _, ns := range []string{"game1", "accelbyte", "studio-game", "ABC123", strings.Repeat("a", 256)} {
		assert.True(t, isValidNamespace(ns), "%q should be valid", ns)
	}
	for _, ns := range invalidNamespaces {
		assert.False(t, isValidNamespace(ns), "%q should be invalid", ns)
	}
}

// An invalid value from one source is ignored, and the next source is used.
func TestExtractNamespace_IgnoresInvalidValues(t *testing.T) {
	t.Parallel()

	newRequest := func(host, header string) *restful.Request {
		httpReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
		httpReq.Host = host
		if header != "" {
			httpReq.Header.Set(namespaceHeader, header)
		}

		return restful.NewRequest(httpReq)
	}

	for _, ns := range invalidNamespaces {
		assert.Empty(t, ExtractNamespace(newRequest("example.com", ns), false, ""), "header %q", ns)
	}

	assert.Equal(t, "game1", ExtractNamespace(newRequest("a_b.example.io", "game1"), true, ""),
		"an invalid subdomain falls through to the header")
	assert.Equal(t, "studio-game", ExtractNamespace(newRequest("studio-game.example.io", "game1"), true, ""),
		"a valid subdomain still ranks above the header")
}

// A path parameter that isn't a namespace, e.g. an encoded "?" or "#", falls through to the next source.
func TestExtractNamespace_IgnoresInvalidPathParameter(t *testing.T) {
	t.Parallel()

	var got string

	ws := new(restful.WebService)
	ws.Route(ws.GET("/v1/public/namespaces/{namespace}/things").To(func(req *restful.Request, resp *restful.Response) {
		got = ExtractNamespace(req, false, "")
		resp.WriteHeader(http.StatusOK)
	}))
	container := restful.NewContainer()
	container.Add(ws)

	tests := map[string]string{
		"/v1/public/namespaces/game1/things":   "game1",
		"/v1/public/namespaces/a%3Fx=1/things": "header-ns",
		"/v1/public/namespaces/a%23b/things":   "header-ns",
		"/v1/public/namespaces/a-b-c/things":   "header-ns",
	}
	for path, want := range tests {
		got = ""
		httpReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
		httpReq.Header.Set(namespaceHeader, "header-ns")
		container.ServeHTTP(httptest.NewRecorder(), httpReq)

		assert.Equal(t, want, got, "path %s", path)
	}
}

// The namespace read from a preflight's URL is validated too.
func TestExtractNamespace_IgnoresInvalidPreflightNamespace(t *testing.T) {
	t.Parallel()

	newPreflight := func(path string) *restful.Request {
		httpReq := httptest.NewRequestWithContext(context.Background(), http.MethodOptions, path, nil)
		httpReq.Header.Set(restful.HEADER_AccessControlRequestMethod, http.MethodGet)
		httpReq.Header.Set(namespaceHeader, "header-ns")

		return restful.NewRequest(httpReq)
	}

	assert.Equal(t, "game1", ExtractNamespaceWithContainer(newPreflight("/v1/public/namespaces/game1/things"), nil, false, ""))
	assert.Equal(t, "header-ns",
		ExtractNamespaceWithContainer(newPreflight("/v1/public/namespaces/a%3Fx=1/things"), nil, false, ""))
}

// The "-" the rate limit service sets when it found no namespace now falls back to the publisher namespace.
func TestGetConfigWithDynamicResolution_PlaceholderHeaderFallsBackToPublisherNamespace(t *testing.T) {
	t.Parallel()

	configClient := &recordingConfigClient{configs: map[string]*CORSConfigValue{}}
	configClient.configs["publisher"] = &CORSConfigValue{
		AllowedDomains: []string{"https://publisher.example.com"},
		AllowedMethods: []string{"GET", "POST"},
	}

	filter := &CrossOriginResourceSharing{
		AllowedDomains:     []string{"https://service.com"},
		AllowedMethods:     []string{"GET"},
		ConfigClient:       configClient,
		PublisherNamespace: "publisher",
		subdomainConfig:    &CORSSubdomainConfig{SubdomainEnabled: false},
		subdomainLoaded:    true,
	}

	httpReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	httpReq.Header.Set("Origin", "https://publisher.example.com")
	httpReq.Header.Set(namespaceHeader, "-")

	chainCalled := false
	resp := &restful.Response{ResponseWriter: httptest.NewRecorder()}
	filter.Filter(restful.NewRequest(httpReq), resp, createTestFilterChain(&chainCalled))

	assert.True(t, chainCalled, "the origin is allowed by the publisher namespace config")
	assert.Equal(t, "https://publisher.example.com", resp.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, []string{"publisher"}, configClient.asked, `"-" is never looked up`)
}

// The config client escapes the namespace, so it can't change which config service URL is called.
func TestConfigClient_EscapesNamespace(t *testing.T) {
	t.Parallel()

	type call struct{ path, query string }

	var (
		mu    sync.Mutex
		calls []call
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls = append(calls, call{path: r.URL.Path, query: r.URL.RawQuery})
		mu.Unlock()
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	client := NewConfigClientWithIAM(server.URL, time.Minute, nil, TransportConfig{})

	_, err := client.GetCORSConfig("game1")
	require.NoError(t, err)
	_, err = client.GetCORSConfig("a?x=1")
	require.NoError(t, err)
	_, err = client.GetSubdomainConfig("a#b")
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []call{
		{path: "/v1/admin/namespaces/game1/configs/CORS", query: "includeParentConfig=studio,publisher"},
		{path: "/v1/admin/namespaces/a?x=1/configs/CORS", query: "includeParentConfig=studio,publisher"},
		{path: "/v1/admin/namespaces/a#b/configs/CORS_SUBDOMAIN", query: ""},
	}, calls)
}

// A Host given as an IP address has no subdomain, so the header is used.
func TestExtractNamespace_IPHostHasNoSubdomain(t *testing.T) {
	t.Parallel()

	for _, host := range []string{"10.0.3.7", "10.0.3.7:8080", "[::1]", "[::1]:8080"} {
		httpReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
		httpReq.Host = host
		httpReq.Header.Set(namespaceHeader, "header-ns")

		assert.Equal(t, "header-ns", ExtractNamespace(restful.NewRequest(httpReq), true, ""), "host %s", host)
	}
}

// When the route's {namespace} position holds an invalid value, the /namespaces/{namespace}/ segment is still tried.
func TestExtractNamespace_PreflightRouteValueInvalidFallsBackToPath(t *testing.T) {
	t.Parallel()

	ws := new(restful.WebService)
	ws.Route(ws.GET("/v1/{namespace}/namespaces/{other}/things").To(func(_ *restful.Request, _ *restful.Response) {}))
	container := restful.NewContainer()
	container.Add(ws)

	httpReq := httptest.NewRequestWithContext(context.Background(), http.MethodOptions, "/v1/a_b/namespaces/game1/things", nil)
	httpReq.Header.Set(restful.HEADER_AccessControlRequestMethod, http.MethodGet)

	assert.Equal(t, "game1", ExtractNamespaceWithContainer(restful.NewRequest(httpReq), container, false, ""))
}
