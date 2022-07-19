# Access Log Format Logger

Access log mode is used to view the full body detail of all request and response in the endpoints.

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

- **FULL_ACCESS_LOG_ENABLED**

  Full access log mode will capture request body and response body. Default: `false`

- **FULL_ACCESS_LOG_SUPPORTED_CONTENT_TYPES**

  Supported content types to shown in request_body and response_body log.
  Default: `application/json,application/xml,application/x-www-form-urlencoded,text/plain,text/html`

- **FULL_ACCESS_LOG_MAX_BODY_SIZE**

  Maximum size of request body or response body that will be processed, will be ignored if exceed more than it. Default: `10240` bytes

- **FULL_ACCESS_LOG_REQUEST_BODY_ENABLED**

  Enable capture request body in full access log mode. Default: `true`

- **FULL_ACCESS_LOG_RESPONSE_BODY_ENABLED**

  Enable capture response body in full access log mode. Default: `true`

### Filter sensitive field(s) in request body or response body

Some endpoint might have sensitive field value in its query params, request body or response body.
For security reason, those sensitive field value should be masked before it printed as a log.

The `log.Attribute` filter can be used to define the field(s) that need to be masked.

```go
ws := new(restful.WebService)
ws.Route(ws.GET("/user/{id}").
    Filter(log.AccessLog).
    Filter(log.Attribute(log.Option{
        MaskedQueryParams: "param1,param2",
        MaskedRequestFields: "field1,field2",
        MaskedResponseFields: "field3,field4",
    })).
    To(func(request *restful.Request, response *restful.Response) {
}))
```
