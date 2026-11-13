# Util

This package utils in go-restful apps.

## Usage

### Importing

```go
import "github.com/AccelByte/go-restful-plugins/v4/pkg/util"
```

### ExtractDefault

ExtractDefault is default function for extracting attribute for filter event logger.
It returns the actor from the IAM JWT claims, the session and flight IDs from the `X-Ab-SessionID` and `x-flight-id`
headers, and the request's OpenTelemetry trace ID, the same value the access log writes as `trace_id`. The trace ID
is empty when the request has no span and arrived without trace headers; it is never `X-Ab-TraceID`.

#### Example event log all endpoints with ExtractDefault

```go
ws := new(restful.WebService)
ws.Filter(event.Log("realm", "serviceName", util.ExtractDefault))
```