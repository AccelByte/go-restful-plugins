# IC Auth Filter

This package enables filtering using IC service in go-restful apps.

## Usage

### Importing

```go
import "github.com/AccelByte/go-restful-plugins/v4/pkg/auth/ic"
```

### Create filter

This filter depends on [IC client](https://github.com/AccelByte/ic-go-sdk) passed through the constructor.

Create Filter:
```go
filter := ic.NewFilter(icClient)
```

### Constructing filter

The default `Auth()` filter only validates if the JWT access token is valid.

```go
ws := new(restful.WebService)
ws.Filter(filter.Auth())
```

However, it can be expanded through `FilterOption` parameters. There are several built-in expansions in this package ready for use.

```go
ws.Filter(service.AuthFilter.Auth(
    auth.WithValidUser(),
    auth.WithPermission(&ic.Permission{
        Resource: "ADMIN:ORG:{organizationId}:PROJ:{projectId}:Info",
        Action:   ic.ActionUpdate,
    })),
).
```

### Reading JWT Claims

`Auth()` filter will inject the parsed IC SDK's JWT claims to `restful.Request.attribute`. To retrieve it, use:

```go
claims := ic.RetrieveJWTClaims(request)
```

**Note**

Retrieved claims can be `nil` if the request not filtered using `Auth()`

### Filter all endpoints

```go
ws := new(restful.WebService)
ws.Filter(filter.Auth())
```

### Filter specific endpoint

```go
ws := new(restful.WebService)
ws.Route(ws.GET("/user/{id}").
    Filter(filter.Auth()).
    To(func(request *restful.Request, response *restful.Response) {
}))
```
