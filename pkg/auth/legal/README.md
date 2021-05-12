# Legal Eligibility Filter

This package enables filtering using Legal service in go-restful apps.

## Usage

### Importing

```go
import "github.com/AccelByte/go-restful-plugins/pkg/auth/legal"
```

### Create filter

This filter depends on [Legal client](https://github.com/AccelByte/legal-go-sdk) passed through the constructor.

```go
filter := legal.NewFilter(legalClient)
```

### Constructing filter

The default `Eligibility()` filter validates the accepted policy versions in JWT claims against the latest crucial mandatory policies.

```go
ws := new(restful.WebService)
ws.Filter(filter.Eligibility())
```

### Reading JWT Claims

`Eligibility()` filter will get JWT claims to `restful.Request.attribute` from the `Auth()` filter.

**Note**

Retrieved claims can be `nil` if the request not filtered using `Auth()` first.  

### Filter all endpoints

```go
ws := new(restful.WebService)
ws.Filter(filter.Eligibility())
```

### Filter specific endpoint

```go
ws := new(restful.WebService)
ws.Route(ws.GET("/user/{id}").
    Filter(filter.Eligibility()).
    To(func(request *restful.Request, response *restful.Response) {
}))
```
