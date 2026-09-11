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
	"strings"

	"github.com/emicklei/go-restful/v3"
)

const (
	namespaceHeader = "x-ab-rl-ns"

	// namespacePathParameter is the route path parameter that every
	// namespace-scoped AGS endpoint declares, e.g.
	// /config/v1/admin/namespaces/{namespace}/configs.
	namespacePathParameter = "namespace"

	// namespacesPathSegment is the static URL segment that precedes the
	// namespace in those paths.
	namespacesPathSegment = "namespaces"
)

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
	return ExtractNamespaceWithContainer(req, nil, subdomainEnabled, baseDomain)
}

// ExtractNamespaceWithContainer extracts the namespace from the request using
// the priority chain:
//  1. Path parameter (highest priority)
//  2. Preflight route resolution — CORS preflights carry no path parameters,
//     so the namespace is recovered from the preflight's own URL
//  3. Subdomain (from Host header) — only when subdomainEnabled is true
//  4. x-ab-rl-ns header (lowest priority)
//
// container is the go-restful container the CORS filter is installed on, used
// only for step 2; pass nil when there is none (step 2 then falls back to the
// AGS path convention).
//
// Returns the extracted namespace or empty string if not found.
func ExtractNamespaceWithContainer(
	req *restful.Request,
	container *restful.Container,
	subdomainEnabled bool,
	baseDomain string,
) string {
	// Try path parameter first
	if ns := req.PathParameter(namespacePathParameter); ns != "" {
		return ns
	}

	// Try the preflight's own URL. This ranks above subdomain and header for
	// the same reason the path parameter does: it names the namespace the
	// request the browser is about to send will actually target.
	if ns := extractPreflightNamespace(req, container); ns != "" {
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

// extractPreflightNamespace resolves the namespace for a CORS preflight, which
// never has a namespace path parameter: go-restful populates path parameters
// during route dispatch, and services register no route for the OPTIONS method,
// so the CORS filter — a container-level filter, running before routing — sees
// an empty one. Namespace-scoped CORS config was therefore unreachable for any
// preflighted cross-origin request wherever subdomain extraction is off, i.e.
// on Private Cloud (AAX-3366).
//
// The preflight's URL is the URL of the request the browser is about to send,
// so the namespace is recoverable from it two ways, tried in order:
//
//  1. Re-select the route using the method the browser announced in
//     Access-Control-Request-Method, then read the namespace path parameter off
//     the matched route. This honours whatever the route itself declares, and
//     needs the container the filter is installed on.
//  2. The AGS path convention, /namespaces/{namespace}/, which needs no
//     container — the filter is not always attached to one, e.g. when a
//     container's filters are promoted onto its WebServices.
//
// Returns empty string for anything that is not a preflight, leaving ordinary
// requests to the path parameter that routing already gave them.
func extractPreflightNamespace(req *restful.Request, container *restful.Container) string {
	// Same test as isPreflightRequest, minus the filter receiver.
	if req.Request == nil || req.Request.Method != http.MethodOptions {
		return ""
	}
	method := req.Request.Header.Get(restful.HEADER_AccessControlRequestMethod)
	if method == "" || req.Request.URL == nil {
		return ""
	}

	if ns := namespaceFromRoute(req.Request, method, container); ns != "" {
		return ns
	}

	return namespaceFromPath(req.Request.URL.Path)
}

// namespaceFromRoute selects the route the preflight is asking about and
// extracts its namespace path parameter. It uses CurlyRouter because that is
// the router restful.NewContainer installs; a container using RouterJSR311
// falls through to namespaceFromPath, since the container keeps its router
// private.
func namespaceFromRoute(httpReq *http.Request, method string, container *restful.Container) string {
	if container == nil {
		return ""
	}
	webServices := container.RegisteredWebServices()
	if len(webServices) == 0 {
		return ""
	}

	// Shallow copy: SelectRoute reads only the method, URL and headers, and the
	// copy does not outlive this call.
	probe := *httpReq
	probe.Method = method

	ws, route, err := restful.CurlyRouter{}.SelectRoute(webServices, &probe)
	if err != nil || ws == nil || route == nil {
		return ""
	}

	return restful.RouterJSR311{}.ExtractParameters(route, ws, probe.URL.Path)[namespacePathParameter]
}

// namespaceFromPath returns the segment following "namespaces" in urlPath, the
// convention every namespace-scoped AGS endpoint follows.
func namespaceFromPath(urlPath string) string {
	segments := strings.Split(strings.Trim(urlPath, "/"), "/")
	for i, segment := range segments {
		if segment == namespacesPathSegment && i+1 < len(segments) {
			return segments[i+1]
		}
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
