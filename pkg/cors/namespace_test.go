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
	"testing"

	"github.com/emicklei/go-restful/v3"
)

func TestExtractNamespace_FromPathParameter(t *testing.T) {
	// Note: Path parameter extraction via restful.Request is tested implicitly
	// in the integration tests. Unit testing it directly is difficult due to
	// the tightly-coupled nature of restful.Request and its route binding.
	// This test documents the expected behavior.
	t.Skip("Path parameter extraction tested via integration tests")
}

func TestExtractNamespace_FromSubdomain(t *testing.T) {
	req := &restful.Request{
		Request: &http.Request{
			Header: http.Header{},
			Host:   "my-namespace.example.com",
		},
	}

	result := ExtractNamespace(req, true, "")
	if result != "my-namespace" {
		t.Errorf("Expected 'my-namespace', got %q", result)
	}
}

func TestExtractNamespace_SubdomainIgnoredWhenDisabled(t *testing.T) {
	req := &restful.Request{
		Request: &http.Request{
			Header: http.Header{},
			Host:   "my-namespace.example.com",
		},
	}
	req.Request.Header.Set("x-ab-rl-ns", "header-namespace")

	result := ExtractNamespace(req, false, "")
	if result != "header-namespace" {
		t.Errorf("Expected header-namespace (subdomain skipped when disabled), got %q", result)
	}
}

func TestExtractNamespace_FromHeader(t *testing.T) {
	req := &restful.Request{
		Request: &http.Request{
			Header: http.Header{},
			Host:   "example.com",
		},
	}
	req.Request.Header.Set("x-ab-rl-ns", "header-namespace")

	result := ExtractNamespace(req, false, "")
	if result != "header-namespace" {
		t.Errorf("Expected 'header-namespace', got %q", result)
	}
}

func TestExtractNamespace_Priority(t *testing.T) {
	httpReq := &http.Request{
		Header: http.Header{},
		Host:   "subdomain.example.com",
	}
	httpReq.Header.Set("x-ab-rl-ns", "header-namespace")

	req := restful.NewRequest(httpReq)

	result := ExtractNamespace(req, true, "")
	if result != "subdomain" {
		t.Errorf("Subdomain should have priority over header, got %q", result)
	}
}

func TestExtractNamespace_NotFound(t *testing.T) {
	req := &restful.Request{
		Request: &http.Request{
			Header: http.Header{},
			Host:   "example.com",
		},
	}

	result := ExtractNamespace(req, false, "")
	if result != "" {
		t.Errorf("Expected empty string, got %q", result)
	}
}

func TestExtractSubdomain(t *testing.T) {
	tests := []struct {
		host       string
		baseDomain string
		expected   string
	}{
		{"api.example.com", "", "api"},
		{"admin.example.com", "", "admin"},
		{"my-namespace.example.io", "", "my-namespace"},
		{"a.b.c.io", "", "a"},
		{"example.com", "", ""},    // 2 parts: no subdomain
		{"localhost", "", ""},      // 1 part
		{"localhost:8080", "", ""}, // 1 part (port removed)
		{"api.example.com:8080", "", "api"},
		// baseDomain filtering
		{"my-ns.myservice.accelbyte.io", "accelbyte.io", "my-ns"},   // suffix matches → extract
		{"gamingservices.xsolla.com", "accelbyte.io", ""},            // suffix mismatch → skip
		{"api.gamingservices.xsolla.com", "accelbyte.io", ""},       // 4-part third-party → skip
		{"my-ns.example.com", "example.com", "my-ns"},               // 3-part with matching suffix
		{"accelbyte.io", "accelbyte.io", ""},                         // base domain itself → no subdomain
	}

	for _, tt := range tests {
		req := &restful.Request{
			Request: &http.Request{
				Header: http.Header{},
				Host:   tt.host,
			},
		}
		result := extractSubdomain(req, tt.baseDomain)
		if result != tt.expected {
			t.Errorf("extractSubdomain(%q, %q) = %q, expected %q", tt.host, tt.baseDomain, result, tt.expected)
		}
	}
}

func TestExtractSubdomain_WithPort(t *testing.T) {
	req := &restful.Request{
		Request: &http.Request{
			Header: http.Header{},
			Host:   "api.example.com:9000",
		},
	}

	result := extractSubdomain(req, "")
	if result != "api" {
		t.Errorf("Expected 'api', got %q", result)
	}
}
