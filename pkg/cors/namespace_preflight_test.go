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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	iam "github.com/AccelByte/iam-go-sdk/v2"
	"github.com/emicklei/go-restful/v3"
)

const (
	gameNamespace      = "accelbytetesting"
	publisherNamespace = "publisher"
	serviceOrigin      = "https://service.example.net"

	// includeParentConfigQueryParameter is the query parameter the config
	// service reads to fold studio and publisher configs into its response.
	includeParentConfigQueryParameter = "includeParentConfig"
)

// namespacedContainer builds a container wired the way a service serving
// namespace-scoped endpoints wires it. This mirrors justice-config-service
// (pkg/configservice/api/api-declaration.go): the dynamic CORS filter is
// installed as a container filter with Container set, no OPTIONSFilter follows
// it, and {namespace} is declared on the WebService root path rather than on
// the route — so route-based resolution has to cope with a parameter that is
// not part of the route's own path.
func namespacedContainer() *restful.Container {
	c, _ := namespacedContainerWithConfigService("", nil)

	return c
}

// namespacedContainerWithConfigService also returns the filter, and points it
// at a config service when configServiceURL is set.
func namespacedContainerWithConfigService(
	configServiceURL string,
	iamClient iam.Client,
) (*restful.Container, *CrossOriginResourceSharing) {
	c := restful.NewContainer()

	// Same argument list justice-config-service passes.
	filter, err := NewCrossOriginResourceSharing(
		configServiceURL,
		iamClient,
		publisherNamespace,
		[]string{serviceOrigin},
		[]string{
			http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete,
			http.MethodOptions,
		},
		[]string{
			"Access-Control-Allow-Origin", "Access-Control-Allow-Methods", "Authorization",
			"Content-Type", "Accept", "X-Amzn-TraceId",
		},
		[]string{},
		true,
		0,
	)
	if err != nil {
		panic("unable to create CORS filter: " + err.Error())
	}
	filter.Container = c
	c.Filter(filter.Filter)

	handler := func(_ *restful.Request, _ *restful.Response) {}
	ws := new(restful.WebService).Path("/config/v1/admin/namespaces/{namespace}")
	ws.Route(ws.GET("/configs").To(handler))
	ws.Route(ws.GET("/configs/{configKey}").To(handler))
	c.Add(ws)

	return c, filter
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

// Namespaces come in several shapes across deployments: a studio namespace is
// often a bare id, and a game namespace under it is that id with a suffix.
// Both must survive resolution intact.
const (
	studioNamespace   = "12345"
	gameUnderStudioNS = "12345-67890"
)

// TestNamespaceFromPath_EndpointShapes covers the namespace-scoped path layouts
// services actually register: with and without a version segment, and with the
// namespace followed by further path segments.
func TestNamespaceFromPath_EndpointShapes(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"/configmigration/admin/namespaces/" + studioNamespace + "/export/rules", studioNamespace},
		{"/cloudsave/v1/namespaces/" + gameUnderStudioNS + "/tags", gameUnderStudioNS},
		{"/config/v1/admin/namespaces/" + gameUnderStudioNS + "/configs/CORS", gameUnderStudioNS},
		{"/iam/v3/public/namespaces/" + studioNamespace + "/users/me", studioNamespace},
	}

	for _, tc := range cases {
		if got := namespaceFromPath(tc.path); got != tc.want {
			t.Errorf("namespaceFromPath(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

// TestExtractNamespace_SubdomainOnCustomBaseDomain covers deployments that
// serve namespaces as subdomains of their own domain rather than an AccelByte
// one, with a path that names no namespace so the subdomain is the only source.
func TestExtractNamespace_SubdomainOnCustomBaseDomain(t *testing.T) {
	const baseDomain = "gamingservices.example.net"

	cases := []struct {
		name          string
		host          string
		configuredFor string
		want          string
	}{
		{"game namespace subdomain", gameUnderStudioNS + "." + baseDomain, baseDomain, gameUnderStudioNS},
		{"studio namespace subdomain", studioNamespace + "." + baseDomain, baseDomain, studioNamespace},
		{"no base domain configured", studioNamespace + "." + baseDomain, "", studioNamespace},
		// The guard that stops third-party hosts from being read as namespaces
		// also silently disables extraction when the configured suffix does not
		// match the deployment's own domain.
		{"configured suffix does not match the host", studioNamespace + "." + baseDomain, "accelbyte.io", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := preflightRequest("/cloudsave/v1/tags", "https://example.com", tc.host)

			got := ExtractNamespaceWithContainer(req, nil, true, tc.configuredFor)
			if got != tc.want {
				t.Errorf("host %q with base domain %q resolved to %q, want %q",
					tc.host, tc.configuredFor, got, tc.want)
			}
		})
	}
}

// recordingConfigClient captures which namespace the filter asked about.
type recordingConfigClient struct {
	asked   []string
	configs map[string]*CORSConfigValue
}

func (r *recordingConfigClient) GetCORSConfig(namespace string) (*CORSConfigValue, error) {
	r.asked = append(r.asked, namespace)

	return r.configs[namespace], nil
}

func (r *recordingConfigClient) GetSubdomainConfig(_ string) (*CORSSubdomainConfig, error) {
	return &CORSSubdomainConfig{SubdomainEnabled: false}, nil
}

// TestFilterPreflight_QueriesNamespaceFromRequest states the contract directly:
// the config lookup for a preflight must use the namespace the request targets,
// falling back to the publisher only when the request names none.
func TestFilterPreflight_QueriesNamespaceFromRequest(t *testing.T) {
	cases := []struct {
		name string
		path string
		want string
	}{
		{"namespace in the path", "/cloudsave/v1/namespaces/" + gameUnderStudioNS + "/tags", gameUnderStudioNS},
		{"namespace in the path, admin route", "/configmigration/admin/namespaces/" + studioNamespace + "/export/rules", studioNamespace},
		{"no namespace in the path", "/cloudsave/v1/tags", "publisher"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := &recordingConfigClient{configs: map[string]*CORSConfigValue{}}
			filter := &CrossOriginResourceSharing{
				AllowedMethods:     []string{http.MethodGet},
				ConfigClient:       client,
				PublisherNamespace: "publisher",
				subdomainConfig:    &CORSSubdomainConfig{SubdomainEnabled: false},
				subdomainLoaded:    true,
			}

			req := preflightRequest(tc.path, "https://example.com", "api.example.net")
			resp := &restful.Response{ResponseWriter: httptest.NewRecorder()}
			chainCalled := false
			filter.Filter(req, resp, createTestFilterChain(&chainCalled))

			if len(client.asked) != 1 {
				t.Fatalf("expected exactly one config lookup, got %v", client.asked)
			}
			if client.asked[0] != tc.want {
				t.Errorf("config lookup used namespace %q, want %q", client.asked[0], tc.want)
			}
		})
	}
}

// recordedRequest is one request the config service stub received.
type recordedRequest struct {
	method string
	path   string
	query  url.Values
	auth   string
}

// stubConfigService serves the two config keys the filter fetches — CORS for a
// namespace and CORS_SUBDOMAIN for the publisher namespace — and records every
// request, so a test can assert the filter calls the real endpoint with the
// real parameters.
func stubConfigService(t *testing.T, domainsByNamespace map[string][]string) (*httptest.Server, *[]recordedRequest) {
	t.Helper()

	var got []recordedRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, recordedRequest{
			method: r.Method,
			path:   r.URL.Path,
			query:  r.URL.Query(),
			auth:   r.Header.Get("Authorization"),
		})

		namespace, key := "", ""
		if parts := splitPath(r.URL.Path); len(parts) == 6 {
			// v1/admin/namespaces/{namespace}/configs/{key}
			namespace, key = parts[3], parts[5]
		}

		if key == "CORS_SUBDOMAIN" {
			writeConfigValue(t, w, namespace, key, CORSSubdomainConfig{SubdomainEnabled: false})

			return
		}

		domains, ok := domainsByNamespace[namespace]
		if !ok {
			w.WriteHeader(http.StatusNotFound)

			return
		}
		writeConfigValue(t, w, namespace, key, CORSConfigValue{AllowedDomains: domains})
	}))
	t.Cleanup(srv.Close)

	return srv, &got
}

