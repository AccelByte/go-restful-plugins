# Util

This package utils in go-restful apps.

## Usage

### Importing

```go
import "github.com/AccelByte/go-restful-plugins/v3/pkg/util"
```

### ExtractDefault

ExtractDefault is default function for extracting attribute for filter event logger

#### Example event log all endpoints with ExtractDefault

```go
ws := new(restful.WebService)
ws.Filter(event.Log("realm", "serviceName", util.ExtractDefault))
```