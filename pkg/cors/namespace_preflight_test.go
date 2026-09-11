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
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/emicklei/go-restful/v3"
)

const gameNamespace = "accelbytetesting"

// namespacedContainer builds a container with the route shape of a
// namespace-scoped AGS endpoint, here justice-config-service's
// /config/v1/admin/namespaces/{namespace}/configs.
func namespacedContainer() *restful.Container {
	c := restful.NewContainer()
	ws := new(restful.WebService).Path("/config/v1/admin")
	ws.Route(ws.GET("/namespaces/{namespace}/configs").
		To(func(_ *restful.Request, _ *restful.Response) {}))
	c.Add(ws)

	return c
}

// preflightRequest builds the request curl sends in the AAX-3366 repro:
// an OPTIONS with Origin and Access-Control-Request-Method, and — like every
// real preflight — no path parameters, since no route has been matched.
func preflightRequest(path, origin, host string) *restful.Request {
	return &restful.Request{
		Request: &http.Request{
			Method: http.MethodOptions,
			URL:    &url.URL{Path: path},
			Host:   host,
			Header: http.Header{
				restful.HEADER_Origin:                     []string{origin},
				restful.HEADER_AccessControlRequestMethod: []string{http.MethodGet},
			},
		},
	}
}

func TestExtractNamespace_PreflightResolvesFromRoute(t *testing.T) {
	req := preflightRequest("/config/v1/admin/namespaces/"+gameNamespace+"/configs",
		"https://example.com", "stage.accelbyte.io")

	got := ExtractNamespaceWithContainer(req, namespacedContainer(), false, "")
	if got != gameNamespace {
		t.Errorf("Expected %q resolved from the preflight route, got %q", gameNamespace, got)
	}
}

// TestExtractNamespace_PreflightFallsBackToPathConvention covers the filter
// running without a container, e.g. when container filters have been promoted
// onto WebServices.
func TestExtractNamespace_PreflightFallsBackToPathConvention(t *testing.T) {
	req := preflightRequest("/session/v1/public/namespaces/"+gameNamespace+"/gamesessions",
		"https://example.com", "stage.accelbyte.io")

	got := ExtractNamespaceWithContainer(req, nil, false, "")
	if got != gameNamespace {
		t.Errorf("Expected %q resolved from the path convention, got %q", gameNamespace, got)
	}
}

// TestExtractNamespace_PreflightOutranksSubdomain keeps the preflight decision
// consistent with the actual request, whose path parameter outranks the
// subdomain.
func TestExtractNamespace_PreflightOutranksSubdomain(t *testing.T) {
	req := preflightRequest("/config/v1/admin/namespaces/"+gameNamespace+"/configs",
		"https://example.com", "othernamespace.gamingservices.accelbyte.io")

	got := ExtractNamespaceWithContainer(req, namespacedContainer(), true, "")
	if got != gameNamespace {
		t.Errorf("Expected %q from the preflight URL to outrank the subdomain, got %q", gameNamespace, got)
	}
}

// TestExtractNamespace_UnknownPreflightPathFallsThrough checks a preflight for a
// path that is neither routed nor namespace-scoped still reaches the subdomain
// and header steps.
func TestExtractNamespace_UnknownPreflightPathFallsThrough(t *testing.T) {
	req := preflightRequest("/config/v1/admin/config", "https://example.com", "sub.example.com")
	req.Request.Header.Set(namespaceHeader, "header-namespace")

	if got := ExtractNamespaceWithContainer(req, namespacedContainer(), true, ""); got != "sub" {
		t.Errorf("Expected subdomain %q, got %q", "sub", got)
	}

	if got := ExtractNamespaceWithContainer(req, namespacedContainer(), false, ""); got != "header-namespace" {
		t.Errorf("Expected header namespace, got %q", got)
	}
}

// TestExtractNamespace_NonPreflightNotPathScanned keeps the new step scoped to
// preflights: an ordinary request's namespace comes from the path parameter
// routing already extracted, never from scanning the URL.
func TestExtractNamespace_NonPreflightNotPathScanned(t *testing.T) {
	cases := []struct {
		name    string
		method  string
		headers http.Header
	}{
		{"GET", http.MethodGet, http.Header{}},
		{"OPTIONS without Access-Control-Request-Method", http.MethodOptions, http.Header{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := &restful.Request{
				Request: &http.Request{
					Method: tc.method,
					URL:    &url.URL{Path: "/config/v1/admin/namespaces/" + gameNamespace + "/configs"},
					Host:   "stage.accelbyte.io",
					Header: tc.headers,
				},
			}

			if got := ExtractNamespaceWithContainer(req, namespacedContainer(), false, ""); got != "" {
				t.Errorf("Expected no namespace for a non-preflight request, got %q", got)
			}
		})
	}
}