func splitPath(p string) []string {
	var parts []string
	for _, s := range strings.Split(p, "/") {
		if s != "" {
			parts = append(parts, s)
		}
	}

	return parts
}

// writeConfigValue emits the config service's response envelope, whose "value"
// field carries the config as a JSON string.
func writeConfigValue(t *testing.T, w http.ResponseWriter, namespace, key string, value interface{}) {
	t.Helper()

	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal config value: %v", err)
	}
	if err := json.NewEncoder(w).Encode(ConfigServiceResponse{
		Namespace: namespace,
		Key:       key,
		Value:     string(raw),
	}); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

// TestPreflightThroughContainer_CallsRealConfigEndpoint drives a preflight
// through the container exactly as an incoming request would, and asserts both
// halves of the round trip: the response the browser sees, and the request the
// filter made to the config service.
func TestPreflightThroughContainer_CallsRealConfigEndpoint(t *testing.T) {
	const namespaceOrigin = "https://example.com"

	srv, recorded := stubConfigService(t, map[string][]string{
		gameNamespace: {namespaceOrigin},
		// The publisher allows a different origin, so a pass can only come from
		// resolving the namespace in the request path.
		publisherNamespace: {"https://publisher.example.org"},
	})

	c, _ := namespacedContainerWithConfigService(srv.URL, iam.NewMockClient())

	req := httptest.NewRequest(http.MethodOptions, "/config/v1/admin/namespaces/"+gameNamespace+"/configs", nil)
	req.Header.Set(restful.HEADER_Origin, namespaceOrigin)
	req.Header.Set(restful.HEADER_AccessControlRequestMethod, http.MethodGet)
	rec := httptest.NewRecorder()
	c.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("preflight status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get(restful.HEADER_AccessControlAllowOrigin); got != namespaceOrigin {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, namespaceOrigin)
	}

	var corsCall *recordedRequest
	for i := range *recorded {
		if (*recorded)[i].query.Get(includeParentConfigQueryParameter) != "" {
			corsCall = &(*recorded)[i]
		}
	}
	if corsCall == nil {
		t.Fatalf("no CORS config request recorded, got %+v", *recorded)
	}

	wantPath := "/v1/admin/namespaces/" + gameNamespace + "/configs/CORS"
	if corsCall.path != wantPath {
		t.Errorf("config service path = %q, want %q", corsCall.path, wantPath)
	}
	if got := corsCall.query.Get(includeParentConfigQueryParameter); got != "studio,publisher" {
		t.Errorf("includeParentConfig = %q, want %q", got, "studio,publisher")
	}
	if corsCall.method != http.MethodGet {
		t.Errorf("config service method = %q, want GET", corsCall.method)
	}
	if corsCall.auth != "Bearer mock_token" {
		t.Errorf("Authorization = %q, want %q", corsCall.auth, "Bearer mock_token")
	}
}

