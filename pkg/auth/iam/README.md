# IAM Auth Filter

This package enables filtering using IAM service in go-restful apps.

## Usage

### Importing

```go
import "github.com/AccelByte/go-restful-plugins/pkg/auth/iam"
```

### Create filter

This filter depends on [IAM client](https://github.com/AccelByte/iam-go-sdk) passed through the constructor.

The client should be ready to do local token validation by calling `iamClient.StartLocalValidation()` first. To do permission checking too, the client will need client token, which can be retrived using `iamClient.ClientTokenGrant()`.

```go
filter := iam.NewFilter(iamClient)
```

### Constructing filter

The default `Auth()` filter only validates if the JWT access token is valid.

```go
ws := new(restful.WebService)
ws.Filter(filter.Auth())
```

However, it can be expanded through `FilterOption` parameters. There are several built-in expansions in this package ready for use.

```go
ws.Filter(
    filter.Auth(
        iam.WithValidUser(),
        iam.WithPermission(
            &iamSDK.Permission{
                Resource: "NAMESPACE:{namespace}:ECHO",
                Action:   iamSDK.ActionCreate | iamSDK.ActionRead,
            }),
    ))
```

### Reading JWT Claims

`Auth()` filter will inject the parsed IAM SDK's JWT claims to `restful.Request.attribute`. To retrieve it, use:

```go
claims := iam.RetrieveJWTClaims(request)
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