// TestExtractNamespace_PreflightWithoutURL guards the request shapes used
// elsewhere in these tests, which carry no URL at all.
func TestExtractNamespace_PreflightWithoutURL(t *testing.T) {
	req := &restful.Request{
		Request: &http.Request{
			Method: http.MethodOptions,
			Header: http.Header{
				restful.HEADER_Origin:                     []string{"https://example.com"},
				restful.HEADER_AccessControlRequestMethod: []string{http.MethodGet},
			},
		},
	}

	if got := ExtractNamespaceWithContainer(req, namespacedContainer(), false, ""); got != "" {
		t.Errorf("Expected no namespace when the request has no URL, got %q", got)
	}
}

func TestNamespaceFromPath(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"/config/v1/admin/namespaces/" + gameNamespace + "/configs", gameNamespace},
		{"/session/v1/public/namespaces/ns/gamesessions/1", "ns"},
		{"/namespaces/ns", "ns"},
		{"/config/v1/admin/config", ""},
		{"/config/v1/admin/namespaces", ""},
		{"/config/v1/admin/namespaces/", ""},
		{"", ""},
	}

	for _, tc := range cases {
		if got := namespaceFromPath(tc.path); got != tc.want {
			t.Errorf("namespaceFromPath(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

// TestFilterPreflight_UsesGameNamespaceConfig is the AAX-3366 regression test:
// an origin registered only in the game namespace's CORS config must pass the
// preflight check, which before this fix resolved against the publisher
// namespace (or the service config) and refused.
func TestFilterPreflight_UsesGameNamespaceConfig(t *testing.T) {
	mockClient := NewMockConfigClient()
	mockClient.configs[gameNamespace] = &CORSConfigValue{
		AllowedDomains: []string{"https://example.com"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
	}
	mockClient.configs["publisher"] = &CORSConfigValue{
		AllowedDomains: []string{"https://publisher.example.com"},
		AllowedMethods: []string{"GET"},
	}

	filter := &CrossOriginResourceSharing{
		AllowedDomains:     []string{"https://stage.accelbyte.io"},
		AllowedMethods:     []string{"GET"},
		ConfigClient:       mockClient,
		PublisherNamespace: "publisher",
		Container:          namespacedContainer(),
		subdomainConfig:    &CORSSubdomainConfig{SubdomainEnabled: false},
		subdomainLoaded:    true,
	}

	req := preflightRequest("/config/v1/admin/namespaces/"+gameNamespace+"/configs",
		"https://example.com", "stage.accelbyte.io")
	respWriter := httptest.NewRecorder()
	resp := &restful.Response{ResponseWriter: respWriter}

	chainCalled := false
	filter.Filter(req, resp, createTestFilterChain(&chainCalled))

	if chainCalled {
		t.Error("FilterChain should not be called for an accepted preflight")
	}
	if got := respWriter.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Errorf("Expected the game namespace config to allow the origin, got Access-Control-Allow-Origin %q", got)
	}
	if got := respWriter.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("Access-Control-Allow-Methods should be set for an accepted preflight")
	}
}

// TestFilterPreflight_RefusesOriginMissingFromGameNamespaceConfig is the other
// half: resolving the game namespace must not turn into allowing everything.
func TestFilterPreflight_RefusesOriginMissingFromGameNamespaceConfig(t *testing.T) {
	mockClient := NewMockConfigClient()
	mockClient.configs[gameNamespace] = &CORSConfigValue{
		AllowedDomains: []string{"https://example.com"},
		AllowedMethods: []string{"GET"},
	}

	filter := &CrossOriginResourceSharing{
		AllowedDomains:     []string{"https://stage.accelbyte.io"},
		AllowedMethods:     []string{"GET"},
		ConfigClient:       mockClient,
		PublisherNamespace: "publisher",
		Container:          namespacedContainer(),
		subdomainConfig:    &CORSSubdomainConfig{SubdomainEnabled: false},
		subdomainLoaded:    true,
	}

	req := preflightRequest("/config/v1/admin/namespaces/"+gameNamespace+"/configs",
		"https://www.google.com", "stage.accelbyte.io")
	respWriter := httptest.NewRecorder()
	resp := &restful.Response{ResponseWriter: respWriter}

	chainCalled := false
	filter.Filter(req, resp, createTestFilterChain(&chainCalled))

	if !chainCalled {
		t.Error("FilterChain should be called when the preflight origin is refused")
	}
	if got := respWriter.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Refused preflight should carry no Access-Control-Allow-Origin, got %q", got)
	}
}
