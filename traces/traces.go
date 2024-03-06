// Package traces provides a wrapper around OpenTelemetry to add standard fields to the span.
package traces

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelCodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	isInitialized bool
	tp            *sdktrace.TracerProvider
	tracer        trace.Tracer
	commonAttrs   []attribute.KeyValue
)

type noopExporter struct{}

func (n noopExporter) ExportSpans(_ context.Context, _ []sdktrace.ReadOnlySpan) error {
	return nil
}

func (n noopExporter) Shutdown(_ context.Context) error {
	return nil
}

func NewNoopExporter() sdktrace.SpanExporter {
	return noopExporter{}
}

// NewHTTPExporter creates a new HTTP exporter.
func NewHTTPExporter(ctx context.Context, url string, opts ...otlptracehttp.Option) (exporter sdktrace.SpanExporter, err error) {
	opts = append(opts, otlptracehttp.WithEndpoint(url))
	return otlptracehttp.New(ctx, opts...)
}

// Initialize initializes the tracing system.
func Initialize(exporter sdktrace.SpanExporter, ver, apiName, buildDate, commitHash, env string) (shutdown func(context.Context) error, err error) {
	isInitialized = false
	ctx := context.Background()

	// all traces will share these attributes
	commonAttrs = []attribute.KeyValue{
		semconv.ServiceNameKey.String(apiName),
		semconv.ServiceVersionKey.String(ver),
		{Key: "buildDate", Value: attribute.StringValue(buildDate)},
		{Key: "commitHash", Value: attribute.StringValue(commitHash)},
		{Key: "env", Value: attribute.StringValue(env)},
	}
	res, _ := resource.New(ctx, resource.WithAttributes(commonAttrs...))

	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	tp = sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	otel.SetTracerProvider(tp)

	// set global propagator to trace context (the default is no-op).
	cp := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	otel.SetTextMapPropagator(cp)
	tracer = tp.Tracer(apiName)

	isInitialized = true
	shutdown = tp.Shutdown
	return
}

// Start starts a new span that has no parent. This is merely a convenience function that wraps the Start function of the tracer.
// in: ctx: The context. If nil, an error is returned.
// in: spanName: The name of the span.
// in: kind: The arg kind is used to set the span kind. The constant trace.SpanKind is defined here: https://pkg.go.dev/go.opentelemetry.io/otel/trace@v1.15.
// in: attributes: The attributes to add to the span.
// out: ctx: The context with the span added.
// out: span: The span.
// out: err: The error if the context is nil.
func Start(ctx context.Context, spanName string, kind trace.SpanKind, attributes ...attribute.KeyValue) (spanCtx context.Context, span trace.Span, err error) {
	if !isInitialized {
		panic("traces.Initialize() must be invoked before invoking NewSpan()")
	}

	if ctx == nil {
		err = fmt.Errorf("context is nil")
		return
	}

	if len(attributes) > 0 {
		commonAttrs = append(commonAttrs, attributes...)
	}

	spanCtx, span = tracer.Start(
		ctx,
		spanName,
		trace.WithSpanKind(kind),
		trace.WithAttributes(commonAttrs...))
	return
}

// End ends the span with the supplied status and error. This is merely a convenience function that wraps the the trace.Span SetStatus and End functions of the span.
// in: span: The span.
// in: status: The status code. 0 is "unset", 1 is "error", and 2 is "ok".
// in: err: The error. Can be nil. If the status is "error", the error the error is used as the description for the status.
func End(span trace.Span, status otelCodes.Code, err error) {
	msg := ""
	if err != nil {
		msg = fmt.Sprintf("%v", err)
	}
	span.SetStatus(status, msg)
	span.End()
}

// NewSpan starts a new span that is a child of the existing span within the supplied context.
// in: ctx: The context. If nil, an error is returned.
// in: spanName: The name of the span.
// in: kind: The arg kind is used to set the span kind. The constant trace.SpanKind is defined here: https://pkg.go.dev/go.opentelemetry.io/otel/trace@v1.15.
// in: attributes: The attributes to add to the span.
// out: ctx: The context with the span added.
// out: span: The span.
// out: err: The error if the context is nil.
//
// Deprecated: Use traces.Start(context.Context, string, trace.SpanKind, ...attribute.KeyValue) instead. This function will be removed in v2.0.0.
func NewSpan(traceCtx context.Context, spanName string, kind trace.SpanKind, attributes ...attribute.KeyValue) (spanCtx context.Context, span trace.Span, err error) {
	if !isInitialized {
		panic("traces.Initialize() must be invoked before invoking NewSpan()")
	}

	if traceCtx == nil {
		err = fmt.Errorf("context is nil")
		return
	}

	if len(attributes) > 0 {
		commonAttrs = append(commonAttrs, attributes...)
	}

	spanCtx, span = tracer.Start(
		traceCtx,
		spanName,
		trace.WithSpanKind(kind),
		trace.WithAttributes(commonAttrs...))
	return
}

// EndOK ends the span with a status of "ok".
//
// Deprecated: use traces.End(trace.Span, otel.Codes, string) instead. This function will be removed in v2.0.0.
func EndOK(span trace.Span) {
	span.SetStatus(otelCodes.Ok, "ok")
	span.End()
}

// EndError ends the span with a status of "error".
//
// Deprecated: use traces.End(trace.Span, otel.Codes, string) instead. This function will be removed in v2.0.0.
func EndError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(otelCodes.Error, "error")
	span.End()
}

func GinTracingMiddleware() gin.HandlerFunc {
	if !isInitialized {
		panic("traces.Initialize() must be invoked before using the tracing middleware")
	}

	return func(c *gin.Context) {
		_, span := tracer.Start(
			c.Request.Context(),
			c.Request.URL.Path,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(commonAttrs...))

		c.Set("trace_id", span.SpanContext().TraceID().String())
		c.Set("span_id", span.SpanContext().SpanID().String())

		log.Trace().Str("path", c.Request.URL.Path).Str("trace_id", span.SpanContext().TraceID().String()).Str("span_id", span.SpanContext().SpanID().String()).Msg("gin tracing middleware invoked")

		c.Next()

		status := c.Writer.Status()

		if status >= 500 {
			if len(c.Errors) > 0 {
				span.RecordError(c.Errors.Last())
			}
			span.SetStatus(otelCodes.Error, "error")
			span.End()
			return
		}
		EndOK(span)
	}
}

func reset() {
	if tp == nil {
		return
	}
	_ = tp.Shutdown(context.Background())
	tp = nil
	tracer = nil
	isInitialized = false
	log.Debug().Msg("tracer reset")
}
