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

func TestMergeConfigs_NilNamespaceConfig(t *testing.T) {
	service := &CORSConfigValue{
		AllowedDomains: []string{"https://example.com"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
		AllowedMethods: []string{"GET", "POST"},
		ExposeHeaders:  []string{"X-Custom-Header"},
		CookiesAllowed: true,
		MaxAge:         3600,
	}

	result := MergeConfigs(service, nil)

	if len(result.AllowedDomains) != 1 || result.AllowedDomains[0] != "https://example.com" {
		t.Errorf("AllowedDomains mismatch: %v", result.AllowedDomains)
	}
	if !result.CookiesAllowed {
		t.Error("CookiesAllowed should be true")
	}
	if result.MaxAge != 3600 {
		t.Errorf("MaxAge mismatch: %d", result.MaxAge)
	}
}

func TestMergeConfigs_BothNil(t *testing.T) {
	result := MergeConfigs(nil, nil)

	if result == nil {
		t.Fatal("MergeConfigs(nil, nil) returned nil")
	}
	if len(result.AllowedDomains) != 0 {
		t.Errorf("AllowedDomains should be empty, got %v", result.AllowedDomains)
	}
}

func TestMergeConfigs_DeduplicateDomains(t *testing.T) {
	service := &CORSConfigValue{
		AllowedDomains: []string{"https://example.com", "https://api.example.com"},
	}
	namespace := &CORSConfigValue{
		AllowedDomains: []string{"https://example.com", "https://cdn.example.com"},
	}

	result := MergeConfigs(service, namespace)

	if len(result.AllowedDomains) != 3 {
		t.Fatalf("AllowedDomains length should be 3, got %d: %v", len(result.AllowedDomains), result.AllowedDomains)
	}
	// Check that no duplicates exist
	seen := make(map[string]bool)
	for _, domain := range result.AllowedDomains {
		if seen[domain] {
			t.Errorf("Duplicate domain found: %s", domain)
		}
		seen[domain] = true
	}
}

func TestMergeConfigs_NamespaceOverridesScalars(t *testing.T) {
	service := &CORSConfigValue{
		CookiesAllowed: false,
		MaxAge:         3600,
	}
	namespace := &CORSConfigValue{
		CookiesAllowed: true,
		MaxAge:         7200,
	}

	result := MergeConfigs(service, namespace)

	if !result.CookiesAllowed {
		t.Error("CookiesAllowed should be true (from namespace)")
	}
	if result.MaxAge != 7200 {
		t.Errorf("MaxAge should be 7200 (from namespace), got %d", result.MaxAge)
	}
}

func TestMergeConfigs_PreservesOrder(t *testing.T) {
	service := &CORSConfigValue{
		AllowedDomains: []string{"https://a.com", "https://b.com"},
	}
	namespace := &CORSConfigValue{
		AllowedDomains: []string{"https://c.com", "https://d.com"},
	}

	result := MergeConfigs(service, namespace)

	expected := []string{"https://a.com", "https://b.com", "https://c.com", "https://d.com"}
	for i, domain := range expected {
		if i >= len(result.AllowedDomains) || result.AllowedDomains[i] != domain {
			t.Errorf("AllowedDomains[%d] should be %s, got %v", i, domain, result.AllowedDomains)
		}
	}
}

func TestMergeConfigs_AllFields(t *testing.T) {
	service := &CORSConfigValue{
		AllowedDomains: []string{"https://service.com"},
		AllowedHeaders: []string{"Content-Type"},
		AllowedMethods: []string{"GET"},
		ExposeHeaders:  []string{"X-Service"},
		CookiesAllowed: false,
		MaxAge:         1000,
	}
	namespace := &CORSConfigValue{
		AllowedDomains: []string{"https://ns.com"},
		AllowedHeaders: []string{"Authorization"},
		AllowedMethods: []string{"POST"},
		ExposeHeaders:  []string{"X-Namespace"},
		CookiesAllowed: true,
		MaxAge:         2000,
	}

	result := MergeConfigs(service, namespace)

	// List fields should contain both
	if len(result.AllowedDomains) != 2 {
		t.Errorf("AllowedDomains should have 2 items, got %d", len(result.AllowedDomains))
	}
	if len(result.AllowedHeaders) != 2 {
		t.Errorf("AllowedHeaders should have 2 items, got %d", len(result.AllowedHeaders))
	}
	if len(result.AllowedMethods) != 2 {
		t.Errorf("AllowedMethods should have 2 items, got %d", len(result.AllowedMethods))
	}
	if len(result.ExposeHeaders) != 2 {
		t.Errorf("ExposeHeaders should have 2 items, got %d", len(result.ExposeHeaders))
	}

	// Scalar fields should use namespace value
	if !result.CookiesAllowed {
		t.Error("CookiesAllowed should be true from namespace")
	}
	if result.MaxAge != 2000 {
		t.Errorf("MaxAge should be 2000 from namespace, got %d", result.MaxAge)
	}
}
