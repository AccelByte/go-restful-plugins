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
	"testing"
	"time"

	iam "github.com/AccelByte/iam-go-sdk/v2"
)

func TestConfigClientGetCORSConfig(t *testing.T) {
	// Create a mock config service
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Expected URL format: /v1/admin/namespaces/{namespace}/configs/CORS
		if r.URL.Path != "/v1/admin/namespaces/test-ns/configs/CORS" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("includeParentConfig") != "studio,publisher" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Return the nested JSON response as per the tech spec
		response := ConfigServiceResponse{
			Namespace: "test-ns",
			Key:       "CORS",
			// Value is a JSON string containing the CORS config
			Value: `{
				"expose_headers": ["X-Custom"],
				"allowed_headers": ["Content-Type"],
				"allowed_domains": ["https://example.com", "https://*.accelbyte.io"],
				"allowed_methods": ["GET", "POST"],
				"cookies_allowed": true,
				"max_age": 3600
			}`,
			CreatedAt: "2026-02-09T07:29:24.357Z",
			UpdatedAt: "2026-02-09T07:29:24.357Z",
			IsPublic:  false,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewConfigClientWithIAM(server.URL, 1*time.Hour, nil, TransportConfig{})

	// First call should hit the server
	config, err := client.GetCORSConfig("test-ns")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if config == nil {
		t.Fatal("Expected config, got nil")
	}

	if len(config.AllowedDomains) != 2 {
		t.Errorf("Expected 2 allowed domains, got %d", len(config.AllowedDomains))
	}

	if config.AllowedDomains[0] != "https://example.com" {
		t.Errorf("Expected domain 'https://example.com', got %q", config.AllowedDomains[0])
	}

	if !config.CookiesAllowed {
		t.Error("CookiesAllowed should be true")
	}

	if config.MaxAge != 3600 {
		t.Errorf("MaxAge should be 3600, got %d", config.MaxAge)
	}

	// Second call should use cache
	config2, err := client.GetCORSConfig("test-ns")
	if err != nil {
		t.Fatalf("Unexpected error on cached call: %v", err)
	}

	if config2 == nil {
		t.Fatal("Expected cached config, got nil")
	}

	if config.AllowedDomains[0] != config2.AllowedDomains[0] {
		t.Error("Cached config should match original config")
	}
}

func TestConfigClientNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewConfigClientWithIAM(server.URL, 1*time.Hour, nil, TransportConfig{})

	// Not found should return nil, not error (fallback to service config)
	config, err := client.GetCORSConfig("nonexistent-ns")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if config != nil {
		t.Error("Expected nil config for not found, got non-nil")
	}
}

func TestConfigClientServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewConfigClientWithIAM(server.URL, 1*time.Hour, nil, TransportConfig{})

	// Server error should return error
	_, err := client.GetCORSConfig("test-ns")
	if err == nil {
		t.Fatal("Expected error for server error")
	}
}

func TestConfigClientInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return invalid JSON
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewConfigClientWithIAM(server.URL, 1*time.Hour, nil, TransportConfig{})

	_, err := client.GetCORSConfig("test-ns")
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}
}

func TestConfigClientInvalidValueJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := ConfigServiceResponse{
			Namespace: "test-ns",
			Key:       "CORS",
			// Invalid JSON in value field
			Value: "not valid json",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewConfigClientWithIAM(server.URL, 1*time.Hour, nil, TransportConfig{})

	_, err := client.GetCORSConfig("test-ns")
	if err == nil {
		t.Fatal("Expected error for invalid value JSON")
	}
}

func TestConfigClientEmptyValue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := ConfigServiceResponse{
			Namespace: "test-ns",
			Key:       "CORS",
			Value:     "", // Empty value
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewConfigClientWithIAM(server.URL, 1*time.Hour, nil, TransportConfig{})

	// Empty value should return nil config (fallback to service config)
	config, err := client.GetCORSConfig("test-ns")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if config != nil {
		t.Error("Expected nil config for empty value")
	}
}

func TestCORSConfigValueJsonTags(t *testing.T) {
	// Test that JSON tags are correctly set on CORSConfigValue
	jsonStr := `{
		"expose_headers": ["X-Header"],
		"allowed_headers": ["Content-Type"],
		"allowed_domains": ["https://example.com"],
		"allowed_methods": ["GET"],
		"cookies_allowed": true,
		"max_age": 7200
	}`

	var config CORSConfigValue
	err := json.Unmarshal([]byte(jsonStr), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if config.MaxAge != 7200 {
		t.Errorf("Expected MaxAge 7200, got %d", config.MaxAge)
	}

	if !config.CookiesAllowed {
		t.Error("CookiesAllowed should be true")
	}

	if len(config.ExposeHeaders) != 1 || config.ExposeHeaders[0] != "X-Header" {
		t.Errorf("ExposeHeaders mismatch: %v", config.ExposeHeaders)
	}
}

// TestIAMClientTokenAuthentication tests that the client includes bearer token in requests
func TestIAMClientTokenAuthentication(t *testing.T) {
	var authHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")

		response := ConfigServiceResponse{
			Namespace: "test-ns",
			Key:       "CORS",
			Value: `{
				"allowed_domains": ["https://example.com"],
				"allowed_methods": ["GET"]
			}`,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// iam.MockClient.ClientToken() returns "mock_token"
	mockIAM := iam.NewMockClient()
	client := NewConfigClientWithIAM(server.URL, 1*time.Minute, mockIAM, TransportConfig{})

	_, err := client.GetCORSConfig("test-ns")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedAuth := "Bearer mock_token"
	if authHeader != expectedAuth {
		t.Errorf("Expected Authorization header '%s', got '%s'", expectedAuth, authHeader)
	}
}
