package traces_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"time"

	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/twistingmercury/monitoring/traces"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/trace"
)

const (
	noTid = "00000000000000000000000000000000"
	noSid = "0000000000000000"
)

type logEntry struct {
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Msg     string    `json:"msg"`
	TraceID string    `json:"trace_id,omitempty"`
	SpanID  string    `json:"span_id,omitempty"`
}

func TestSpanCreation(t *testing.T) {
	exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		t.Fatal(err)
	}

	shutdown, err := traces.Initialize(exp, "test_service", "test_version", "2023-01-01", "123456", "test")
	if err != nil {
		t.Fatal(err)
	}
	assert.NoError(t, err)
	assert.NotNil(t, shutdown)
	defer func() {
		traces.Reset()
		_ = shutdown(context.Background())
	}()

	attribs := []attribute.KeyValue{
		attribute.String("test", "test"),
		attribute.Int("test_int", 1),
	}

	ctx := context.Background()

	cCtx, span, err := traces.NewSpan(ctx, "test_span", trace.SpanKindUnspecified, attribs...)
	assert.NotNil(t, cCtx)
	assert.NoError(t, err)
	assert.NotNil(t, span)
	traces.EndOK(span)

	_, span, err = traces.NewSpan(context.TODO(), "test_span", trace.SpanKindUnspecified)
	assert.NoError(t, err)
	traces.EndError(span, errors.New("test error"))

	_, _, err = traces.NewSpan(nil, "test_span", trace.SpanKindUnspecified)
	assert.Error(t, err)
}

func TestSpanStart(t *testing.T) {
	exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		t.Fatal(err)
	}

	assert.Panics(t, func() {
		traces.Reset()
		defer traces.Reset()
		traces.Start(context.Background(), "test_span", trace.SpanKindUnspecified)
	})

	shutdown, err := traces.Initialize(exp, "test_service", "test_version", "2023-01-01", "123456", "test")
	if err != nil {
		t.Fatal(err)
	}
	assert.NoError(t, err)
	assert.NotNil(t, shutdown)
	defer func() {
		traces.Reset()
		_ = shutdown(context.Background())
	}()

	attribs := []attribute.KeyValue{
		attribute.String("test", "test"),
		attribute.Int("test_int", 1),
	}

	ctx := context.Background()

	cCtx, span, err := traces.Start(ctx, "test_span", trace.SpanKindUnspecified, attribs...)
	assert.NotNil(t, cCtx)
	assert.NoError(t, err)
	assert.NotNil(t, span)
	traces.EndOK(span)

	_, span, err = traces.Start(context.TODO(), "test_span", trace.SpanKindUnspecified)
	assert.NoError(t, err)
	traces.EndError(span, errors.New("test error"))

	var spanCtx context.Context = nil
	_, _, err = traces.Start(spanCtx, "test_span", trace.SpanKindUnspecified)
	assert.Error(t, err)
}

func TestSpanEnd(t *testing.T) {

	assert.Panics(t, func() {
		traces.Reset()
		defer traces.Reset()
		traces.End(trace.SpanFromContext(context.Background()), codes.Ok, nil)
	})

	buf := &bytes.Buffer{}
	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	zerolog.TimeFieldFormat = time.RFC3339Nano

	svr := mockOtelSvr(t)
	svr.Start()
	defer svr.Close()

	shutdown, err := traces.Initialize(traces.NewNoopExporter(), "test_service", "test_version", "2023-01-01", "123456", "test")
	assert.NoError(t, err)
	assert.NotNil(t, shutdown)
	defer func() {
		traces.Reset()
		buf.Reset()
		_ = shutdown(context.Background())
	}()

	w := httptest.NewRecorder()
	gin.SetMode(gin.TestMode)
	c, r := gin.CreateTestContext(w)
	r.GET("/", func(c *gin.Context) {
		c.Writer.WriteHeader(http.StatusOK)
	})
	c.Request, _ = http.NewRequest("GET", "/test", nil)
	r.Use(traces.GinTracingMiddleware())
	r.ServeHTTP(w, c.Request)

	re := buf.String()
	lines := strings.Split(re, "\n")

	entries := unmarshalLogEntries(lines)

	for _, entry := range entries {
		if len(entry.TraceID) == 0 {
			continue
		}
		assert.Equal(t, "gin tracing middleware invoked", entry.Msg)
		assert.NotEqual(t, noSid, entry.SpanID)
		assert.NotEmpty(t, noTid, entry.TraceID)
	}

	_, span, err := traces.Start(context.Background(), "test_span", trace.SpanKindUnspecified)
	assert.NoError(t, err)

	err = errors.New("test error")
	traces.End(span, codes.Error, err)
}