// TestPreflightThroughContainer_RefusedOriginGets405 pins the failure mode the
// report describes: with no OPTIONSFilter installed, a refused preflight runs
// out of filters and go-restful answers 405, since no route serves OPTIONS.
func TestPreflightThroughContainer_RefusedOriginGets405(t *testing.T) {
	srv, _ := stubConfigService(t, map[string][]string{
		gameNamespace: {"https://example.com"},
	})

	c, _ := namespacedContainerWithConfigService(srv.URL, iam.NewMockClient())

	req := httptest.NewRequest(http.MethodOptions, "/config/v1/admin/namespaces/"+gameNamespace+"/configs", nil)
	req.Header.Set(restful.HEADER_Origin, "https://www.google.com")
	req.Header.Set(restful.HEADER_AccessControlRequestMethod, http.MethodGet)
	rec := httptest.NewRecorder()
	c.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("refused preflight status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get(restful.HEADER_AccessControlAllowOrigin); got != "" {
		t.Errorf("refused preflight carries Access-Control-Allow-Origin %q, want none", got)
	}
}

func TestNamespaceFromRouteTemplate(t *testing.T) {
	cases := []struct {
		name      string
		routePath string
		urlPath   string
		want      string
	}{
		{
			"parameter on the WebService root path",
			"/config/v1/admin/namespaces/{namespace}/configs",
			"/config/v1/admin/namespaces/" + gameNamespace + "/configs",
			gameNamespace,
		},
		{
			"parameter carrying a regular expression",
			"/config/v1/admin/namespaces/{namespace:[a-z0-9-]+}/configs",
			"/config/v1/admin/namespaces/" + gameNamespace + "/configs",
			gameNamespace,
		},
		{
			"other parameters before the namespace",
			"/service/{version}/namespaces/{namespace}/things/{id}",
			"/service/v2/namespaces/" + gameNamespace + "/things/7",
			gameNamespace,
		},
		{
			"url shorter than the template",
			"/config/v1/admin/namespaces/{namespace}/configs",
			"/config/v1/admin",
			"",
		},
		{"no namespace parameter", "/config/v1/admin/configs", "/config/v1/admin/configs", ""},
		{"empty template", "", "/config/v1/admin", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := namespaceFromRouteTemplate(tc.routePath, tc.urlPath); got != tc.want {
				t.Errorf("namespaceFromRouteTemplate(%q, %q) = %q, want %q",
					tc.routePath, tc.urlPath, got, tc.want)
			}
		})
	}
}

// TestExtractNamespace_MalformedPreflightPathsDoNotPanic covers resolution
// against the faithful container for request paths a caller can craft freely —
// the filter runs before authentication, on unauthenticated OPTIONS requests.
func TestExtractNamespace_MalformedPreflightPathsDoNotPanic(t *testing.T) {
	container := namespacedContainer()

	paths := []string{
		"/",
		"",
		"/config",
		"/config/v1/admin/namespaces",
		"/config/v1/admin/namespaces/",
		"/config/v1/admin/namespaces//configs",
		"/config/v1/admin/namespaces/" + gameNamespace,
		"/config/v1/admin/namespaces/" + gameNamespace + "/configs/CORS/extra/segments",
		"/config/v1/admin/namespaces/%2e%2e/configs",
		"/../../etc/passwd",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := preflightRequest(path, "https://example.com", "api.example.net")
			// The assertion is that this returns rather than panics; any value
			// is acceptable, since a crafted path has no correct namespace.
			_ = ExtractNamespaceWithContainer(req, container, false, "")
		})
	}
}
