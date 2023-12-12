[![Go](https://github.com/twistingmercury/monitoring/actions/workflows/go-test.yml/badge.svg?branch=develop)](https://github.com/twistingmercury/monitoring/actions/workflows/go-test.yml)

# Monitoring boilerplate for Go apps

This package was created to help me reduce a bunch of repetitive tasks in creating a Go application. All of the apps and services need logging, distributed tracing, metrics and a healthcheck / heartbeat.

All packages are designed around [Gin Web Framework](https://pkg.go.dev/github.com/gin-gonic/gin). The logs, metrics, and traces packages provide middleware that can be used with the gin.Engine:

```go

r := gin.New()
r.Use(logs.GinMiddleware(), traces.GinMiddleware(), metrics.GinMiddleware()

```

## Contents

| Directory                       | Depends on Package(s)                                                           | Description                                                                                                                        |
| ------------------------------- | ------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| [/heatlh](./health/readme.md)   | n/a                                                                             | Provides a custom health-check implementation.                                                                                     |
| [/logs](./logs/readme.md)       | [zerolog](https://pkg.go.dev/github.com/rs/zerolog)                             | Provides logging middleware for gin.engine. Also, it will add the necessary values for ensuring logs and traces can be correlated. |
| [/metrics](./metrics/readme.md) | [Prometheus](https://pkg.go.dev/github.com/prometheus/client_golang/prometheus) | Provides metrics middleware for gin.engine. Uses Prometheus, OTel compatible.                                                                       |
| [/traces](./traces/readme.md)   | [OpenTelemetry-Go](https://pkg.go.dev/go.opentelemetry.io/otel)                 | Provides distributed tracing capability for the gin.engine. Uses OTel.                                                             |

