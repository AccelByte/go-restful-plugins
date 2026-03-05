# CORS Plugin — Test & Verification Summary

## Results

| Metric | Value |
|--------|-------|
| Total tests | 82 (81 pass, 1 skip) |
| Overall coverage | **87.5%** |
| Race conditions | **0** |
| Test files | 7 |

Run command:
```bash
go test ./pkg/cors/... -v -race -count=1
```

---

## Coverage by File

| File | Coverage | Notes |
|------|----------|-------|
| `config_cache.go` | 92% | `Get` error-unwrap path not hit (gcache internal) |
| `config_client.go` | 93% | HTTP redirect path not tested |
| `config_merge.go` | 100% | Full branch coverage |
| `cors_filter.go` | 88% | Legacy `isOriginAllowed`/`doPreflightRequest` partially covered |
| `errors.go` | 60% | `Error()`/`Unwrap()` methods and `newNamespaceExtractionError` not directly called in tests |
| `namespace.go` | 93% | Path-parameter branch tested via integration only (skipped unit) |
| `pattern_matcher.go` | 90% | All match types covered; one `MatchOrigin` nil branch untested |

---

## Test Files

### `config_cache_test.go`
Unit tests for the gcache loading cache.

| Test | Verifies |
|------|----------|
| `TestNewConfigCache` | Cache is created without error |
| `TestCacheLoaderCalledOnMiss` | Loader function invoked on first Get |
| `TestCacheLoaderCalledOnceWithinTTL` | Loader not called again within TTL (cached) |
| `TestCacheNilResult` | `(nil, nil)` from loader is cached (404 fallback) |
| `TestCachePurge` | Single namespace purge forces reload |
| `TestCachePurgeAll` | Full cache flush forces reload for all keys |
| `TestCacheConcurrentAccess` | No data races under concurrent Gets |
| `TestCacheLoaderErrorNotCached` | Loader errors are not cached; next call retries |

### `config_client_test.go`
Unit tests for the HTTP config-service client.

| Test | Verifies |
|------|----------|
| `TestConfigClientGetCORSConfig` | Full happy-path fetch and two-step JSON decode; second call uses cache |
| `TestConfigClientNotFound` | 404 returns `(nil, nil)` — no error, graceful fallback |
| `TestConfigClientServerError` | 5xx returns error |
| `TestConfigClientInvalidJSON` | Malformed outer JSON returns error |
| `TestConfigClientInvalidValueJSON` | Malformed inner `value` JSON returns error |
| `TestConfigClientEmptyValue` | Empty `value` field returns `(nil, nil)` |
| `TestCORSConfigValueJsonTags` | JSON snake_case tags decode correctly |
| `TestIAMClientTokenAuthentication` | `Authorization: Bearer <token>` header sent when IAM client provided |

### `config_merge_test.go`
Unit tests for the merge algorithm.

| Test | Verifies |
|------|----------|
| `TestMergeConfigs_NilNamespaceConfig` | Returns copy of service config when namespace config is nil |
| `TestMergeConfigs_BothNil` | Returns empty config when both inputs are nil |
| `TestMergeConfigs_DeduplicateDomains` | Duplicate domains across service+namespace are deduplicated |
| `TestMergeConfigs_NamespaceOverridesScalars` | `cookies_allowed` and `max_age` use namespace value |
| `TestMergeConfigs_PreservesOrder` | Service domains appear before namespace domains |
| `TestMergeConfigs_AllFields` | All list and scalar fields merged correctly |

### `namespace_test.go`
Unit tests for namespace extraction.

| Test | Verifies |
|------|----------|
| `TestExtractNamespace_FromPathParameter` | *(skipped — requires go-restful routing context)* |
| `TestExtractNamespace_FromSubdomain` | Subdomain extracted from Host with ≥3 parts |
| `TestExtractNamespace_FromHeader` | Falls back to `x-ab-rl-ns` header |
| `TestExtractNamespace_Priority` | Subdomain takes priority over header |
| `TestExtractNamespace_SubdomainOverHeader` | Subdomain wins when both subdomain and header present |
| `TestExtractNamespace_NotFound` | Returns `""` when no namespace source found |
| `TestExtractSubdomain` | Correct first component extracted from multi-part host |
| `TestExtractSubdomain_WithPort` | Port stripped before subdomain extraction |

### `pattern_matcher_test.go`
Unit tests for origin pattern matching.

| Test | Verifies |
|------|----------|
| `TestCompileExactMatch` | Exact URL compiles without error |
| `TestCompileWildcardValid` | `https://*.accelbyte.io` compiles successfully |
| `TestCompileWildcardInvalid_TooBroad` | `https://*.io` (single-label TLD) rejected |
| `TestCompileWildcardInvalid_NoDot` | `https://*` rejected |
| `TestCompileRegex` | `re:` prefix compiles as regex |
| `TestCompileRegexInvalid` | Invalid regex returns error |
| `TestMatchOriginExact` | Exact match accepted/rejected correctly |
| `TestMatchOriginWildcard` | Subdomain matches; base domain and two-level subdomain rejected |
| `TestMatchOriginWildcardWithPort` | Port in wildcard pattern matched correctly |
| `TestMatchOriginRegex` | Regex match accepted/rejected correctly |
| `TestMatchOriginExactAllowAll` | `*` pattern matches any origin |
| `TestValidateWildcardPattern_Valid` | Multi-dot host patterns accepted |
| `TestValidateWildcardPattern_Invalid` | Single-label host patterns rejected |
| `TestWildcardMatchingEdgeCases` | Protocol mismatch, path suffix, wrong domain rejected |
| `TestWildcardWithoutDot` | `*.example` (no dot in host) rejected |
| `TestWildcardWithMultipleDots` | `https://*.a.b.example.io` accepted |

