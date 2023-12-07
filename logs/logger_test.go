package logs_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/twistingmercury/monitoring/logs"
	"go.opentelemetry.io/otel"
	stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const (
	rBody = "hello world"
	noTid = "00000000000000000000000000000000"
	noSid = "0000000000000000"
)

var tracer trace.Tracer

func initTestTracer(t *testing.T) (provider *sdktrace.TracerProvider) {
	exporter, err := stdout.New(stdout.WithPrettyPrint())
	if err != nil {
		t.Fatalf("failed to initialize stdout export pipeline: %v", err)
	}

	provider = sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter))
	otel.SetTracerProvider(provider)

	tracerProvider := sdktrace.NewTracerProvider()

	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tracerProvider)
	tracer = tracerProvider.Tracer("logging_unit_test")
	return
}

func testHandler(ctx *gin.Context) {
	if ctx.Request.Context() == nil {
		log.Fatal("gin context is nil")
	}
	if tracer == nil {
		log.Fatal("tracer is nil")
	}
	_, span := tracer.Start(ctx.Request.Context(), "hello")
	defer span.End()

	ctx.Set("trace_id", span.SpanContext().TraceID().String())
	ctx.Set("span_id", span.SpanContext().SpanID().String())

	println("tid", span.SpanContext().TraceID().String())
	println("sid", span.SpanContext().SpanID().String())
	ctx.String(200, rBody)
}

func TestGinLoggingMiddleware(t *testing.T) {
	tout := &bytes.Buffer{}
	logs.Initialize(zerolog.DebugLevel, "0.0.1", "logs_test", "now", "456789", "local", tout)

	provider := initTestTracer(t)
	defer func() { _ = provider.Shutdown(context.Background()) }()

	gin.SetMode(gin.ReleaseMode)
	tr := gin.New()

	tr.Use(logs.GinLoggingMiddleware())
	tr.GET("/test", testHandler)
	req := newTestRequest(testUserAgents[0].ua)
	w := httptest.NewRecorder()
	tr.ServeHTTP(w, req)

	response, _ := io.ReadAll(w.Body)
	assert.Equal(t, rBody, string(response))
	assert.Equal(t, http.StatusOK, w.Code)

	raw := tout.String()
	assert.Truef(t, len(raw) > 0, "log entry is empty")

	le := make(map[string]any)

	println(raw)

	err := json.Unmarshal(tout.Bytes(), &le)
	if err != nil {
		t.Errorf("failed to unmarshal log entry: %v", err)
	}

	tid := le[logs.TraceIDAttr]
	sid := le[logs.SpanIDAttr]

	assert.NotEqual(t, tid, noTid, "trace id is empty")
	assert.NotEqual(t, sid, noSid, "span id is empty")
}

func newTestRequest(ua string) (req *http.Request) {
	req, _ = http.NewRequest("GET", "/test?shoe_size=9.0", nil)
	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-encoding", "gzip, deflate, br")
	req.Header.Set("connection", "close")
	req.Header.Set("user-agent", ua)
	return req
}

type testUserAgent struct {
	ua      string
	uaType  string
	browser string
}

const nilValue = "<nil>"

