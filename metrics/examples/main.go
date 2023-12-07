package main

import (
	"github.com/gin-gonic/gin"
	metrics "github.com/twistingmercury/monitoring-metrics"
	"github.com/twistingmercury/monitoring-metrics/examples/data"
	"github.com/twistingmercury/monitoring-metrics/examples/handlers"
	"log/slog"
	"os"
)

func main() {
	opt := &slog.HandlerOptions{
		AddSource: false,
		Level:     slog.LevelDebug,
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, opt))
	slog.SetDefault(log)

	gin.SetMode(gin.DebugMode)

	metrics.Initialize("9090")
	dataMetrics := data.Metrics()
	metrics.RegisterCustomMetrics(dataMetrics...)
	metrics.Publish()

	r := gin.New()
	r.Use(metrics.GinMetricsMiddleWare())

	r.GET("/person", handlers.GetPersonHandler)
	r.POST("/person", handlers.AddPersonHandler)
	r.PUT("/person/:id", handlers.UpdatePersonHandler)
	r.DELETE("/person/:id", handlers.DeletePersonHandler)

	if err := r.Run(":8080"); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
