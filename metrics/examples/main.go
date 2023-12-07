package main

import (
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"github.com/twistingmercury/monitoring/metrics"
	"github.com/twistingmercury/monitoring/metrics/examples/data"
	"github.com/twistingmercury/monitoring/metrics/examples/handlers"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	gin.SetMode(gin.DebugMode)

	// Metrics are hosted on in a separate goroutine on the port specified.
	// This needs to be invoked before any metrics are registered. It can be called
	// multiple times, but only the first call will have any effect.
	metrics.Initialize("9090", "example")

	// Register the metrics from any packages that have them, in this example,
	// he data package has metrics.
	dataMetrics := data.Metrics()

	// this can be called multiple times if there are metrics in multiple packages.
	metrics.RegisterCustomMetrics(dataMetrics...)

	// Publish exposes the metrics for scraping. This needs to be called after
	// all metrics have been registered. It can be called multiple times, but
	// only the first call will have any effect.
	metrics.Publish()

	// Create a gin router and add the middleware to it as one normally would.
	r := gin.New()
	r.Use(metrics.GinMetricsMiddleWare())

	r.GET("/person", handlers.GetPersonHandler)
	r.POST("/person", handlers.AddPersonHandler)
	r.PUT("/person/:id", handlers.UpdatePersonHandler)
	r.DELETE("/person/:id", handlers.DeletePersonHandler)

	if err := r.Run(":8080"); err != nil {
		logger.Fatal().Err(err).Msg("failed to start server")
	}
}
