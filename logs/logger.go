package logs

import (
	"context"
	"errors"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/mileusna/useragent"
	"go.opentelemetry.io/otel/trace"

	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

const (
	TraceIDAttr    = "dd.trace_id"
	SpanIDAttr     = "dd.span_id"
	HttpMethod     = "http.request.method"
	HttpPath       = "http.request.path"
	HttpRemoteAddr = "http.request.remoteAddr"
	HttpStatus     = "http.response.status"
	HttpLatency    = "http.response.latency"
	LogLevel       = "level"
)

var (
	logger        zerolog.Logger
	isInitialized bool
)

// Logger returns a pointer to the logger that is
// used by the logging system.
func Logger() *zerolog.Logger {
	return &logger
}

// Initialize initializes the logging system.
// It returns a logger that can be used to log messages, though it is not required.
func Initialize(level zerolog.Level, ver, apiName, buildDate, commitHash, env string, writer io.Writer) {
	if writer == nil {
		panic("nil writer passed to logger")
	}

	zerolog.SetGlobalLevel(level)
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	logger = zerolog.New(writer).
		With().
		Timestamp().
		Str("service", apiName).
		Str("version", ver).
		Str("buildDate", buildDate).
		Str("commitHash", commitHash).
		Str("env", env).
		Logger()

	isInitialized = true
}

// GinLoggingMiddleware logs the incoming request and starts the trace.
func GinLoggingMiddleware() gin.HandlerFunc {
	if !isInitialized {
		panic("logs.Initialize() must be invoked before using the logging middleware")
	}

	return func(ctx *gin.Context) {
		s := time.Now()
		ctx.Next()
		e := time.Since(s)
		status := ctx.Writer.Status()
		args := map[string]any{
			HttpMethod:     ctx.Request.Method,
			HttpPath:       ctx.Request.URL.Path,
			HttpRemoteAddr: ctx.Request.RemoteAddr,
			HttpStatus:     status,
			HttpLatency:    e.String(),
		}

		scheme := "http"
		if ctx.Request.TLS != nil {
			scheme = "https"
			args["http.TLS"] = ctx.Request.TLS.Version
		}

		args["http.scheme"] = scheme
		args["http.request.host"] = ctx.Request.Host

		if rQuery := ctx.Request.URL.RawQuery; len(rQuery) > 0 {
			args[QueryString] = rQuery
		}

		hd := ParseHeaders(ctx.Request.Header)
		args = mergeMaps(args, hd)
		ua := ParseUserAgent(ctx.Request.UserAgent())
		args = mergeMaps(args, ua)
		tId, ok := ctx.Get("trace_id")

		if !ok {
			tId = noTid
		}

		sId, ok := ctx.Get("span_id")
		if !ok {
			sId = noSid
		}

		args[TraceIDAttr] = tId
		args[SpanIDAttr] = sId

		if status > 499 || ctx.Errors.Last() != nil {
			errs := strings.Join(ctx.Errors.Errors(), ";")
			logger.Error().
				Fields(args).
				Err(errors.New(errs)).
				Msg("request failed")
			return
		}

		logger.Info().
			Fields(args).
			Msg("request successful")
	}
}

func mergeMaps(m1 map[string]any, m2 map[string]any) map[string]any {
	merged := make(map[string]any)
	for k, v := range m1 {
		merged[k] = v
	}
	for key, value := range m2 {
		merged[key] = value
	}
	return merged
}

// ParseHeaders parses the headers and returns a map of attributes.
func ParseHeaders(headers map[string][]string) (args map[string]any) {
	args = make(map[string]any)
	for k, v := range headers {
		args[strings.ToLower("http."+k)] = strings.ToLower(strings.Join(v, ", "))
	}
	return
}

const (
	UserAgentOS             = "http.user_agent.os"
	UserAgentOSVersion      = "http.user_agent.os_version"
	UserAgentDevice         = "http.user_agent.device"
	UserAgentBrowser        = "http.user_agent.browser"
	UserAgentBrowserVersion = "http.user_agent.browser_version"
	BrowserChrome           = "chrome"
	BrowserSafari           = "safari"
	BrowserFirefox          = "firefox"
	BrowserOpera            = "opera"
	BrowserIE               = "ie"
	BrowserEdge             = "edge"
	BrowserTrident          = "Trident"
	QueryString             = "http.query"
	noTid                   = "00000000000000000000000000000000"
	noSid                   = "0000000000000000"
)

// ParseUserAgent parses the user agent string and returns a map of attributes.
func ParseUserAgent(rawUserAgent string) (args map[string]any) {
	if len(rawUserAgent) == 0 {
		return //no-op
	}

	args = make(map[string]any)
	ua := useragent.Parse(rawUserAgent)

	args[UserAgentOS] = ua.OS
	args[UserAgentOSVersion] = ua.OSVersion

	var device string
	switch {
	case ua.Mobile || ua.Tablet:
		device = "mobile"
	case ua.Desktop:
		device = "desktop"
	case ua.Bot:
		device = "bot"
	}

	args[UserAgentDevice] = device

	var browser string
	if ua.Mobile || ua.Tablet || ua.Desktop {
		switch {
		case ua.IsChrome():
			browser = BrowserChrome
		case ua.IsSafari():
			browser = BrowserSafari
		case ua.IsFirefox():
			browser = BrowserFirefox
		case ua.IsOpera():
			browser = BrowserOpera
		case ua.IsInternetExplorer() || strings.Contains(rawUserAgent, BrowserTrident):
			browser = BrowserIE
		case ua.IsEdge():
			browser = BrowserEdge
		}

		args[UserAgentBrowser] = browser
		args[UserAgentBrowserVersion] = ua.Version
	}
	return
}

// Debug logs a debug message and adds the trace id and span id fount in the ctx.
// The args are key value pairs and are optional.
func Debug(ctx context.Context, message string, args map[string]any) {
	tInf := traceInfo(ctx)
	args = mergeMaps(args, tInf)
	logger.Debug().
		Fields(args).
		Msg(message)
}

// Info logs an info message and adds the trace id and span id fount in the ctx.
// The args are key value pairs and are optional.
func Info(ctx context.Context, message string, args map[string]any) {
	tInf := traceInfo(ctx)
	args = mergeMaps(args, tInf)
	logger.Info().
		Fields(args).
		Msg(message)
}

// Warn logs a warning message and adds the trace id and span id fount in the ctx.
// The args are key value pairs and are optional.
func Warn(ctx context.Context, message string, args map[string]any) {
	tInf := traceInfo(ctx)
	args = mergeMaps(args, tInf)
	logger.Warn().
		Fields(args).
		Msg(message)
}

// Error logs an error message and adds the trace id and span id fount in the ctx.
func Error(ctx context.Context, err error, message string, args map[string]any) {
	tInf := traceInfo(ctx)
	args = mergeMaps(args, tInf)
	logger.Error().
		Fields(args).
		Err(err).
		Msg(message)
}

// Fatal logs a fatal message and adds the trace id and span id fount in the ctx.
func Fatal(ctx context.Context, err error, message string, args map[string]any) {
	tInf := traceInfo(ctx)
	args = mergeMaps(args, tInf)
	logger.Fatal().
		Fields(args).
		Err(err).
		Msg(message)
}

// traceInfo returns the trace id and span id found in the ctx.
func traceInfo(ctx context.Context) (tMap map[string]any) {
	if !isInitialized {
		panic("log.Initialize() must be invoked before using the logging system")
	}

	tMap = make(map[string]any, 2)
	span := trace.SpanFromContext(ctx)
	spanCtx := span.SpanContext()
	if !spanCtx.TraceID().IsValid() {
		tMap[TraceIDAttr] = noTid
		tMap[SpanIDAttr] = noSid
		return
	}

	tMap[TraceIDAttr] = spanCtx.TraceID().String()
	tMap[SpanIDAttr] = spanCtx.SpanID().String()
	return
}
