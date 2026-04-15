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

// MergeConfigs merges service-level and namespace-level CORS configurations.
// List fields are deduplicated and concatenated; scalar fields use the namespace value as override.
//
// If ns is nil, a copy of the service config is returned.
// If both are nil, a zero-valued MergedCORSConfig is returned.
func MergeConfigs(service, ns *CORSConfigValue) *MergedCORSConfig {
	if ns == nil {
		// No namespace config, use service config
		if service == nil {
			return &MergedCORSConfig{}
		}
		return &MergedCORSConfig{
			AllowedDomains: append([]string{}, service.AllowedDomains...),
			AllowedHeaders: append([]string{}, service.AllowedHeaders...),
			AllowedMethods: append([]string{}, service.AllowedMethods...),
			ExposeHeaders:  append([]string{}, service.ExposeHeaders...),
			CookiesAllowed: service.CookiesAllowed,
			MaxAge:         service.MaxAge,
		}
	}

	// Both service and namespace configs exist; merge them
	result := &MergedCORSConfig{
		AllowedDomains: dedup(append(service.AllowedDomains, ns.AllowedDomains...)),
		AllowedHeaders: dedup(append(service.AllowedHeaders, ns.AllowedHeaders...)),
		AllowedMethods: dedup(append(service.AllowedMethods, ns.AllowedMethods...)),
		ExposeHeaders:  dedup(append(service.ExposeHeaders, ns.ExposeHeaders...)),
		CookiesAllowed: ns.CookiesAllowed, // namespace override
		MaxAge:         ns.MaxAge,          // namespace override (0 = not set, use namespace's value)
	}

	return result
}

// dedup removes duplicate strings from a slice while preserving order.
func dedup(items []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}