var testUserAgents = []testUserAgent{
	{"Mozilla/5.0 (Linux; Android 7.0; SM-T827R4 Build/NRD90M) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.116 Safari/537.36", "mobile", logs.BrowserChrome},
	{"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)", "bot", nilValue},
	{"Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)", "bot", nilValue},
	{"Mozilla/5.0 (compatible; Yahoo! Slurp; http://help.yahoo.com/help/us/ysearch/slurp)", "bot", nilValue},
	{"Mozilla/5.0 (compatible; Baiduspider/2.0; +http://www.baidu.com/search/spider.html)", "bot", nilValue},
	{"Mozilla/5.0 (compatible; YandexBot/3.0; +http://yandex.com/bots)", "bot", nilValue},
	{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/42.0.2311.135 Safari/537.36 Edge/12.246", "desktop", logs.BrowserEdge},
	{"Mozilla/5.0 (iPhone13,2; U; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/602.1.50 (KHTML, like Gecko) Version/10.0 Mobile/15E148 Safari/602.1", "mobile", logs.BrowserSafari},
	{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_2) AppleWebKit/601.3.9 (KHTML, like Gecko) Version/9.0.2 Safari/601.3.9", "desktop", logs.BrowserSafari},
	{"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:15.0) Gecko/20100101 Firefox/15.0.1", "desktop", logs.BrowserFirefox},
	{"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36 OPR/102.0.0.0", "desktop", logs.BrowserOpera},
	{"Mozilla/5.0 (Windows NT 10.0; Trident/7.0; rv:11.0) like Gecko", "desktop", logs.BrowserIE},
}

func TestParseUserAgent(t *testing.T) {
	for _, tua := range testUserAgents {
		kvps := make(map[any]any)
		raw := logs.ParseUserAgent(tua.ua)
		for k, v := range raw {
			kvps[k] = v
		}

		actual := kvps[logs.UserAgentDevice]
		assert.Equal(t, tua.uaType, actual)
	}
}

func testErrHandler(ctx *gin.Context) {
	rCtx, span := tracer.Start(ctx.Request.Context(), "internal server error")
	ctx.Request = ctx.Request.Clone(rCtx)
	defer span.End()
	ctx.String(500, rBody)
}

func testNoTracingHandler(ctx *gin.Context) {
	ctx.String(200, rBody)
}

func TestGinLoggingErrorMiddleware(t *testing.T) {
	provider := initTestTracer(t)
	defer func() { _ = provider.Shutdown(context.Background()) }()
	tout := &bytes.Buffer{}
	logs.Initialize(zerolog.DebugLevel, "0.0.1", "logs_test", "now", "456789", "local", tout)
	gin.SetMode(gin.ReleaseMode)
	tr := gin.New()

	tr.Use(logs.GinLoggingMiddleware())
	tr.GET("/test", testErrHandler)
	req := newTestRequest(testUserAgents[0].ua)
	w := httptest.NewRecorder()
	tr.ServeHTTP(w, req)

	response, _ := io.ReadAll(w.Body)
	assert.Equal(t, rBody, string(response))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGinLoggingMiddleware_no_tracing(t *testing.T) {
	provider := initTestTracer(t)
	defer func() { _ = provider.Shutdown(context.Background()) }()
	tout := &bytes.Buffer{}
	logs.Initialize(zerolog.DebugLevel, "0.0.1", "logs_test", "now", "456789", "local", tout)
	gin.SetMode(gin.ReleaseMode)
	tr := gin.New()

	tr.Use(logs.GinLoggingMiddleware())
	tr.GET("/test", testNoTracingHandler)
	req := newTestRequest(testUserAgents[0].ua)
	w := httptest.NewRecorder()
	tr.ServeHTTP(w, req)

	response, _ := io.ReadAll(w.Body)
	assert.Equal(t, rBody, string(response))
	assert.Equal(t, http.StatusOK, w.Code)

	raw := tout.String()
	assert.Truef(t, len(raw) > 0, "log entry is empty")

	le := make(map[string]any)

	err := json.Unmarshal(tout.Bytes(), &le)
	if err != nil {
		t.Errorf("failed to unmarshal log entry: %v", err)
	}

	assert.Equal(t, le[logs.TraceIDAttr], noTid, "trace id should be empty")
	assert.Equal(t, le[logs.SpanIDAttr], noSid, "span id  should be empty")
}

func TestInitPanicRecovers(t *testing.T) {
	assert.Panics(t, func() { logs.Initialize(zerolog.DebugLevel, "0.0.1", "logs_test", "now", "456789", "local", nil) })
}

func TestDebug(t *testing.T) {
	tout := &bytes.Buffer{}
	logs.Initialize(zerolog.DebugLevel, "0.0.1", "logs_test", "now", "456789", "local", tout)
	logs.Debug(context.TODO(), "test", map[string]any{"arg1": "value1", "arg2": "value2"})
	le := make(map[string]any)
	err := json.Unmarshal(tout.Bytes(), &le)
	if err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}
	tout.Reset()
	assert.Equal(t, zerolog.DebugLevel.String(), le[logs.LogLevel])
}

func TestInfo(t *testing.T) {
	tout := &bytes.Buffer{}
	logs.Initialize(zerolog.InfoLevel, "0.0.1", "logs_test", "now", "456789", "local", tout)
	logs.Info(context.TODO(), "test", map[string]any{"arg1": "value1", "arg2": "value2"})
	le := make(map[string]any)
	err := json.Unmarshal(tout.Bytes(), &le)
	if err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}
	tout.Reset()
	assert.Equal(t, zerolog.InfoLevel.String(), le[logs.LogLevel])
}

func TestWarn(t *testing.T) {
	tout := &bytes.Buffer{}
	logs.Initialize(zerolog.WarnLevel, "0.0.1", "logs_test", "now", "456789", "local", tout)
	logs.Warn(context.TODO(), "test", map[string]any{"arg1": "value1", "arg2": "value2"})
	le := make(map[string]any)
	err := json.Unmarshal(tout.Bytes(), &le)
	if err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}
	tout.Reset()
	assert.Equal(t, zerolog.WarnLevel.String(), le[logs.LogLevel])
}

func TestError(t *testing.T) {
	tout := &bytes.Buffer{}
	logs.Initialize(zerolog.ErrorLevel, "0.0.1", "logs_test", "now", "456789", "local", tout)
	logs.Error(context.TODO(), errors.New("test error"), "test", map[string]any{"arg1": "value1", "arg2": "value2"})
	le := make(map[string]any)
	err := json.Unmarshal(tout.Bytes(), &le)
	if err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}
	tout.Reset()
	assert.Equal(t, zerolog.ErrorLevel.String(), le[logs.LogLevel])
}
