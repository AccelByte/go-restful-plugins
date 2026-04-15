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
	"testing"
)

func TestCompileExactMatch(t *testing.T) {
	pm, err := Compile("https://example.com")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if pm.Type != PatternTypeExact {
		t.Errorf("Expected PatternTypeExact, got %v", pm.Type)
	}
}

func TestCompileWildcardValid(t *testing.T) {
	pm, err := Compile("https://*.accelbyte.io")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if pm.Type != PatternTypeWildcard {
		t.Errorf("Expected PatternTypeWildcard, got %v", pm.Type)
	}
}

func TestCompileRegex(t *testing.T) {
	pm, err := Compile("re:^https://.*\\.example\\.com$")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if pm.Type != PatternTypeRegex {
		t.Errorf("Expected PatternTypeRegex, got %v", pm.Type)
	}
}

func TestCompileRegexInvalid(t *testing.T) {
	_, err := Compile("re:[invalid")
	if err == nil {
		t.Fatal("Expected error for invalid regex pattern")
	}
}

// Wildcard patterns with a broad suffix (e.g. "*.io") compile without error.
// Validation of what is allowed is the caller's responsibility.
func TestCompileWildcard_BroadSuffixNoError(t *testing.T) {
	patterns := []string{
		"https://*.io",
		"https://*.com",
		"https://*.co",
	}
	for _, p := range patterns {
		pm, err := Compile(p)
		if err != nil {
			t.Errorf("Compile(%q) should succeed without error, got: %v", p, err)
		}
		if pm.Type != PatternTypeWildcard {
			t.Errorf("Compile(%q) expected PatternTypeWildcard, got %v", p, pm.Type)
		}
	}
}

func TestMatchOriginExact(t *testing.T) {
	pm, _ := Compile("https://example.com")

	tests := []struct {
		origin   string
		expected bool
	}{
		{"https://example.com", true},
		{"https://other.com", false},
		{"http://example.com", false},
	}

	for _, tt := range tests {
		if pm.MatchOrigin(tt.origin) != tt.expected {
			t.Errorf("MatchOrigin(%s) = %v, expected %v", tt.origin, pm.MatchOrigin(tt.origin), tt.expected)
		}
	}
}

func TestMatchOriginWildcard(t *testing.T) {
	pm, _ := Compile("https://*.accelbyte.io")

	tests := []struct {
		origin   string
		expected bool
	}{
		{"https://api.accelbyte.io", true},
		{"https://admin.accelbyte.io", true},
		{"https://dev.api.accelbyte.io", false}, // Multiple subdomains not allowed
		{"https://accelbyte.io", false},          // No subdomain
		{"http://api.accelbyte.io", false},       // Wrong scheme
		{"https://api.accelbyte.io:8080", true},  // With port
		{"https://evilaccelbyte.io", false},      // Different domain
	}

	for _, tt := range tests {
		if pm.MatchOrigin(tt.origin) != tt.expected {
			t.Errorf("MatchOrigin(%s) = %v, expected %v", tt.origin, pm.MatchOrigin(tt.origin), tt.expected)
		}
	}
}

func TestMatchOriginWildcardWithPort(t *testing.T) {
	pm, _ := Compile("https://*.example.com:8080")

	tests := []struct {
		origin   string
		expected bool
	}{
		{"https://api.example.com", true},
		{"https://api.example.com:8080", true},
		{"https://api.example.com:9000", true}, // Port in pattern is ignored
	}

	for _, tt := range tests {
		if pm.MatchOrigin(tt.origin) != tt.expected {
			t.Errorf("MatchOrigin(%s) = %v, expected %v", tt.origin, pm.MatchOrigin(tt.origin), tt.expected)
		}
	}
}

func TestMatchOriginRegex(t *testing.T) {
	pm, _ := Compile("re:^https://.*\\.example\\.com$")

	tests := []struct {
		origin   string
		expected bool
	}{
		{"https://api.example.com", true},
		{"https://admin.example.com", true},
		{"https://a.b.c.example.com", true},
		{"https://example.com", false}, // No subdomain
		{"http://api.example.com", false},
	}

	for _, tt := range tests {
		if pm.MatchOrigin(tt.origin) != tt.expected {
			t.Errorf("MatchOrigin(%s) = %v, expected %v", tt.origin, pm.MatchOrigin(tt.origin), tt.expected)
		}
	}
}

func TestMatchOriginExactAllowAll(t *testing.T) {
	// "*" is a special exact match pattern that allows any origin
	pm, err := Compile("*")
	if err != nil {
		t.Fatalf("Expected no error for \"*\", got %v", err)
	}

	tests := []struct {
		origin   string
		expected bool
	}{
		{"https://example.com", true},
		{"http://any.domain.io", true},
		{"any-origin", true},
	}

	for _, tt := range tests {
		if pm.MatchOrigin(tt.origin) != tt.expected {
			t.Errorf("MatchOrigin(%s) = %v, expected %v", tt.origin, pm.MatchOrigin(tt.origin), tt.expected)
		}
	}
}

func TestValidateWildcardPattern_Valid(t *testing.T) {
	tests := []string{
		"https://*.accelbyte.io",
		"https://*.example.com",
		"https://*.co.uk",
		"https://*.game-studio.io",
	}

	for _, pattern := range tests {
		_, err := Compile(pattern)
		if err != nil {
			t.Errorf("Pattern %s should be valid, got error: %v", pattern, err)
		}
	}
}

func TestWildcardMatchingEdgeCases(t *testing.T) {
	tests := []struct {
		pattern  string
		origin   string
		expected bool
	}{
		// Subdomain with multiple levels should not match
		{"https://*.example.io", "https://a.b.example.io", false},
		// Exactly the suffix (no subdomain) should not match
		{"https://*.example.io", "https://example.io", false},
		// Case-insensitive subdomain matching
		{"https://*.example.io", "https://API.example.io", true},
	}

	for _, tt := range tests {
		pm, err := Compile(tt.pattern)
		if err != nil {
			t.Fatalf("Failed to compile pattern %s: %v", tt.pattern, err)
		}
		if pm.MatchOrigin(tt.origin) != tt.expected {
			t.Errorf("Pattern %s matching %s = %v, expected %v",
				tt.pattern, tt.origin, pm.MatchOrigin(tt.origin), tt.expected)
		}
	}
}

func TestWildcardWithMultipleDots(t *testing.T) {
	// *.co.uk should match example.co.uk
	pm, err := Compile("https://*.co.uk")
	if err != nil {
		t.Fatalf("Expected no error for *.co.uk, got %v", err)
	}

	if !pm.MatchOrigin("https://example.co.uk") {
		t.Error("Pattern *.co.uk should match example.co.uk")
	}
}

// Wildcard not in subdomain position will simply not match — no compile error.
func TestWildcard_NotInSubdomain_NoMatch(t *testing.T) {
	tests := []struct {
		pattern string
		origin  string
	}{
		{"https://example.*.com", "https://example.api.com"},
		{"https://example.*", "https://example.com"},
	}
	for _, tt := range tests {
		pm, err := Compile(tt.pattern)
		if err != nil {
			t.Errorf("Compile(%q) should not error, got: %v", tt.pattern, err)
			continue
		}
		if pm.MatchOrigin(tt.origin) {
			t.Errorf("Pattern %q should not match %q", tt.pattern, tt.origin)
		}
	}
}
