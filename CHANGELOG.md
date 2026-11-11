Release v4.29.0 (2026-11-12)
==================
- Update `iam-go-sdk/v2` to v2.9.1 and `ic-go-sdk` to v0.1.1, which removes `go-restful-plugins/v3` and `emicklei/go-restful` v2 from the dependency graph
- Update `emicklei/go-restful/v3` to v3.12.2, `logrus` to v1.9.3 and `go-jose` to v2.1.6
- Require Go 1.26 (`go 1.26.0`)
- Importing `pkg/auth/iam`, `pkg/auth/ic` or `pkg/jaeger` no longer sets the global logrus formatter; `pkg/logger/event` and `pkg/response` still do
- `pkg/jaeger`, `pkg/apm/datadog`: deprecated in favour of OpenTelemetry tracing
- `pkg/logger/event`: deprecate `TraceIDKey`, as `X-Ab-TraceID` is being replaced by the OpenTelemetry trace ID; deprecate `SessionIDKey` and `FlightIDKey` in favour of the new `constant.SessionID` and `constant.FlightID`
- `pkg/logger/event`: `Log` accepts a `nil` extract function, which leaves the actor, trace, session and flight IDs empty
- `pkg/trace`: deprecated, as requests are identified by their OpenTelemetry trace ID instead of `X-Ab-TraceID`
- `pkg/profiling/pprof`: deprecated, as it registers routes on the global `http.DefaultServeMux`; register pprof routes on the service's own go-restful container instead
- `pkg/logger/log`: **Behaviour change:** the access log's `trace_id` is now the request's OpenTelemetry trace ID instead of `X-Ab-TraceID`: the span in the request context, otherwise the incoming trace context, otherwise empty. Services without OpenTelemetry tracing log an empty `trace_id`
- `pkg/util`: **Behaviour change:** `ExtractDefault` returns the request's OpenTelemetry trace ID, the same value as the access log's `trace_id`, instead of the `X-Ab-TraceID` header. The event log's `trace_id` changes too, and so does any value callers take from it, such as a Kafka event's `traceId`
- `pkg/constant`: add `TraceID` (`trace_id`), the log field name for the OpenTelemetry trace ID
- `pkg/cors`: use a namespace only when it is a valid AGS namespace; an invalid value is ignored and the next source is tried, so the `-` that the edge sends in `x-ab-rl-ns` when it has no namespace now falls back to the publisher namespace. A host given as an IP address is no longer read as a subdomain. Escape the namespace in config service URLs
- `pkg/logger/log`: log the whole response body when a handler writes it in several chunks; only the last chunk was logged, with its size instead of the body's. Fields from the earlier chunks are now logged too, so check that `MaskedResponseFields` covers them. `resp_body_size_kb` is the whole body's size also with full access logging off, and an empty JSON response no longer logs a minify warning
- `pkg/logger/log`: fix a data race in the tests, which made `make test` fail intermittently
- Services that use anything deprecated in this release (`pkg/jaeger`, `pkg/apm/datadog`, `pkg/trace`, `pkg/profiling/pprof`, or `event.TraceIDKey`, `SessionIDKey` and `FlightIDKey`) get staticcheck SA1019 warnings after upgrading, which fail `golangci-lint` until the uses are replaced or suppressed
- Migrate golangci-lint to v2.12.2 (`make lint`, `.travis.yml`); `.travis.yml` builds with Go 1.26

Release v4.28.4 (2026-09-16)
==================
- `pkg/cors`: Resolve the namespace during CORS preflight (OPTIONS) requests
  - A preflight carries no path parameters: go-restful extracts those during route dispatch, and
    services register no route for the OPTIONS method, so the CORS filter — a container-level
    filter, running before routing — always saw an empty `namespace`. Namespace-scoped CORS config
    was therefore unreachable for any preflighted cross-origin request wherever subdomain
    extraction is off, i.e. Private Cloud: the preflight was refused and go-restful, finding no
    OPTIONS route, answered 405
  - Fix: `ExtractNamespaceWithContainer` resolves the namespace from the preflight's own URL, by
    re-selecting the route with the method the browser announced in `Access-Control-Request-Method`
    and reading the namespace path parameter off it, falling back to the `/namespaces/{namespace}/`
    path convention when the filter is not attached to a container
  - The new step ranks above subdomain and header extraction, matching the priority the actual
    request gives its path parameter, and applies to preflights only
  - `ExtractNamespace` keeps its signature and its behavior for non-preflight requests

