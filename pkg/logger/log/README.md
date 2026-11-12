# Access Log Format Logger

`log.AccessLog` writes one `log_type=access` line per request to stdout, with the method, path, status, duration,
response length, source IP, user agent, referer, `trace_id`, namespace, user ID, client ID, operation, flight ID and
the game, SDK and OSS version headers. In full access log mode (`FULL_ACCESS_LOG_ENABLED=true`), the line also
carries the request and response bodies.

## Usage

### Importing

```go
import "github.com/AccelByte/go-restful-plugins/v4/pkg/logger/log"
```

### Log all endpoints

```go
ws := new(restful.WebService)
ws.Filter(log.AccessLog)
```

### Log specific endpoint

```go
ws := new(restful.WebService)
ws.Route(ws.GET("/user/{id}").
    Filter(log.AccessLog).
    To(func(request *restful.Request, response *restful.Response) {
}))
```

### Environment variables

These are read once, when the program starts, so they must be set in the process environment; setting them later
with `os.Setenv` has no effect. A service can still change the exported `log.FullAccessLog*` variables in code.

- **FULL_ACCESS_LOG_ENABLED**

  Full access log mode adds the request body and response body to the access log line. A response written in
  several chunks is logged as a whole. For a response of a supported content type, `resp_body_size_kb` counts every
  byte the handler writes, before any compression, whether full access logging is on or not. When it is off, the line is
  still written, without the bodies. Default: `false`

  `common.Log` in `pkg/logger/common` also reads this setting: when it is off, `common.Log` writes a common log
  format line instead of the access log line, and that line has no `trace_id`.

- **FULL_ACCESS_LOG_SUPPORTED_CONTENT_TYPES**

  Content types whose bodies are shown in `request_body` and `response_body`, comma-separated. Each entry is matched
  as a substring of the `Content-Type` header, so `application/json; charset=utf-8` matches `application/json`.
  Default: `application/json,application/xml,application/x-www-form-urlencoded,text/plain,text/html`

- **FULL_ACCESS_LOG_MAX_BODY_SIZE**

  Maximum size of a request or response body to log. A larger body is logged as `data too large`; its size is still
  logged in `req_body_size_kb` or `resp_body_size_kb`. Default: `10240` bytes

- **FULL_ACCESS_LOG_REQUEST_BODY_ENABLED**

  Enable capture request body in full access log mode. Default: `true`

  When the request body isn't logged, which includes `FULL_ACCESS_LOG_ENABLED=false`, and its `Content-Length` is
  known, the body isn't read: `req_body_size_kb` comes from the `Content-Length`, so a truncated body is logged with its
  declared size. The handler reads such a body from the connection, so on HTTP/1.x it has to read it before writing
  the response. A body of unknown length, e.g. a chunked one, is read to measure it.

- **FULL_ACCESS_LOG_RESPONSE_BODY_ENABLED**

  Enable capture response body in full access log mode. Default: `true`

### Filter sensitive/PII field(s) in request body or response body

Some endpoint might have sensitive/PII field value in its query params, request body or response body.
For security reason, those sensitive field value should be masked before it printed as a log.

The `log.Attribute` filter can be used to define the field(s) that need to be masked. There are two styles:

- `MaskedQueryParams`, `MaskedRequestFields` and `MaskedResponseFields` replace the whole value with `******`.
- `MaskedPIIQueryParams`, `MaskedPIIRequestFields` and `MaskedPIIResponseFields` mask part of the value and keep its
  first characters; for an email address, the domain is kept.

Query parameters are masked in the logged path whether full access log mode is on or not. Body fields are masked
only in full access log mode, the only mode in which bodies are logged.

```go
ws := new(restful.WebService)
ws.Route(ws.GET("/user/{id}").
    Filter(log.AccessLog).
    Filter(log.Attribute(log.Option{
        MaskedQueryParams: "param1,param2",
        MaskedRequestFields: "field1,field2",
        MaskedResponseFields: "field3,field4",
        MaskedPIIQueryParams: "param1,param2",
        MaskedPIIRequestFields: "field1,field2",
        MaskedPIIResponseFields: "field3,field4",
    })).
    To(func(request *restful.Request, response *restful.Response) {
}))
```

### Manually specify log's field value

We could manually set specific field value via request attribute.
Please refer to the table below for attribute name of each field.

| Field     | Default Value                 | Attribute Name           |
|-----------|-------------------------------|--------------------------|
| namespace | Extracted from IAM JWT claims | `log.NamespaceAttribute` |
| user_id   | Extracted from IAM JWT claims | `log.UserIDAttribute`    |
| client_id | Extracted from IAM JWT claims | `log.ClientIDAttribute`  |

The defaults come from the claims that the `pkg/auth/iam` filter stores on the request. A request authenticated by
another filter, such as `pkg/auth/ic`, gets empty values unless these attributes are set.

Example: 

```go
// ... your service logic
request.SetAttribute(log.NamespaceAttribute, "myNamespace")
request.SetAttribute(log.UserIDAttribute, "myUserId")
request.SetAttribute(log.ClientIDAttribute, "myClientId")
// ... your service logic
```

### Trace ID

The access log's `trace_id` field holds the request's OpenTelemetry trace ID, so a log line can be looked up in the tracing backend. It is, in order:

1. The trace ID of the span in the request context. Register observability-go-sdk's request filter (`trace.InstrumentCommonAttributes`) or another OpenTelemetry filter that puts the span on `req.Request`; it can come before or after `log.AccessLog`.
2. Otherwise, the trace ID of the trace context the request arrived with, such as `traceparent` or B3 headers, read with the global OpenTelemetry propagator.
3. Otherwise, empty.

`trace_id` never holds the `X-Ab-TraceID` request ID.
