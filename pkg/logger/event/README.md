# Event Logger

This package enables logging using AccelByte's Event Log Format in go-restful apps.

## Usage

### Importing

```go
import (
	"github.com/AccelByte/go-restful-plugins/v4/pkg/logger/event"
	"github.com/AccelByte/go-restful-plugins/v4/pkg/util"
)
```

### Log events on all endpoints

```go
ws := new(restful.WebService)
ws.Filter(event.Log("realm", "serviceName", util.ExtractDefault))
```

### Log events on a specific endpoint

```go
ws := new(restful.WebService)
ws.Route(ws.GET("/user/{id}").
    Filter(event.Log("realm", "serviceName", util.ExtractDefault)).
    To(func(request *restful.Request, response *restful.Response) {
}))
```

`event.Log` doesn't log every request. It writes one event line after the request, and only when the handler set an
event with a non-zero event ID by calling one of the functions in [Set the event](#set-the-event). A request that
sets no event produces no line.

### Actor User ID & Namespace

If you are using [IAM Auth Filter](https://github.com/AccelByte/go-restful-plugins/tree/master/pkg/auth/iam),
`util.ExtractDefault` extracts the actor's user ID, client ID and namespace from the `iam.JWTClaims` in the request,
the session and flight IDs from the `X-Ab-SessionID` and `x-flight-id` headers, and the request's OpenTelemetry trace
ID as `trace_id`, the same value as the access log's `trace_id`. The trace ID is empty when the request has no span
and arrived without trace headers.

If you don't need these fields, pass `nil` as the extract function; the event line then leaves them empty.

If you are using your own auth filter, set the actor on the request in that filter, and pass your own extract
function to `event.Log`. For example:

Set the attributes in the auth filter, for each request
```go
// get token
token := parseToken(request)

// parse the JWT claims
claims := parseClaim(token)

// set the request attribute with claims
request.SetAttribute("userID", claims.Subject)
request.SetAttribute("clientID", claims.ClientID)
request.SetAttribute("namespace", claims.Namespace)
```

The extract function
```go
extractFunc := func(req *restful.Request) (userID string, clientID []string, namespace string,
	traceID string, sessionID string, flightID string) {
	// the trace, session and flight IDs as util.ExtractDefault reads them
	_, _, _, traceID, sessionID, flightID = util.ExtractDefault(req)

	userID, _ = req.Attribute("userID").(string)
	if id, ok := req.Attribute("clientID").(string); ok {
		clientID = []string{id}
	}
	namespace, _ = req.Attribute("namespace").(string)

	return userID, clientID, namespace, traceID, sessionID, flightID
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

### Set the event

To set the event's ID, type, level and message, call one of:

```go
event.Debug(req *restful.Request, eventID int, eventType int, eventLevel int, msg string)
event.Info(req *restful.Request, eventID int, eventType int, eventLevel int, msg string)
event.Warn(req *restful.Request, eventID int, eventType int, eventLevel int, msg string)
event.Error(req *restful.Request, eventID int, eventType int, eventLevel int, msg string)
event.Fatal(req *restful.Request, eventID int, eventType int, eventLevel int, msg string)
```

The function sets the logrus level of the event line: `Debug` logs at debug level, `Info` at info level, and so on.
Calling one again in the same request replaces the event set before.

**`event.Fatal` exits the process.** The event line is logged with logrus's `Fatal`, which calls `os.Exit(1)` after
the request, so one request that sets a fatal event stops the whole service, and every service bundled with it in the
same process.

Event lines are written with the global logrus logger, so the service's log level filters them: a service at `warn`
level drops the lines of `event.Info` and `event.Debug` events.

### Additional log fields

Add any additional log fields using

```go
event.AdditionalFields(req *restful.Request, fields map[string]interface{})
```

Each call replaces the fields of the previous call in the same request, so pass all the fields in one call.
The additional fields are added after the default fields, so a key with a default field's name overwrites it.

### Event line fields

| Field              | Value                                                              |
|--------------------|--------------------------------------------------------------------|
| `event_id`         | Event ID set by `event.Debug`, `Info`, `Warn`, `Error` or `Fatal`   |
| `event_type`       | Event type set by the same call                                    |
| `event_level`      | Event level set by the same call                                   |
| `msg`              | Message set by the same call                                       |
| `realm`            | `realm` passed to `event.Log`                                      |
| `service`          | `service` passed to `event.Log`                                    |
| `log_type`         | Always `event`                                                     |
| `user_id`          | User ID from the extract function                                  |
| `client_ids`       | Client IDs from the extract function                               |
| `namespace`        | Namespace from the extract function                                |
| `trace_id`         | Trace ID from the extract function                                 |
| `session_id`       | Session ID from the extract function                               |
| `flight_id`        | Flight ID from the extract function                                |
| `target_user_ids`  | Target user ID set by `event.TargetUser`                           |
| `target_namespace` | Target namespace set by `event.TargetUser`                         |
| `action`           | Action set by `event.Action`                                       |
| `resp`, `status`   | Never set by this package, so always `0` and empty                 |

### Timestamp format

By importing this package, or `pkg/response`, which imports it, the logrus formatter of the entire service is set to a
text formatter with RFC3339 timestamps of up to millisecond precision (trailing zeros are dropped), and the time is
converted to UTC. This timestamp format is required for the AccelByte Event Log service.

If you want to use your own format, call `logrus.SetFormatter` in your code, for example in `main()`. It replaces the
formatter for the lines written after the call. To keep the time in UTC, wrap your formatter in `event.UTCFormatter`.
This is the example of setting the timestamp format to `RFC3339Nano` in UTC:
```go
logrus.SetFormatter(event.UTCFormatter{Formatter: &logrus.TextFormatter{TimestampFormat: time.RFC3339Nano}})
```