Release v4.28.2 (2026-06-23)
==================
- `pkg/auth/iam`, `pkg/auth/ic`: Fix case-insensitive Bearer scheme parsing per RFC 7235 §2.1
  - `parseAccessToken` was rejecting `Authorization` headers with non-canonical casing (e.g. `bearer`, `BEARER`)
  - Fix: prefix check now lowercases the header value before comparison; original token value is preserved

Release v4.28.1 (2026-04-17)
==================
- `pkg/cors`: Fix CORS filter cancelling request context for downstream handlers
  - `getConfigWithDynamicResolution` was replacing `req.Request` with a `context.WithTimeout`-wrapped
    request and deferring `cancel()`, which cancelled the context the moment the function returned —
    poisoning every downstream DB/cache/RPC call with `context canceled`
  - `ConfigClient.GetCORSConfig` does not accept a `context.Context`, so the timeout context
    was never forwarded to the actual fetch; the `req.Request` replacement served no purpose
  - Fix: removed the timeout context block and `req.Request` replacement entirely
  - Added regression test `TestFilterDoesNotCancelRequestContext`

Release v4.28.0 (2026-04-15)
==================
- `pkg/cors`: Add namespace-scoped dynamic CORS configuration fetched from justice-config-service
  - Namespace resolved from request via path parameter → subdomain → `x-ab-rl-ns` header
  - Merge namespace config with service-level defaults (list fields deduplicated, scalar fields namespace-wins)
  - Wildcard domain pattern support (`https://*.example.io`) in `AllowedDomains`
  - New constructor for explicit initialization:
    `NewCrossOriginResourceSharing(configServiceURL, iamClient, allowedDomains, allowedMethods, allowedHeaders, exposeHeaders, cookiesAllowed, maxAge)`
    - `configServiceURL`: base URL of justice-config-service; set empty string to disable dynamic config
    - `iamClient`: `iam.Client` from `github.com/AccelByte/iam-go-sdk/v2` for bearer-token auth to config-service; pass `nil` to disable auth
  - **Breaking prerequisite for dynamic config**: the IAM client must hold permission
    `[READ] "ADMIN:NAMESPACE:{namespace}:CONFIG:COMMON"` to call justice-config-service for namespace CORS config

Release v3.2.0 (2022-03-29)
===========================
- Add filter to match the subdomain to the namespace in the user token
- Add option to allow validating referer from a subdomain

Release v3.1.3 (2019-12-12)
===========================
- Add `WriteErrorWithEventID` in `response` package
- Update `Write` in `response` package

Release v3.1.2 (2019-10-21)
===========================
- Update auth function to return error response based on standard
- Update auth error response field name to camelCase

Release v3.1.1 (2019-10-21)
===========================
- Update ip extraction on `logger/common` package

Release v3.1.0 (2019-10-18)
===========================
- Add `trace` package
- Update `response` package for Error Code Standards
- Update ExtractDefault in `util` package

Release v3.0.9 (2019-10-02)
===========================
- Update `iam-go-sdk` to v1.1.2

Release v3.0.8 (2019-09-23)
===========================
- Added `response` package
- Added `util` package

Release v3.0.7 (2019-09-19)
===========================
- Updated go module to denote v3 

Release v3.0.6 (2019-08-19)
===========================
- Updated `iam-go-sdk` to v1.1.1 

Release v3.0.5 (2019-07-27)
===========================
- Fixed event codes

Release v3.0.4 (2019-07-27)
===========================
- Fixed `aud` and `scope` validation error response  

Release v3.0.3 (2019-07-25)
===========================
- Separating `aud` and `scope` validation  

Release v3.0.1 (2019-07-24)
===========================
- Fixed `aud` validation error message  

Release v3.0.0 (2019-07-23)
===========================
- Added `aud` and `scope` token fields validation    

Release v2.0.1 (2019-04-26)
===========================
- Export Trace `TraceIDKey` and Session `SessionIDKey` request header constants publicly

Release v2.0.0 (2019-04-24)
===========================
### BREAKING CHANGES
1. package: [logger-event](https://github.com/AccelByte/go-restful-plugins/tree/master/pkg/logger/event): 
    * Change  [event struct](https://github.com/AccelByte/go-restful-plugins/blob/master/pkg/logger/event/event.go) to match new log event standard.
    * Refactor log parameters. (Include ```Debug```, ```Info```, ```Warn```, ```Error```, ```Fatal```)
        * example: ```Info()``` from ```Info(request, 99, "get user info message")``` to ```Info(request, 99, 50, 3, "get user info message")``` , 50 is eventType and 3 is eventLevel.