func TestTraceInitialize(t *testing.T) {
	svr := mockOtelSvr(t)
	defer svr.Close()

	tContext := context.Background()
	exporter, err := traces.NewHTTPExporter(tContext, "http://localhost:4318")
	assert.NoError(t, err)
	assert.NotNil(t, exporter)

	shutdown, _ := traces.Initialize(traces.NewNoopExporter(), "test_service", "test_version", "2023-01-01", "123456", "test")
	defer func() {
		traces.Reset()
		_ = shutdown(context.Background())
	}()
}

func TestGinTracingMiddleware_NotInitialized(t *testing.T) {
	assert.Panics(t, func() {
		traces.Reset()
		defer traces.Reset()

		tm := traces.GinTracingMiddleware()
		assert.NotNil(t, tm)

		w := httptest.NewRecorder()
		gin.SetMode(gin.ReleaseMode)
		_, e := gin.CreateTestContext(w)
		req, _ := http.NewRequest("GET", "/", nil)
		e.Use(tm)
		e.ServeHTTP(w, req)
	})
}

func TestGinTracingMiddleware_EndOK(t *testing.T) {

	buf := &bytes.Buffer{}
	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	zerolog.TimeFieldFormat = time.RFC3339Nano

	svr := mockOtelSvr(t)
	svr.Start()
	defer svr.Close()

	shutdown, err := traces.Initialize(traces.NewNoopExporter(), "test_service", "test_version", "2023-01-01", "123456", "test")
	assert.NoError(t, err)
	assert.NotNil(t, shutdown)
	defer func() {
		traces.Reset()
		buf.Reset()
		_ = shutdown(context.Background())
	}()

	w := httptest.NewRecorder()
	gin.SetMode(gin.TestMode)
	c, r := gin.CreateTestContext(w)
	r.GET("/", func(c *gin.Context) {
		c.Writer.WriteHeader(http.StatusOK)
	})
	c.Request, _ = http.NewRequest("GET", "/test", nil)
	r.Use(traces.GinTracingMiddleware())
	r.ServeHTTP(w, c.Request)

	re := buf.String()
	lines := strings.Split(re, "\n")

	entries := unmarshalLogEntries(lines)

	for _, entry := range entries {
		if len(entry.TraceID) == 0 {
			continue
		}
		assert.Equal(t, "gin tracing middleware invoked", entry.Msg)
		assert.NotEqual(t, noSid, entry.SpanID)
		assert.NotEmpty(t, noTid, entry.TraceID)
	}

}

func TestGinTracingMiddleware_EndError(t *testing.T) {
	buf := &bytes.Buffer{}

	svr := mockOtelSvr(t)
	svr.Start()
	defer svr.Close()

	shutdown, err := traces.Initialize(traces.NewNoopExporter(), "test_service", "test_version", "2023-01-01", "123456", "test")
	assert.NoError(t, err)
	assert.NotNil(t, shutdown)
	defer func() {
		traces.Reset()
		buf.Reset()
		_ = shutdown(context.Background())
	}()

	w := httptest.NewRecorder()
	gin.SetMode(gin.TestMode)
	c, r := gin.CreateTestContext(w)
	r.Use(traces.GinTracingMiddleware())
	r.GET("/", func(c *gin.Context) {
		c.Writer.WriteHeader(http.StatusInternalServerError)
	})
	_, _ = traces.Initialize(traces.NewNoopExporter(), "test_service", "test_version", "2023-01-01", "123456", "test")
	c.Request, _ = http.NewRequest("GET", "/", nil)

	r.ServeHTTP(w, c.Request)

	re := buf.String()
	lines := strings.Split(re, "\n")

	entries := unmarshalLogEntries(lines)

	for _, entry := range entries {
		if len(entry.TraceID) == 0 {
			continue
		}
		assert.Equal(t, "gin tracing middleware invoked", entry.Msg)
		assert.NotEqual(t, noSid, entry.SpanID)
		assert.NotEmpty(t, noTid, entry.TraceID)
	}
}

func unmarshalLogEntries(lines []string) (entries []logEntry) {
	entries = make([]logEntry, 0)

	for _, line := range lines {
		if line == "" {
			continue
		}
		var le logEntry
		_ = json.Unmarshal([]byte(line), &le)
		entries = append(entries, le)
	}
	return
}

func mockOtelSvr(t *testing.T) *httptest.Server {
	svr := httptest.NewUnstartedServer(traceHttpHandler())
	l, err := net.Listen("tcp", "localhost:4318")
	if err != nil {
		t.Fatal(err)
	}
	err = svr.Listener.Close()
	if err != nil {
		t.Fatal(err)
	}
	svr.Listener = l
	return svr
}

func traceHttpHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte(`test`))
	}
}
