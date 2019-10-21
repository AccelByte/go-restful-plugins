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
ws.Filter(event.Log("realm", "serviceName", event.extractNull))
```

### Log specific endpoint

```go
ws := new(restful.WebService)
ws.Route(ws.GET("/user/{id}").
    Filter(event.Log("realm", "serviceName"", event.extractNull)).
    To(func(request *restful.Request, response *restful.Response) {
}))
```

### Actor User ID & Namespace

If you are using [IAM Auth Filter](https://github.com/AccelByte/go-restful-plugins/tree/master/pkg/auth/iam), 
you can use this `extractFunc` for extracting `iam.JWTClaims` from the `restful.Request` to get the actor's user ID 
and namespace.
```go
extractFunc := func(req *restful.Request) (userID string, clientID []string, namespace string, traceID string, 
	sessionID string) {
		claims := iamAuth.RetrieveJWTClaims(req)
		if claims != nil {
			return claims.Subject, []string{claims.ClientID}, claims.Namespace,
				req.HeaderParameter("X-Ab-TraceID"), req.HeaderParameter("X-Ab-SessionID")
		}
		return "", []string{}, "", req.HeaderParameter("X-Ab-TraceID"), req.HeaderParameter("X-Ab-SessionID")
		}
```

However, if you are using your own auth filter, you need to set the attributes to the request when registering the 
route and you need to implement your custom extractFunc as well. For example:

Set the attributes to the request in the auth filter function 
```go
// get token
token := parseToken(request)

// parse the JWT claims
claims := parseClaim(token)

// set the request attribute with claims
request.SetAttribute("userID", claims.Audience)
request.SetAttribute("clientID", claims.Subject)
request.SetAttribute("namespace", claims.Namespace)
```

The logging filter function 
```go
extractFunc := func(req *restful.Request) (userID string, clientID string, namespace string){
	userID, _ = req.Attribute("userID").(string)
	clientID, _ = req.Attribute("clientID").(string)
	namespace, _ = req.Attribute("namespace").(string)
	
	return userID, clientID, namespace
}

ws := new(restful.WebService)
ws.Route(ws.GET("/user/{id}").
    Filter(event.Log("realm", "serviceName", extractFunc)).
    To(func(request *restful.Request, response *restful.Response) {
}))
```


### Target User ID & Namespace

To put target user ID & namespace to the log, call:

```go
event.TargetUser(req *restful.Request, id, namespace string)
```

### Set event ID & log level

To put event ID & level, call one of:

```go
event.Debug(req *restful.Request, eventID int, eventType int, eventLevel int)
event.Info(req *restful.Request, eventID int, eventType int, eventLevel int)
event.Warn(req *restful.Request, eventID int, eventType int, eventLevel int)
event.Error(req *restful.Request, eventID int, eventType int, eventLevel int)
event.Fatal(req *restful.Request, eventID int, eventType int, eventLevel int)
```

You can put a log message there too.

### Additional log fields

Add any additional log fields using

```go
event.AdditionalFields(req *restful.Request, fields map[string]interface{})
```

Pay attention on the field key name not to overwrite the existing default fields.

### Timestamp format

By importing this package, it will set the logrus timestamp for entire service to use RFC3339 in millisecond precision 
and will force the timezone to be UTC. This timestamp format is required for the AccelByte Event Log service.

However if you want to use your own timestamp format, you can override it by calling `logrus.SetFormatter` in the 
`main()` function of your code before writing any logs.
This is the example of setting timestamp format to use `RFC3339Nano`.
```go
logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.RFC3339Nano})
```
