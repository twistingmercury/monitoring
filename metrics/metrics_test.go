package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/twistingmercury/monitoring/metrics"
)

func TestInitializePanics(t *testing.T) {
	defer metrics.Reset()
	assert.Panics(t, func() { metrics.Initialize("") })

	metrics.Reset()
	assert.Panics(t, func() { metrics.Initialize("1023") })

	metrics.Reset()
	assert.Panics(t, func() { metrics.Initialize("49152") })

	metrics.Reset()
	assert.Panics(t, func() { metrics.GinMetricsMiddleWare() })

	metrics.Reset()
	assert.Panics(t, func() { metrics.RegisterCustomMetrics() })

	metrics.Reset()
	assert.Panics(t, func() { metrics.Publish() })
}

func TestInitalize(t *testing.T) {
	defer metrics.Reset()
	metrics.Initialize("1024")
	assert.True(t, metrics.IsInitialized())
	assert.Equalf(t, "1024", metrics.Port(), "Port() should return 1024")
	assert.Equal(t, []string{"path", "http_method", "status_code"}, metrics.MetricApiLabels())
}

func TestPublish(t *testing.T) {
	defer metrics.Reset()
	metrics.Initialize("1024")
	assert.NotPanics(t, func() { metrics.Publish() })
}

func TestGinMiddleware(t *testing.T) {
	defer metrics.Reset()
	metrics.Initialize("1024")
	req, w, r := setupMiddlewareTests("/good", http.MethodGet, http.StatusOK)
	r.ServeHTTP(w, req)

	tVal, err := metrics.GetMetricValue(metrics.TotalCalls())
	assert.NoError(t, err)
	cVal, err := metrics.GetMetricValue(metrics.ConcurrentCalls())
	assert.NoError(t, err)
	dVal, err := metrics.GetMetricValue(metrics.CallDuration())
	assert.NoError(t, err)

	assert.Equal(t, float64(1), tVal)
	assert.Equal(t, float64(0), cVal)
	assert.Greater(t, dVal, float64(0))
}

func TestGinMiddlewareNames(t *testing.T) {
	defer metrics.Reset()
	expected := []string{
		"metrics_test_concurrent_calls",
		"metrics_test_total_calls",
		"metrics_test_call_duration"}
	metrics.Initialize("1024")
	metrics.Publish()
	assert.Equal(t, expected, metrics.MetricNames())
}

func TestRegisterCustomMetrics(t *testing.T) {
	defer metrics.Reset()
	metrics.Initialize("1024")
	metrics.Publish()
	metrics.RegisterCustomMetrics(customMetrics()...)
}

func setupMiddlewareTests(path, method string, status int) (*http.Request, *httptest.ResponseRecorder, *gin.Engine) {
	defer metrics.Reset()
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	gin.SetMode(gin.ReleaseMode)
	r.Use(metrics.GinMetricsMiddleWare())
	r.GET(path, func(c *gin.Context) { c.String(status, "") })
	req, _ := http.NewRequest(method, path, nil)
	return req, w, r
}

// Metrics returns a slice of prometheus.Collector that can be registered
func customMetrics() (c []prometheus.Collector) {
	labels := []string{"function", "cmd_type", "cmd", "result"}
	ctr := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "custom",
		Name:      `counter_1`,
		Help:      "The total count of calls to the func data.DoBusinessLogicStuff",
	}, labels)

	dur := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "custom",
		Name:      `duration_milliseconds`,
		Help:      "Duration the func data.DoDatabaseStuff took to execute successfully",
		Buckets:   prometheus.ExponentialBuckets(0.1, 1.5, 5),
	}, labels)

	c = []prometheus.Collector{ctr, dur}
	return
}
