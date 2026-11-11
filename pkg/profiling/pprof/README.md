# pprof Routes

> **Deprecated:** register pprof routes on the service's own go-restful container instead. This package registers them on the global `http.DefaultServeMux`, which panics when two services in one process use the same base path. It is kept for compatibility and gets no new features.

This package adds pprof profiling routes to go-restful apps.

## Usage

### Importing

```go
import "github.com/AccelByte/go-restful-plugins/v4/pkg/profiling/pprof"
```

### Add pprof routes

```go
pprof.Route("/basepath")
```

Now the pprof paths can be accessed from `http://host:port/basepath/debug/pprof`.

Reference: [https://golang.org/pkg/net/http/pprof/](https://golang.org/pkg/net/http/pprof/)
