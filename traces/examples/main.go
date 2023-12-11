package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/twistingmercury/monitoring/traces"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"time"
)

func main() {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		panic(err)
	}

	// any trace.SpanExporter can be used here
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
