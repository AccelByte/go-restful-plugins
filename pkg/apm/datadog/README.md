# Datadog Tracing

This package enables Datadog tracing in go-restful apps.

The `Trace()` filter function will automatically detect the trace in HTTP
request header, if any, and generate trace for that request.

## Usage

### Importing

```go
import "github.com/AccelByte/go-restful-plugins/pkg/apm/datadog"
```

### Start the tracer

**Important**, Datadog tracer will not send any trace if not started.
You only need to call this once.

```go
datadog.Start("localhost:8126", "example-service", false)
```

### Trace all endpoints

```go
ws := new(restful.WebService)
ws.Filter(datadog.Trace)
```

### Trace specific endpoint

```go
ws := new(restful.WebService)
ws.Route(ws.GET("/user/{id}").
    Filter(datadog.Trace).
    To(func(request *restful.Request, response *restful.Response) {
}))
```

### Inject trace header to outgoing request

To propagate trace into other services, instrument the request by
injecting Datadog trace to the request.

```go
outReq := httptest.NewRequest("GET", "localhost/example", nil)
Inject(outReq, request)
http.DefaultClient.Do(outReq)
```
