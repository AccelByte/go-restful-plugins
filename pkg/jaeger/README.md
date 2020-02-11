# Jaeger

This package contains filter for generating `jaeger span`

## Usage

### Importing

```go
import "github.com/AccelByte/go-restful-plugins/v3/pkg/jaeger"
```

### Filter

Filter is restful.FilterFunction for generating jaeger span using zipkin headers

#### Example usage of filter for all endpoints

```go
    span := GetSpanFromRestfulContext(request.Request.Context())
    
    // to add a tag
    AddTag(span, "exampleTag", "tag_value")

    //to add a baggage item
    AddBaggage(span, "exampleBaggage", "example")

    // to add a log
    AddLog(span, "exampleLog", "example")
```