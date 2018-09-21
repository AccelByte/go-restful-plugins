# Event Logger

This package enables logging using AccelByte's Event Log Format in go-restful apps.

## Usage

### Importing

```go
import "github.com/AccelByte/go-restful-plugins/pkg/logger/event"
```

### Log all endpoints

```go
ws := new(restful.WebService)
ws.Filter(event.Log("realm"))
```

### Log specific endpoint

```go
ws := new(restful.WebService)
ws.Route(ws.GET("/user/{id}").
    Filter(event.Log("realm")).
    To(func(request *restful.Request, response *restful.Response) {
}))
```

### Actor User ID & Namespace

By default the logger will try to read `iam.JWTClaims` from the `restful.Request` to get the actor's user ID and namespace

### Target User ID & Namespace

To put target user ID & namespace to the log, call:

```go
event.TargetUser(req *restful.Request, id, namespace string)
```

### Set event ID & log level

To put event ID & level, call one of:

```go
event.Debug(req *restful.Request, eventID int, message ...string)
event.Info(req *restful.Request, eventID int, message ...string)
event.Warn(req *restful.Request, eventID int, message ...string)
event.Error(req *restful.Request, eventID int, message ...string)
event.Fatal(req *restful.Request, eventID int, message ...string)
```

You can put a log message there too.

### Additional log fields

Add any additional log fields using

```go
event.AdditionalFields(req *restful.Request, fields map[string]interface{})
```

Pay attention on the field key name not to overwrite the existing default fields.
