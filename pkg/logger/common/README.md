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
