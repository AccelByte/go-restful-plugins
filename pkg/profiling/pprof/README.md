# pprof Routes

This package adds pprof profiling routes to go-restful apps.

## Usage

### Importing

```go
import "github.com/AccelByte/go-restful-plugins/pkg/profiling/pprof"
```

### Add pprof routes

```go
pprof.Route("/basepath")
```

Now the pprof paths can be accessed from `http://host:port/basepath/debug/pprof`.

Reference: [https://golang.org/pkg/net/http/pprof/](https://golang.org/pkg/net/http/pprof/)
