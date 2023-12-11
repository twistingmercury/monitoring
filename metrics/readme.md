#  Monitoring: Metrics

This repository contains fn middleware for [gin and gonic](https://github.com/gin-gonic/gin) using [github.com/prometheus/client_golang]( https://pkg.go.dev/github.com/prometheus/client_golang/prometheus). 

:eyes: At the time this was initially worked on, OTel metrics for Go were not stable. They are stable now, and will eventually be migrated to use [OTel Metrics](https://opentelemetry.io/docs/instrumentation/go/manual/#metrics).

This middleware will add collectors (vectors) for each endpoint:

| Metric                                                     | Description                          |
| ---------------------------------------------------------- | ------------------------------------ |
| `<namespace>_total_calls_<api name>_<http method>`         | Total number of requests             |
| `<namespace>_current_calls_<api_name>_<http method>`       | Current number of active requests    |
| `<namespace>_request_duration_ms_<api_name>_<http method>` | Duration of each request (histogram) |

Since vectors are used, the three counters will be incremented using following labels:

* `path`            : The path that was invoked
* `http_method`     : Which method/verb was used, e.g., GET, POST, PUT, PATCH, DELETE, and so on,
* `status_code`     : The result of the call, i.e., 2xx, 3xx, and so on.

## Installation

```bash
go get -u github.com/twistingmercury/monitoring
```

## Initialization

This is the general process for initializing the metrics:

1. Initialize the metrics with the `metrics.Initialize` function. This function must be called first. This function takes three parameters:
    * The port to expose the metrics on
    * The namespace used to help identify the metrics
  
2. Register any custom metrics with the `metrics.RegisterCustomMetrics` function. This function takes one or more `prometheus.Collector` instances. Creating fn `prometheus.Collector` is beyond the scope of this document. See the [prometheus documentation](https://pkg.go.dev/github.com/prometheus/client_golang/prometheus@v1.17.0#pkg-types) for more information.

3. Publish the metrics with the `metrics.Publish` function. This function takes no parameters.

## Usage

### Instrumenting RESTful APIs

A working example can be found in the [examples](./examples/main.go) directory.
The below code sample demonstrates basic usage:

```go
func main() {
    // setup zerolog...

	gin.SetMode(gin.DebugMode)

	// Metrics are hosted on in a separate goroutine on the port specified.
	// This needs to be invoked before any metrics are registered. It can be called
	// multiple times, but only the first call will have any effect.
	metrics.Initialize("9090", "examples")

	// Publish exposes the metrics for scraping. This needs to be called after
	// all metrics have been registered. It can be called multiple times, but
	// only the first call will have any effect.
	metrics.Publish()

	// Create a gin router and add the middleware to it as one normally would.
	r := gin.New()
	r.Use(metrics.GinMetricsMiddleWare())

    // proceed with setting up gin...
}
```
### Instrumenting other Funcs

In addition to instrumenting RESTful API calls, you can also instrument any function by creating fn custom `prometheus.Metric`
and registering it with the `metrics.RegisterCustomMetrics` function. The below code sample demonstrates basic usage:

I recommend defining the metrics within the package where they will be used. This will help keep the code organized.

```go  
package somePkg

// Metrics returns the metrics that are defined for the data package.
func Metrics() ([]prometheus.Collector) {
	labels := []string{apiLabel, pckLabel, fncLabel, errLabel}

	tCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metrics.Namespace(),     // You can use the namespace set during initialization, or use a different one.
		Name:      "<package>_total_calls", // I typically use package name as a prefix.
		Help:      "The total count of calls to the funcs in the data package"},
		labels)

	dHist = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.Namespace(),
		Name:      "data_call_duration",
		Help:      "Duration each func call within the data package",
		Buckets:   prometheus.ExponentialBuckets(0.1, 1.5, 5),
	}, labels)

	return []prometheus.Collector{tCtr, dHist}
}
```

Then register the custom metrics with the `metrics.RegisterCustomMetrics` function before you make the call to publish:

```go
    // initialize the metrics as in the previous examples...

    cMetrics:= data.Metrics()
    metrics.RegisterCustomMetrics(cMetrics...)

    metrics.Publish()

    // continue as in the previous examples...
```
