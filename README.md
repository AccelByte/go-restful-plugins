[![Build Status](https://travis-ci.com/AccelByte/go-restful-plugins.svg?branch=master)](https://travis-ci.com/AccelByte/go-restful-plugins)

# Go-Restful Plugins

This project contains plugins for [go-restful](https://github.com/emicklei/go-restful) projects.

## Packages

| Package | Description |
|---------|-------------|
| [pkg/cors](pkg/cors/README.md) | CORS filter with static and dynamic namespace-scoped configuration |
| [pkg/auth/iam](pkg/auth/iam/README.md) | IAM-based authentication filter |
| [pkg/auth/ic](pkg/auth/ic/README.md) | IC-based authentication filter |
| [pkg/logger/common](pkg/logger/common/README.md) | Common request/response logger |
| [pkg/logger/event](pkg/logger/event/README.md) | Event logger |
| [pkg/logger/log](pkg/logger/log/README.md) | Log package |
| [pkg/trace](pkg/trace/README.md) | Trace ID propagation |
| [pkg/jaeger](pkg/jaeger/README.md) | Jaeger tracing integration |
| [pkg/apm/datadog](pkg/apm/datadog/README.md) | Datadog APM integration |
| [pkg/response](pkg/response/README.md) | Standard response helpers |
| [pkg/util](pkg/util/README.md) | Utility functions |
| [pkg/profiling/pprof](pkg/profiling/pprof/README.md) | pprof profiling endpoint |

Find each package's full documentation in its own README.