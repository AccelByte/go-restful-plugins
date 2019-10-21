# Trace

This package contains filter for generating `traceID` (`X-Ab-TraceID` header field) if there is no `X-Ab-TraceID` in request header.

## Usage

### Importing

```go
import "github.com/AccelByte/go-restful-plugins/v3/pkg/trace"
```

### Filter

Filter is restful.FilterFunction for generating traceID (X-Ab-TraceID header field) if there is no X-Ab-TraceID in request header.

#### Example usage of filter for all endpoints

```go
ws := new(restful.WebService)
ws.Filter(trace.Filter())
```