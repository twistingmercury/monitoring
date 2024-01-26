#  Monitoring: Logging

This repository contains a middleware for [gin and gonic](https://github.com/gin-gonic/gin) using [zerolog](https://pkg.go.dev/github.com/rs/zerolog).
It is intended to be used by services that are instrumented with the [Open Telemetry]("go.opentelemetry.io/otel/trace") framework for Go.
This middleware will inject the trace id and span id into the log entry so that the traces and logs can be correlated. 

This package will also work as middleware if the project doesn't implement OTel distributed tracing. Neither is gin and gonic
required; you can initialize the logging package and use the helper functions to log manually.

## Installation

```bash

go get -u github.com/twistingmercury/monitoring

```

:eyes: Zerolog is currently used for logging since there isn't yet an [OTel logging implementation for Go](https://opentelemetry.io/docs/instrumentation/go/#status-and-releases).


## Log Collectors and Agents

This has been tested using [Vector](https://vector.dev/) and the [Datadog Agent](https://docs.datadoghq.com/agent/), with the destination being [Datadog](https://www.datadoghq.com/).
If the destination is a different provider, you may need to change the values for the constant defs `TraceIDAttr` and `SpanIDAttr` in the [middleware](./logger.go) file.

## Initialization

### io.Writer
Typically you will write to `os.Stdout`. However, if not containerizing, any [io.Writer](https://pkg.go.dev/io#Writer) will work. This can be useful if you want to test the logging.

```go
// if testing, you can use a bytes.Buffer
buffer := &bytes.Buffer{}
logs.Initialize(zerolog.DebugLevel, buildVersion, serviceName, buildDate, buildCommit, env, buffer)


// if not testing, you can use os.Stdout
logs.Initialize(zerolog.DebugLevel, buildVersion, serviceName, buildDate, buildCommit, env, os.Stdout)
```

This value is passed in with the `writer` parameter. If you do not provide a value, the application will panic. This is in keeping with the "fail fast" philosophy.

### Logging Level

The logging level is set using the [zerolog.Level](https://github.com/rs/zerolog/blob/master/log.go#L129) type. This value is passed in with the `level` parameter. If a value is provide 
that is not valid, the application will panic. Again, this is in keeping with the "fail fast" philosophy.

## Usage

To use the wrappers, you will need to initialize each wrapper you intend to use:

```go
package main

import (
    "github.com/gin-contrib/requestid"
    "github.com/gin-gonic/gin"
    "github.com/rs/zerolog"
    "os"

    "github.com/twistingmercury/monitoring/logs"
    ...
)

const serviceName = "my-service"

var ( // build info will be set during the build process
    buildDate    = "{not set}"
    buildVersion = "{not set}"
    buildCommit  = "{not set}"
	env           =  os.Getenv("ENVIRONMENT") // or however you want to set this
)

func main(){
	logs.Initialize(zerolog.DebugLevel, buildVersion, serviceName, buildDate, buildCommit, env, os.Stdout)

	// do stuff...start your service, etc.
	r := gin.New()
	r.Use(logs.GinLoggingMiddleware())
	r.GET("/api/v1/ready", func(c *gin.Context) {
		c.JSON(200, gin.H{"ready": true})
	})

	if r.Run(":8080");err != nil {
		log.Fatal()Err(err).Msg( "error encountered in the gin.Engine.run func")
	}
}
```

### Manual Logging

Log entries can be added manually that are correlated with the request. Helper funcs are provided for the various log levels. You must provide the context.Context that contains the trace information, and the message to log. A way to do this is might be:

```go
package logic

import (
	"context"
	"time"
)

func foo(ctx context.Context, args ...interface{}) (err error) {
	logs.Debug(ctx, "starting BusinessLogic", "args", args)
	s := time.Now()
	
	// do stuff...

	if err != nil {
		return
	}
	
	l := time.Since(s)
	logs.Debug(ctx, "finished BusinessLogic", "duration", l)   
	// do more stuff...
	return
}
```

...And in the gin handler:

```go
// myApi is a gin handler that is already logged with the gin middleware
func myApi (c *gin.Context){
	// do stuff...
    err := logic.Foo(c.Request.Context(), args...)
    if err != nil {
        // the middleware will log the error, so you don't have to
        c.JSON(500, gin.H{"error": "something went wrong"})
        return
    }
    // do more stuff...
    c.JSON(200, gin.H{"success": true})
}
```
## Access the Logger

You can get a pointer to the logger by calling `logs.Logger()`. This is useful if you want to log something outside of the middleware or helper functions.

```go

zlogger := logs.Logger()
zlogger.Info().Msg("this is a log entry")

```
