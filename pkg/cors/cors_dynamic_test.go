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
	"testing"
	"time"

	iam "github.com/AccelByte/iam-go-sdk/v2"
	"github.com/emicklei/go-restful/v3"
)

// MockConfigClient is a test implementation of ConfigClient
type MockConfigClient struct {
	configs         map[string]*CORSConfigValue
	errors          map[string]error
	subdomainConfig *CORSSubdomainConfig
}

func (m *MockConfigClient) GetCORSConfig(namespace string) (*CORSConfigValue, error) {
	if err, ok := m.errors[namespace]; ok {
		return nil, err
	}
	return m.configs[namespace], nil
}

func (m *MockConfigClient) GetSubdomainConfig(_ string) (*CORSSubdomainConfig, error) {
	return m.subdomainConfig, nil
}

func NewMockConfigClient() *MockConfigClient {
	return &MockConfigClient{
		configs: make(map[string]*CORSConfigValue),
		errors:  make(map[string]error),
	}
}

func TestStaticCORSFilter(t *testing.T) {
	// Test with just static config (no ConfigClient)
	filter := &CrossOriginResourceSharing{
		AllowedDomains: []string{"https://example.com"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
		ExposeHeaders:  []string{"X-Custom"},
		CookiesAllowed: true,
		MaxAge:         3600,
		ConfigClient:   nil, // No dynamic config
	}

	if filter == nil {
		t.Fatal("Filter is nil")
	}
	if filter.ConfigClient != nil {
		t.Error("ConfigClient should be nil for static filter")
	}
	if len(filter.AllowedDomains) != 1 {
		t.Errorf("AllowedDomains mismatch: %v", filter.AllowedDomains)
	}
}

func TestDynamicCORSFilter(t *testing.T) {
	mockClient := NewMockConfigClient()

	// Test with ConfigClient enabled
	filter := &CrossOriginResourceSharing{
		AllowedDomains:      []string{"https://service.com"},
		AllowedMethods:      []string{"GET"},
		AllowedHeaders:      []string{"Content-Type"},
		ExposeHeaders:       []string{"X-Service"},
		CookiesAllowed:      false,
		MaxAge:              3600,
		ConfigClient:       mockClient, // Dynamic config enabled
		ConfigFetchTimeout: 200 * time.Millisecond,
	}

	if filter == nil {
		t.Fatal("Filter is nil")
	}
	if filter.ConfigClient == nil {
		t.Error("ConfigClient should not be nil for dynamic filter")
	}
}

func TestGetStaticConfig(t *testing.T) {
	filter := &CrossOriginResourceSharing{
		AllowedDomains: []string{"https://example.com"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
		ExposeHeaders:  []string{"X-Custom"},
		CookiesAllowed: true,
		MaxAge:         3600,
	}

	config := filter.getStaticConfig()

	if len(config.AllowedDomains) != 1 || config.AllowedDomains[0] != "https://example.com" {
		t.Errorf("AllowedDomains mismatch: %v", config.AllowedDomains)
	}
	if !config.CookiesAllowed {
		t.Error("CookiesAllowed should be true")
	}
	if config.MaxAge != 3600 {
		t.Errorf("MaxAge should be 3600, got %d", config.MaxAge)
	}
}

func TestIsOriginAllowedWithConfig(t *testing.T) {
	filter := &CrossOriginResourceSharing{}

	tests := []struct {
		name     string
		config   *MergedCORSConfig
		origin   string
		expected bool
	}{
		// exact match
		{"exact match allowed", &MergedCORSConfig{AllowedDomains: []string{"https://example.com"}}, "https://example.com", true},
		{"exact match denied", &MergedCORSConfig{AllowedDomains: []string{"https://example.com"}}, "https://other.com", false},
		// allow-all
		{"allow-all wildcard", &MergedCORSConfig{AllowedDomains: []string{"*"}}, "https://any.com", true},
		// empty list
		{"empty domains allows all", &MergedCORSConfig{AllowedDomains: []string{}}, "https://any.com", true},
		// wildcard pattern
		{"wildcard matches subdomain", &MergedCORSConfig{AllowedDomains: []string{"https://*.accelbyte.io"}}, "https://game-ns.accelbyte.io", true},
		{"wildcard rejects base domain", &MergedCORSConfig{AllowedDomains: []string{"https://*.accelbyte.io"}}, "https://accelbyte.io", false},
		{"wildcard rejects two-level subdomain", &MergedCORSConfig{AllowedDomains: []string{"https://*.accelbyte.io"}}, "https://a.b.accelbyte.io", false},
		{"wildcard rejects wrong domain", &MergedCORSConfig{AllowedDomains: []string{"https://*.accelbyte.io"}}, "https://game-ns.other.io", false},
		// regex pattern
		{"regex matches", &MergedCORSConfig{AllowedDomains: []string{"re:^https://.*\\.example\\.com$"}}, "https://sub.example.com", true},
		{"regex rejects non-match", &MergedCORSConfig{AllowedDomains: []string{"re:^https://.*\\.example\\.com$"}}, "https://sub.example.io", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.isOriginAllowedWithConfig(tt.config, tt.origin)
			if result != tt.expected {
				t.Errorf("isOriginAllowedWithConfig domains:%v origin:%s = %v, expected %v",
					tt.config.AllowedDomains, tt.origin, result, tt.expected)
			}
		})
	}
}

func TestFilterWithoutOrigin(t *testing.T) {
	filter := &CrossOriginResourceSharing{
		AllowedDomains: []string{"https://example.com"},
		AllowedMethods: []string{"GET"},
	}

	req := &restful.Request{
		Request: &http.Request{
			Method: "GET",
			Header: http.Header{},
		},
	}

	chainCalled := false
	chain := createTestFilterChain(&chainCalled)

	resp := &restful.Response{
		ResponseWriter: httptest.NewRecorder(),
	}

	filter.Filter(req, resp, chain)

	if !chainCalled {
		t.Error("FilterChain should be called when no Origin header present")
	}
}

// Test utilities

func createTestFilterChain(called *bool) *restful.FilterChain {
	return &restful.FilterChain{
		Filters: make([]restful.FilterFunction, 0),
		Target: func(req *restful.Request, resp *restful.Response) {
			*called = true
		},
	}
}

func TestFilterWithDynamicConfig(t *testing.T) {
	mockClient := NewMockConfigClient()
	mockClient.configs["game-ns"] = &CORSConfigValue{
		AllowedDomains: []string{"https://game.example.com"},
		AllowedMethods: []string{"GET", "POST"},
	}

	filter := &CrossOriginResourceSharing{
		AllowedDomains:     []string{"https://service.com"},
		AllowedMethods:     []string{"GET"},
		AllowedHeaders:     []string{"Content-Type"},
		ConfigClient:       mockClient,
		ConfigFetchTimeout: 200 * time.Millisecond,
		subdomainConfig:    &CORSSubdomainConfig{SubdomainEnabled: true},
		subdomainLoaded:    true,
	}

	req := &restful.Request{
		Request: &http.Request{
			Method: "GET",
			Header: http.Header{
				"Origin": []string{"https://game.example.com"},
			},
			Host: "game-ns.example.com",
		},
	}

	chainCalled := false
	chain := createTestFilterChain(&chainCalled)
	resp := &restful.Response{
		ResponseWriter: httptest.NewRecorder(),
	}

	filter.Filter(req, resp, chain)

	if !chainCalled {
		t.Error("FilterChain should be called for allowed origin")
	}
}

func TestFilterWithDynamicWildcardConfig(t *testing.T) {
	mockClient := NewMockConfigClient()
	mockClient.configs["game-ns"] = &CORSConfigValue{
		AllowedDomains: []string{"https://*.game.example.com"},
		AllowedMethods: []string{"GET", "POST"},
	}

	filter := &CrossOriginResourceSharing{
		AllowedDomains:     []string{"https://service.com"},
		AllowedMethods:     []string{"GET"},
		AllowedHeaders:     []string{"Content-Type"},
		ConfigClient:       mockClient,
		ConfigFetchTimeout: 200 * time.Millisecond,
		subdomainConfig:    &CORSSubdomainConfig{SubdomainEnabled: true},
		subdomainLoaded:    true,
	}

	makeReq := func(origin, host string) (*restful.Request, *restful.Response) {
		req := &restful.Request{
			Request: &http.Request{
				Method: "GET",
				Header: http.Header{"Origin": []string{origin}},
				Host:   host,
			},
		}
		resp := &restful.Response{ResponseWriter: httptest.NewRecorder()}
		return req, resp
	}
	dummy := false
	chain := createTestFilterChain(&dummy)

	// Wildcard subdomain from namespace config — Access-Control-Allow-Origin should be set
	req1, resp1 := makeReq("https://client.game.example.com", "game-ns.example.com")
	filter.Filter(req1, resp1, chain)
	if resp1.Header().Get("Access-Control-Allow-Origin") != "https://client.game.example.com" {
		t.Errorf("Expected CORS allow header for wildcard match, got %q", resp1.Header().Get("Access-Control-Allow-Origin"))
	}

	// Origin in service config — Access-Control-Allow-Origin should be set
	req2, resp2 := makeReq("https://service.com", "game-ns.example.com")
	filter.Filter(req2, resp2, chain)
	if resp2.Header().Get("Access-Control-Allow-Origin") != "https://service.com" {
		t.Errorf("Expected CORS allow header for service config match, got %q", resp2.Header().Get("Access-Control-Allow-Origin"))
	}

	// Origin not in any config — Access-Control-Allow-Origin must not be set
	req3, resp3 := makeReq("https://unknown.com", "game-ns.example.com")
	filter.Filter(req3, resp3, chain)
	if resp3.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("Expected no CORS allow header for unknown origin, got %q", resp3.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestFilterPreflightRequest(t *testing.T) {
	filter := &CrossOriginResourceSharing{
		AllowedDomains: []string{"https://example.com"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
		MaxAge:         3600,
	}

	req := &restful.Request{
		Request: &http.Request{
			Method: "OPTIONS",
			Header: http.Header{
				"Origin":                        []string{"https://example.com"},
				"Access-Control-Request-Method": []string{"POST"},
				"Access-Control-Request-Headers": []string{"Content-Type"},
			},
		},
	}

	respWriter := httptest.NewRecorder()
	resp := &restful.Response{
		ResponseWriter: respWriter,
	}

	chainCalled := false
	chain := createTestFilterChain(&chainCalled)

	filter.Filter(req, resp, chain)

	// Preflight should not call chain
	if chainCalled {
		t.Error("FilterChain should not be called for preflight")
	}

	// Check that preflight headers were set
	if _, ok := respWriter.Header()["Access-Control-Allow-Origin"]; !ok {
		t.Error("Access-Control-Allow-Origin header should be set")
	}
	if _, ok := respWriter.Header()["Access-Control-Allow-Methods"]; !ok {
		t.Error("Access-Control-Allow-Methods header should be set")
	}
}

func TestInitConfigServiceClient(t *testing.T) {
	// ConfigServiceURL empty — should leave ConfigClient nil
	filter := &CrossOriginResourceSharing{
		AllowedDomains: []string{"https://example.com"},
		AllowedMethods: []string{"GET"},
		// ConfigServiceURL not set
	}

	filter.initConfigServiceClient()
	if filter.ConfigClient != nil {
		t.Error("ConfigClient should be nil when ConfigServiceURL is empty")
	}
}

func TestInitConfigServiceClientWithURL(t *testing.T) {
	iamClient := iam.NewMockClient()
	filter := &CrossOriginResourceSharing{
		AllowedDomains:   []string{"https://example.com"},
		AllowedMethods:   []string{"GET"},
		ConfigServiceURL: "http://test-config-service/config",
		IAMClient:        iamClient,
	}

	filter.initConfigServiceClient()

	if filter.ConfigClient == nil {
		t.Error("ConfigClient should be initialized when ConfigServiceURL and IAMClient are set")
	}
}

func TestInitConfigServiceClientMissingIAM(t *testing.T) {
	// ConfigServiceURL set but IAMClient nil — ConfigClient must stay nil (falls back to static)
	filter := &CrossOriginResourceSharing{
		AllowedDomains:   []string{"https://example.com"},
		AllowedMethods:   []string{"GET"},
		ConfigServiceURL: "http://test-config-service/config",
		// IAMClient intentionally omitted
	}

	filter.initConfigServiceClient()

	if filter.ConfigClient != nil {
		t.Error("ConfigClient should remain nil when IAMClient is not set")
	}
}

func TestFilterAutoInitConfigClient(t *testing.T) {
	// ConfigClient is auto-initialized on the first request when ConfigServiceURL and IAMClient are set
	iamClient := iam.NewMockClient()
	filter := &CrossOriginResourceSharing{
		AllowedDomains:   []string{"https://example.com"},
		AllowedMethods:   []string{"GET"},
		ConfigServiceURL: "http://test-config-service/config",
		IAMClient:        iamClient,
	}

	if filter.ConfigClient != nil {
		t.Error("ConfigClient should be nil initially")
	}

	req := &restful.Request{
		Request: &http.Request{
			Method: "GET",
			Header: http.Header{
				"Origin": []string{"https://example.com"},
			},
		},
	}

	chainCalled := false
	chain := createTestFilterChain(&chainCalled)
	resp := &restful.Response{
		ResponseWriter: httptest.NewRecorder(),
	}

	filter.Filter(req, resp, chain)

	if filter.ConfigClient == nil {
		t.Error("ConfigClient should be initialized after first request")
	}
}

func TestNewCrossOriginResourceSharingWithIAM(t *testing.T) {
	iamClient := iam.NewMockClient()

	filter, err := NewCrossOriginResourceSharing(
		"http://config-service/config",
		iamClient,
		"",
		[]string{"https://example.com"},
		[]string{"GET", "POST"},
		[]string{"Content-Type"},
		[]string{"X-Custom"},
		true,
		3600,
	)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if filter == nil {
		t.Fatal("Filter should not be nil")
	}
	if filter.IAMClient != iamClient {
		t.Error("IAMClient should be set to the provided client")
	}
	if filter.ConfigServiceURL != "http://config-service/config" {
		t.Error("ConfigServiceURL should be set from constructor")
	}
	if len(filter.AllowedDomains) != 1 || filter.AllowedDomains[0] != "https://example.com" {
		t.Error("AllowedDomains should be set from constructor")
	}
	if filter.MaxAge != 3600 {
		t.Error("MaxAge should be set from constructor")
	}
}

func TestNewCrossOriginResourceSharingWithoutIAM(t *testing.T) {
	filter, err := NewCrossOriginResourceSharing(
		"",
		nil,
		"",
		[]string{"https://example.com"},
		[]string{"GET"},
		[]string{"Content-Type"},
		[]string{},
		false,
		3600,
	)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if filter == nil {
		t.Fatal("Filter should not be nil")
	}
	if filter.IAMClient != nil {
		t.Error("IAMClient should be nil when not provided")
	}
	if filter.ConfigServiceURL != "" {
		t.Error("ConfigServiceURL should be empty when not provided")
	}
}

func TestNewCrossOriginResourceSharingMissingIAMClient(t *testing.T) {
	// configServiceURL set but iamClient nil — must return error
	filter, err := NewCrossOriginResourceSharing(
		"http://config-service/config",
		nil,
		"",
		[]string{"https://example.com"},
		[]string{"GET"},
		[]string{"Content-Type"},
		[]string{},
		false,
		3600,
	)

	if err == nil {
		t.Error("Expected error when configServiceURL is set but iamClient is nil")
	}
	if filter != nil {
		t.Error("Filter should be nil on error")
	}
}

func TestFilterInitWithIAMClient(t *testing.T) {
	iamClient := iam.NewMockClient()

	filter, err := NewCrossOriginResourceSharing(
		"http://test-config-service/config",
		iamClient,
		"",
		[]string{"https://example.com"},
		[]string{"GET"},
		[]string{"Content-Type"},
		[]string{},
		false,
		3600,
	)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	req := &restful.Request{
		Request: &http.Request{
			Method: "GET",
			Header: http.Header{
				"Origin": []string{"https://example.com"},
			},
		},
	}

	chainCalled := false
	chain := createTestFilterChain(&chainCalled)
	resp := &restful.Response{
		ResponseWriter: httptest.NewRecorder(),
	}

	filter.Filter(req, resp, chain)

	if filter.IAMClient != iamClient {
		t.Error("IAMClient should be preserved after initialization")
	}
	if filter.ConfigClient == nil {
		t.Error("ConfigClient should be initialized")
	}
}
