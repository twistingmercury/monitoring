# Observability Wrappers: OTEL Tracing

This repository contains a middleware for [gin and gonic](https://github.com/gin-gonic/gin) for services that are
need tracing using the [Open Telemetry]("go.opentelemetry.io/otel/trace") framework for Go.

## Installation

```bash
go get -u github.com/twistingmercury/monitoring
```

## Collectors and Agents

This has been tested using the [OTEL Collector](https://github.com/open-telemetry/opentelemetry-collector-contrib) and
the [Datadog Agent](https://docs.datadoghq.com/containers/docker/?tab=standard), with the destination being Datadog. Both
agents were tested using both http and grpc protocols.

### Trace and Log Correlation

This middleware pairs nicely with
the [Monitoring: Logging](../logs/readme.md).
When used together, the logs and traces will be correlated in by adding the trace id to the log entry.

## Initialization

### trace.SpanExporter

You will need to provide a trace.SpanExporter to the middleware. This can be any type that implements the
[trace.SpanExporter](https://pkg.go.dev/go.opentelemetry.io/otel/sdk/trace#SpanExporter) interface.

```go

func main() {
	// create a exporter...
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		panic(err)
	}

	// any trace.SpanExporter can be used here.
	shutdown, err := traces.Initialize(exporter, "0.0.1", "trace-example", time.Now().String(), "A12BC3", "localhost")
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = shutdown(context.Background())
	}()

	gin.SetMode(gin.DebugMode)
	r := gin.New()
	r.Use(gin.Recovery(), traces.GinTracingMiddleware())
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})
	if err := r.Run(":8080"); err != nil {
		panic(err)
	}
}

```
A working example is here: [examples](./examples/main.go).

