# cors

CORS (Cross-Origin Resource Sharing) filter for go-restful. Supports static service-level configuration and optional dynamic namespace-scoped configuration fetched from justice-config-service.

## Usage

Use `NewCrossOriginResourceSharing` to initialize the filter. This is the recommended way to set up CORS as it makes all configuration explicit:

```go
import (
    "github.com/AccelByte/go-restful-plugins/v4/pkg/cors"
    iam "github.com/AccelByte/iam-go-sdk/v2"
)

filter := cors.NewCrossOriginResourceSharing(
    "http://justice-config-service/config", // configServiceURL: set empty string to disable dynamic config
    iamClient,                              // iam.Client from iam-go-sdk/v2; pass nil to disable auth
    []string{"https://example.com", "https://*.mycompany.io"}, // allowedDomains
    []string{"GET", "POST", "PUT", "DELETE"},                  // allowedMethods
    []string{"Content-Type", "Authorization"},                 // allowedHeaders
    []string{},                                                // exposeHeaders
    true,                                                      // cookiesAllowed
    3600,                                                      // maxAge (seconds)
)
filter.Container = container
container.Filter(filter.Filter)
```

**Static config only** (no dynamic config, no IAM auth):

```go
filter := cors.NewCrossOriginResourceSharing(
    "",   // empty configServiceURL disables dynamic config
    nil,  // nil iamClient disables auth
    []string{"https://example.com"},
    []string{"GET", "POST"},
    []string{"Content-Type"},
    []string{},
    true,
    0,
)
filter.Container = container
container.Filter(filter.Filter)
```

## Allowed Domain Patterns

`AllowedDomains` supports four pattern types:

| Pattern | Example | Behavior |
|---------|---------|----------|
| Exact | `https://example.com` | Matches only that exact origin |
| Allow-all | `*` | Matches any origin |
| Wildcard | `https://*.mycompany.io` | Matches one subdomain level (`game.mycompany.io` ✅, `a.b.mycompany.io` ❌) |
| Regex | `re:^https://.*\.example\.com$` | Full regular expression match |

**Wildcard validation:** the static host after `*.` must contain at least one dot. `https://*.io` is rejected as too broad; `https://*.accelbyte.io` is valid.

## Dynamic Namespace-Scoped Configuration

When `configServiceURL` is non-empty, the filter fetches per-namespace CORS config from justice-config-service on the first request and caches it for 1 minute. If empty, only static config is used.

### Namespace Resolution

The namespace is resolved from each request in priority order:

1. **Path parameter** — route contains `{namespace}` (e.g. `/namespaces/accelbyte/...` → `accelbyte`)
2. **Subdomain** — first component of `Host` header with ≥3 parts (e.g. `game-ns.prod.example.io` → `game-ns`)
3. **Header** — `x-ab-rl-ns` request header
4. **Fallback** — service static config used if no namespace found

### Config Merging

Namespace config is merged with service defaults:

- **List fields** (`allowed_domains`, `allowed_headers`, `allowed_methods`, `expose_headers`): combined and deduplicated
- **Scalar fields** (`cookies_allowed`, `max_age`): namespace value takes precedence

```
Service config:   allowed_domains: ["https://service.com"]
Namespace config: allowed_domains: ["https://*.game.example.com"]
Merged result:    allowed_domains: ["https://service.com", "https://*.game.example.com"]
```

### Config Service Response Format

```json
{
  "namespace": "game3",
  "key": "CORS",
  "value": "{\"allowed_domains\":[\"https://*.example.com\"],\"allowed_methods\":[\"GET\",\"POST\"],\"allowed_headers\":[\"Content-Type\"],\"cookies_allowed\":true,\"max_age\":3600}",
  "createdAt": "2026-02-09T07:29:24.357Z",
  "updatedAt": "2026-02-09T07:29:24.357Z",
  "isPublic": false
}
```

### IAM Authentication

The `iamClient` parameter is typed as `iam.Client` from `github.com/AccelByte/iam-go-sdk/v2` — the same SDK used throughout the rest of the plugin stack. When provided, `ClientToken()` is called before each config-service request and injected as `Authorization: Bearer <token>`. Pass `nil` to make unauthenticated requests.
