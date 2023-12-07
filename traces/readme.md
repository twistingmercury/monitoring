# Observability Wrappers: OTEL Tracing

This repository contains a middleware for [gin and gonic](https://github.com/gin-gonic/gin) for services that are
need tracing using the [Open Telemetry]("go.opentelemetry.io/otel/trace") framework for Go.

## Installation

```bash
go get -u github.com/twistingmercury/monitoring/traces
```

## Collectors and Agents

This has been tested using the [OTEL Collector](https://github.com/open-telemetry/opentelemetry-collector-contrib) and
the
[Datadog Agent](https://docs.datadoghq.com/containers/docker/?tab=standard), with the destination being Datadog. Both
agents
were tested using both http and grpc protocols.

### Trace and Log Correlation

This middleware pairs nicely with
the [Monitoring: Logging](../logs/readme.md).
When used together, the logs and traces will be correlated in by adding the trace id to the log entry.

## Initialization

### trace.SpanExporter

You will need to provide a trace.SpanExporter to the middleware. This can be any type that implements the
[trace.SpanExporter](https://pkg.go.dev/go.opentelemetry.io/otel/sdk/trace#SpanExporter) interface. There is a built in
helper for setting up an HTTP exporter
via  [traces.NewHttpExporter](https://pkg.go.dev/github.com/twistingmercury/observability-traces/traces#NewHTTPExporter).

```go
package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/twistingmercury/monitoring/traces"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"log/slog"
	"os"
	"time"
)

const (
	serviceName  = "my-service" // build info will be set during the build process
	buildDate    = time.Now().String()
	buildVersion = "v0.0.1"
	buildCommit  = "f1g4b5c6"
)

func main() {
	opt := &slog.HandlerOptions{
		AddSource: false,
		Level:     slog.LevelDebug,
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, opt))
	slog.SetDefault(logger)

	// in PRODUCTION, you most likely should use a secure connection
	exporter, err = traces.NewHTTPExporter(context.Background(), "localhost:4318", otlptracehttp.WithInsecure())
	if err != nil {
		slog.Error("failed to initialize exporter: %v", err)
		os.Exit(1)
	}

	// any trace.SpanExporter can be used here
	shutdown, err := traces.Initialize(exporter, buildVersion, serviceName, buildDate, buildCommit, "local")
	if err != nil {
		slog.Error("failed to initialize exporter: %v", err)
		os.Exit(2)
	}
	defer shutdown(context.Background())

	gin.SetMode(gin.DebugMode)
	r := gin.New()
	r.Use(traces.GinTracingMiddleware())
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "hello world"})
	})
	if err := r.Run(":8080"); err != nil {
		slog.Error("failed to run gin server: %v", err)
		os.Exit(3)
	}
}
```

### Exporting via gRPC

If exporting via gRPC is desired, you can create a gRPC exporter using
the [otlptracegrpc.New](https://pkg.go.dev/go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc#New) func:

```go
package main

import (
    // other imports ...
    "github.com/twistingmercury/monitoring/traces"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
 )
 
//
//...
//

func main(){
    // in PRODUCTION, you should handle errors appropriately, and depending on your use case, you should use 
    // a secure connection in production.
    
    grpcConn, _ := grpc.Dial("localhost:4317", grpc.WithTransportCredentials(insecure.NewCredentials()))
    grpcExporter, _ := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(grpcConn))
    shutdown, _ := traces.Initialize(grpcExporter, buildVersion, serviceName, buildDate, buildCommit, "local")
    defer shutdown(context.Background())
    
    //
    // do more stuff...
    //
}
```

