package metrics

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
)

const (
	initErrMsg = "metrics must be initialized before registering metrics"
)

var (
	apiName         string
	mPort           string
	nspace          string
	registry        *prometheus.Registry
	isInit          bool
	pubOnce         = &sync.Once{}
	initOnce        = &sync.Once{}
	svr             *http.Server
	mNames          []string
	totalCalls      *prometheus.CounterVec
	concurrentCalls *prometheus.GaugeVec
	callDuration    *prometheus.HistogramVec
)

func IsInitialized() bool {
	return isInit
}

func Port() string {
	return mPort
}

// ConcurrentCalls returns the number of concurrent calls to the API.
func ConcurrentCalls() prometheus.Collector {
	return concurrentCalls
}

// TotalCalls returns the total number of calls to the API.
func TotalCalls() prometheus.Collector {
	return totalCalls
}

// CallDuration returns the duration of calls to the API.
func CallDuration() prometheus.Collector {
	return callDuration
}

// MetricNames returns the names of the metrics associated with the Collector.
func MetricNames() []string {
	return mNames
}

// CounterValue returns the value of the metric associated with the Collector
// This is to facilitate unit testing of the package.
func CounterValue(col prometheus.Collector) (v float64, err error) {
	//goland:noinspection GoVetCopyLock
	collect(col, func(m dto.Metric) {
		if h := m.GetHistogram(); h != nil {
			v = float64(h.GetSampleCount())
		} else {
			v = m.GetCounter().GetValue()
		}
	})
	return
}

// collect calls the function for each metric associated with the Collector.
// This is to facilitate unit testing of the package.
//
//goland:noinspection ALL
func collect(col prometheus.Collector, do func(dto.Metric)) {
	c := make(chan prometheus.Metric)
	go func(c chan prometheus.Metric) {
		col.Collect(c)
		close(c)
	}(c)
	for x := range c { // eg range across distinct label vector values
		m := dto.Metric{}
		_ = x.Write(&m)
		do(m)
	}
}

// Namespace returns the Namespace for the metrics of the API.
func Namespace() string {
	return nspace
}

func reset() {
	isInit = false
	svr = nil

	if registry != nil {
		registry.Unregister(concurrentCalls)
		registry.Unregister(totalCalls)
		registry.Unregister(callDuration)
	}

	pubOnce = &sync.Once{}
	initOnce = &sync.Once{}
}

// Initialize initializes metrics system so it can TestRegisterFuncs metrics.
// This must be called before any metrics are registered.
func Initialize(port string, namespace string) {
	initOnce.Do(func() {
		if len(port) == 0 {
			panic("port for metrics must be specified")
		}
		if len(namespace) == 0 {
			panic("namespace for metrics must be specified")
		}

		p, err := strconv.Atoi(port)
		if err != nil || p < 1024 || p > 49151 {
			panic(fmt.Sprintf("invalid port value: `%s`; a valid port is a number between 1024 and 49151", port))
		}

		mPort = port
		nspace = namespace

		idx := strings.LastIndex(os.Args[0], `/`)
		n := strings.TrimLeft(os.Args[0][idx+1:], `_`)
		apiName = strings.Replace(n, `.`, `_`, 1)

		newApiMetrics()

		isInit = true
	})
}

func MetricApiLabels() []string {
	return []string{"path", "http_method", "status_code"}
}

// newApiMetrics creates a new metrics object 'p' is the path of the API and 'm' is the HTTP method of the API.
func newApiMetrics() {
	registry = prometheus.NewRegistry()

	concurentCallsName := normalize(fmt.Sprintf("%s_concurrent_calls", apiName))
	concurrentCalls = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: Namespace(),
		Name:      concurentCallsName,
		Help:      "the count of concurrent calls to the APIs, grouped by API name, path, and response code"},
		[]string{"path", "http_method"})

	totalCallsName := normalize(fmt.Sprintf("%s_total_calls", apiName))
	totalCalls = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: Namespace(),
		Name:      totalCallsName,
		Help:      "The count of all call to the API, grouped by API name, path, and response code"},
		MetricApiLabels())

	callDurationName := normalize(fmt.Sprintf("%s_call_duration", apiName))
	callDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: Namespace(),
		Name:      callDurationName,
		Help:      "The duration in milliseconds calls to the API, grouped by API name, path, and response code",
		Buckets:   prometheus.ExponentialBuckets(0.1, 1.5, 5)},
		MetricApiLabels())

	mNames = []string{concurentCallsName, totalCallsName, callDurationName}

	registry.MustRegister(concurrentCalls, totalCalls, callDuration)
	log.Debug().Msg("newApiMetrics invoked")
}

// Publish exposes the metrics for scraping.
func Publish() {
	pubOnce.Do(func() {
		if !isInit {
			panic(initErrMsg)
		}
		go func() {
			gin.SetMode(gin.ReleaseMode)
			router := gin.New()
			router.Use(gin.Recovery())
			promHandler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
			router.GET("/metrics", gin.WrapH(promHandler))
			router.GET("/metrics/names", func(c *gin.Context) {
				c.JSON(http.StatusOK, mNames)
			})
			svr = &http.Server{
				Addr:    fmt.Sprintf(":%s", mPort),
				Handler: router.Handler(),
			}

			if err := svr.ListenAndServe(); err != nil {
				log.Error().Err(err).Msg("metrics endpoint failed with error")
			}
		}()
		log.Info().Msg("metrics endpoint started")
	})
}

// RegisterCustomMetrics allows one to add a custom metric to the registry. This will panic if Initialize has not
// been called first. This is useful for adding metrics that are not API related. You can add Gauge, Counter, and
// Histogram metrics that you have defined.
func RegisterCustomMetrics(cMetrics ...prometheus.Collector) {
	if !isInit {
		panic(initErrMsg)
	}
	registry.MustRegister(cMetrics...)
}

// GinMetricsMiddleWare is a middleware function that captures quantitative metrics for the request.
func GinMetricsMiddleWare() gin.HandlerFunc {
	if !isInit {
		panic(initErrMsg)
	}
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		method := c.Request.Method
		var statusCode string
		var duration float64

		concurrentCalls.WithLabelValues(path, method).Inc()

		defer func() {
			concurrentCalls.WithLabelValues(path, method).Dec()
			callDuration.WithLabelValues(path, method, statusCode).Observe(duration)
			totalCalls.WithLabelValues(path, method, statusCode).Inc()
		}()

		start := time.Now()
		c.Next()
		duration = float64(time.Since(start).Milliseconds())
		statusCode = strconv.Itoa(c.Writer.Status())
	}
}

func normalize(name string) string {
	name = strings.ReplaceAll(name, ".", "_")
	return strings.ReplaceAll(name, "-", "_")
}
