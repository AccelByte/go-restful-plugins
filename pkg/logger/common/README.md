# Common Log Format Logger

This package enables logging using Common Log Format in go-restful apps.

## Usage

### Importing

```go
import "github.com/AccelByte/go-restful-plugins/pkg/logger/common"
```

### Log all endpoints

```go
ws := new(restful.WebService)
ws.Filter(common.Log)
```

### Log specific endpoint

```go
ws := new(restful.WebService)
ws.Route(ws.GET("/user/{id}").
    Filter(common.Log).
    To(func(request *restful.Request, response *restful.Response) {
}))
```

### Environment variables
#### FULL_ACCESS_LOG_ENABLED
Enable full access log mode. Default: false.

#### FULL_ACCESS_LOG_SUPPORTED_CONTENT_TYPES
Supported content types to shown in request_body and response_body log.
Default: application/json,application/xml,application/x-www-form-urlencoded,text/plain,text/html.

#### FULL_ACCESS_LOG_MAX_BODY_SIZE
Maximum size of request body or response body that will be processed, will be ignored if exceed more than it.
Default: 10240 bytes