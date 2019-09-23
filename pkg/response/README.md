# Response with Logger

This package enables response with event logging in go-restful apps.
For the event logging please read logger/event package

## Usage

### Importing

```go
import "github.com/AccelByte/go-restful-plugins/v3/pkg/response"
```

### Write Response Success

```go
Write(request, response, httpStatusCode, serviceType, eventID, message, entity)
```

### Write Response Error

```go
WriteError(request, response, httpStatusCode, serviceType, eventErr, errorResponse)
```

### Error Response Example 
```go
&Error{
    ErrorCode:    unableToWriteResponse,
    ErrorMessage: "unable to write response",
    ErrorLogMsg:  fmt.Sprintf("unable to write response: %+v, body: %+v, error: %v", response, entity, err),
})
```
We recommend use `"github.com/pkg/errors"` to create error and wrap the errors with stack trace to help with debugging