### `cors_filter_test.go`
Unit tests for the main filter logic (static config path).

| Test | Verifies |
|------|----------|
| `TestIsOriginAllowed` | Exact, allow-all, wildcard, empty-list, and denied cases |
| `TestFilter_ActualRequest` | Actual request sets `Access-Control-Allow-Origin`; denied origin omits header |
| `TestFilter_ActualRequest_WildcardDomain` | Wildcard domain in static config matched correctly |
| `TestFilter_PreflightRequest` | OPTIONS sets `Allow-Methods`/`Allow-Headers`; chain not called |
| `TestFilter_PreflightRequest_WildcardDomain` | Wildcard domain in preflight accepted |
| `TestIsValidAccessControlRequestMethod` | Allowed/denied method checked correctly |
| `TestIsValidAccessControlRequestHeader` | Case-insensitive header validation |
| `TestPreflightRequest` | Full preflight response headers verified |
| `TestPreflightRequest_MaxAgeConfigured` | `Access-Control-Max-Age` set when `MaxAge > 0` |
| `TestSetOptionHeaders` | `Allow-Origin`, `Expose-Headers`, `Allow-Credentials` set correctly |

### `cors_dynamic_test.go`
Integration-style tests for the dynamic config path and constructor.

| Test | Verifies |
|------|----------|
| `TestStaticCORSFilter` | Static filter struct initializes with nil `ConfigClient` |
| `TestDynamicCORSFilter` | Dynamic filter struct initializes with non-nil `ConfigClient` |
| `TestGetStaticConfig` | `getStaticConfig()` maps struct fields to `MergedCORSConfig` correctly |
| `TestIsOriginAllowedWithConfig` | 10 sub-cases: exact, allow-all, empty, wildcard (4 cases), regex (2 cases) |
| `TestFilterWithoutOrigin` | Missing `Origin` header skips all CORS logic; chain called |
| `TestFilterWithDynamicConfig` | Namespace config fetched via mock `ConfigClient`; allowed origin passes |
| `TestFilterWithDynamicWildcardConfig` | Wildcard in namespace config: allowed/denied origins set/omit header correctly |
| `TestFilterPreflightRequest` | Preflight with dynamic config sets correct headers |
| `TestInitConfigServiceClient` | Empty `ConfigServiceURL` leaves `ConfigClient` nil |
| `TestInitConfigServiceClientWithURL` | URL + IAM client → `ConfigClient` initialized |
| `TestInitConfigServiceClientMissingIAM` | URL set but IAM nil → `ConfigClient` stays nil (error logged) |
| `TestFilterAutoInitConfigClient` | `ConfigClient` lazily initialized on first request |
| `TestNewCrossOriginResourceSharingWithIAM` | Constructor sets all fields correctly |
| `TestNewCrossOriginResourceSharingWithoutIAM` | Constructor with empty URL + nil IAM succeeds |
| `TestNewCrossOriginResourceSharingMissingIAMClient` | Constructor returns error when URL set but IAM nil |
| `TestFilterInitWithIAMClient` | IAM client preserved after filter initialization |

---

## Key Invariants Verified

| Invariant | Where tested |
|-----------|-------------|
| `iamClient` required when `configServiceURL` non-empty | `TestNewCrossOriginResourceSharingMissingIAMClient`, `TestInitConfigServiceClientMissingIAM` |
| IAM bearer token injected in every config-service request | `TestIAMClientTokenAuthentication` |
| 404 from config-service → `nil` config, no error (fallback to static) | `TestConfigClientNotFound` |
| Config-service errors not cached; next request retries | `TestCacheLoaderErrorNotCached` |
| `nil` (404) results cached to prevent hammering | `TestCacheNilResult` |
| Wildcard `*.io` (single-label TLD) rejected | `TestCompileWildcardInvalid_TooBroad` |
| Wildcard matches exactly one subdomain level | `TestMatchOriginWildcard`, `TestIsOriginAllowedWithConfig` |
| Namespace config list fields deduplicated | `TestMergeConfigs_DeduplicateDomains` |
| Namespace config scalar fields override service defaults | `TestMergeConfigs_NamespaceOverridesScalars` |
| No CORS headers set for disallowed origin (chain still called) | `TestFilterWithDynamicWildcardConfig` |
| Preflight does not call chain (handler not executed) | `TestFilter_PreflightRequest`, `TestFilterPreflightRequest` |
| No Origin header → CORS filter is a no-op | `TestFilterWithoutOrigin` |
| No race conditions under concurrent cache access | `TestCacheConcurrentAccess` (run with `-race`) |
