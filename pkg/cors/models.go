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

// CORSConfigValue represents CORS configuration for a service or namespace.
// List fields (AllowedDomains, AllowedHeaders, AllowedMethods, ExposeHeaders) are additive during merge.
// Scalar fields (CookiesAllowed, MaxAge) use the namespace value as override if set (MaxAge=0 means "not set").
type CORSConfigValue struct {
	AllowedDomains []string `json:"allowed_domains"`
	AllowedHeaders []string `json:"allowed_headers"`
	AllowedMethods []string `json:"allowed_methods"`
	ExposeHeaders  []string `json:"expose_headers"`
	CookiesAllowed bool     `json:"cookies_allowed"`
	MaxAge         int      `json:"max_age"` // 0 = "not set", use default
}

// ConfigServiceResponse represents the API response from the justice-config-service.
// The CORS configuration is nested: the outer response has a "value" field which is
// a JSON string containing the actual CORSConfigValue.
// Example:
//   {
//     "namespace": "game3",
//     "key": "CORS",
//     "value": "{\"expose_headers\":[],\"allowed_headers\":[],\"allowed_domains\":[...],\"allowed_methods\":[],\"cookies_allowed\":true}",
//     "createdAt": "2026-02-09T07:29:24.357Z",
//     "updatedAt": "2026-02-09T07:29:24.357Z",
//     "isPublic": false
//   }
type ConfigServiceResponse struct {
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
	// Value is a JSON string that must be decoded to get CORSConfigValue
	Value     string `json:"value"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	IsPublic  bool   `json:"isPublic"`
}

// CORSSubdomainConfig holds subdomain-based namespace extraction settings for the publisher namespace.
// Fetched from the config service using the CORS_SUBDOMAIN key.
type CORSSubdomainConfig struct {
	SubdomainEnabled    bool   `json:"subdomain_enabled"`
	SubdomainBaseDomain string `json:"subdomain_base_domain"`
}

// MergedCORSConfig is the result of merging service-level and namespace-level configs.
// It combines the deduplicated lists from both sources with namespace scalars as overrides.
type MergedCORSConfig struct {
	AllowedDomains []string
	AllowedHeaders []string
	AllowedMethods []string
	ExposeHeaders  []string
	CookiesAllowed bool
	MaxAge         int
}
