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
	"strings"

	"github.com/emicklei/go-restful/v3"
)

const namespaceHeader = "x-ab-rl-ns"

// ExtractNamespace extracts the namespace from the request using the priority chain:
// 1. Path parameter (highest priority)
// 2. Subdomain (from Host header) — only when subdomainEnabled is true
// 3. x-ab-rl-ns header (lowest priority)
//
// Subdomain extraction is controlled by CORS_SUBDOMAIN_ENABLED. When baseDomain is non-empty
// (from CORS_SUBDOMAIN_BASE_DOMAIN_SUFFIX), subdomain extraction is further restricted to
// hosts whose suffix matches ".<baseDomain>", preventing false positives from third-party domains.
//
// Returns the extracted namespace or empty string if not found.
func ExtractNamespace(req *restful.Request, subdomainEnabled bool, baseDomain string) string {
	// Try path parameter first
	if ns := req.PathParameter("namespace"); ns != "" {
		return ns
	}

	// Try subdomain extraction from Host header
	if subdomainEnabled {
		if ns := extractSubdomain(req, baseDomain); ns != "" {
			return ns
		}
	}

	// Try x-ab-rl-ns header
	if ns := req.Request.Header.Get(namespaceHeader); ns != "" {
		return ns
	}

	return ""
}

// extractSubdomain extracts the namespace from the Host header as the first subdomain component.
// For example, "namespace.accelbyte.io" -> "namespace".
// When baseDomain is non-empty, the host must end with ".<baseDomain>" for extraction to proceed,
// preventing third-party domains (e.g. "gamingservices.xsolla.com") from being misidentified.
// Returns empty string if no valid subdomain is found.
func extractSubdomain(req *restful.Request, baseDomain string) string {
	host := req.Request.Host
	if host == "" {
		return ""
	}

	// Remove port if present
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// When baseDomain is set, only proceed if host belongs to that domain
	if baseDomain != "" && !strings.HasSuffix(host, "."+baseDomain) {
		return ""
	}

	// Only extract subdomain if there are at least 3 parts (e.g., sub.example.io)
	// This avoids extracting from simple domains like "example.com" (2 parts)
	parts := strings.Split(host, ".")
	if len(parts) >= 3 {
		return parts[0]
	}

	return ""
}
