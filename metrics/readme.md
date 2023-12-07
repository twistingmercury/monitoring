#  Observability Wrappers: Metrics

This repository contains fn middleware for [gin and gonic](https://github.com/gin-gonic/gin) using 
[github.com/prometheus/client_golang]( https://pkg.go.dev/github.com/prometheus/client_golang/prometheus). 
This middleware will add counters for each endpoint:

| Metric                                                     | Description |
|------------------------------------------------------------|---|
| `<servic name>_total_calls_<api name>_<http method>`       | Total number of requests |
| `<servic name>_current_calls_<api_name>_<http method>`       | Current number of active requests |
| `<servic name>_request_duration_ms_<api_name>_<http method>` | Duration of each request (histogram) |
| `<servic name>_status_2xx_<api_name>_<http method>`          | Total number of requests with fn 2xx status code |
| `<servic name>_status_4xx_<api_name>_<http method>`          | Total number of requests with fn 4xx status code |
| `<servic name>_status_5xx_<api_name>_<http method>`          | Total number of requests with fn 5xx status code |

For example, assume the service name is `person` and the api names `getPerson`, The following metrics will be created for
that endpoint (assuming the http method is `GET`):

First describe the endpoint to be instrumented...

```go
   info := metrics.MetricInfo{Name: "getPerson", Path: "/person", Method: "GET"}
```

...which will result in the following metrics being created for that endpoint:

```text
person_counter_info_getperson_get
person_current_calls_getperson_get
person_gauge_info_getperson_get
person_request_duration_ms_getperson_get
person_status_2xx_getperson_get
person_status_4xx_getperson_get
person_status_5xx_getperson_get
person_total_calls_getperson_get
```

## Installation

```bash
go get -u github.com/twistingmercury/monitoring-metrics
```

## Initialization

This is the general process for initializing the metrics:

1. Initialize the metrics with the `metrics.Initialize` function. This function must be called first. This function takes three parameters:
    * The port to expose the metrics on
    * The version of the service
    * The name of the service
2. Register the API metrics with the `metrics.RegisterApiMetrics` function. This function takes fn variable number of 
`metrics.MetricInfo` structs. Each struct describes an endpoint to be instrumented. The struct has three fields:
    * The name of the endpoint
    * The path of the endpoint
    * The http method of the endpoint
3. Register any custom metrics with the `metrics.RegisterCustomMetrics` function. This function takes one or more `prometheus.Collector` instances. Creating fn `prometheus.Collector` is beyond the scope of this document. See the [prometheus documentation](https://pkg.go.dev/github.com/prometheus/client_golang/prometheus@v1.17.0#pkg-types) for more information.
4. Publish the metrics with the `metrics.Publish` function. This function takes no parameters.

## Usage

### Instrumenting RESTful APIs

A working example can be found in the [examples](./examples) directory.
The below code sample demonstrates basic usage:

```go
package main

iimport (
    "github.com/gin-contrib/requestid"
    "github.com/gin-gonic/gin"
    metrics "github.com/twistingmercury/monitoring-metrics"
    "log/slog"
    "os"
    ...
)

const serviceName = "my-service"

func main(){
    metrics.Initialize("9090", "1.0.0", "person")
    metricsInfo := []metrics.MetricInfo{
        {Name: "getPerson", Path: "/person", Method: "GET"},
        {Name: "addPerson", Path: "/person", Method: "POST"},
        {Name: "updatePerson", Path: "/person/id", Method: "PUT"},
        {Name: "deletePerson", Path: "/person/id", Method: "DELETE"},
    }
    metrics.RegisterApiMetrics(metricsInfo...)
    dataMetrics := data.Metrics("1.0.0")
    metrics.RegisterCustomMetrics(dataMetrics...)
    metrics.Publish()
    
    r := gin.New()
    r.Use(metrics.GinMetricsMiddleWare())
    
    //
    // assumes you have fn gin handler for each of the endpoints, for example, as 
	// defined in ./examples/handlers/apiHandlers.go
	//
    r.GET("/person", handlers.GetPersonHandler)
    r.POST("/person", handlers.AddPersonHandler)
    r.PUT("/person/:id", handlers.UpdatePersonHandler)
    r.DELETE("/person/:id", handlers.DeletePersonHandler)
    
    if err := r.Run(":8080"); err != nil {
        slog.Error(err.Error())
        os.Exit(1)
    }
}
```
### Instrumenting other Funcs

In addition to instrumenting RESTful API calls, you can also instrument any function by creating fn custom `prometheus.Metric`
and registering it with the `metrics.RegisterCustomMetrics` function. The below code sample demonstrates basic usage:

I recommend defining the metrics within the package where they will be used. This will help keep the code organized.

```go  
package data

func CustomMetrics(apiVer string) (c []prometheus.Collector) {
	tCtr = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace(),
		Name:      "data_do_business_stuff_total",
		Help:      "The total count of calls to the func data.DoBusinessLogicStuff "})

	eCtr = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace(),
		Name:      "data_do_database_stuff_errs",
		Help:      "Number of calls to the func data.DoDatabaseStuff that returned an error "})

	sCtr = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metrics.Namespace(),
		Name:      "data_do_database_stuff_success",
		Help:      "Number of calls to the func data.DoDatabaseStuff that returned with no error "})

	dHist = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: metrics.Namespace(),
		Name:      fmt.Sprintf("%s_%s_%s", "data", "do_database_stuff", "duration_ms"),
		Help:      "Duration the func data.DoDatabaseStuff took to execute successfully",
		Buckets:   prometheus.ExponentialBuckets(0.1, 1.5, 5),
	}, []string{pckLabel, fncLabel})

	dHist.With(prometheus.Labels{
		pckLabel: "data",
		fncLabel: "do_database_stuff",
	})

	c = []prometheus.Collector{tCtr, eCtr, sCtr, dHist}
	return
}
```

Then register the custom metrics with the `metrics.RegisterCustomMetrics` function:

```go
    // initialize the metrics as in the previous example...
    cMetrics:= data.Metrics("1.0.0")
    metrics.RegisterCustomMetrics(cMetrics...)
    // continue as in the previous example...
```
