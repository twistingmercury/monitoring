package main

import (
	"github.com/rs/zerolog"
	"log"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/twistingmercury/monitoring/logs"
)

const (
	apiVersion  = "1.0.0"
	serviceName = "examples"
	commit      = "123456"
	env         = "local"
)

func main() {
	// initialize the logging
	logs.Initialize(zerolog.InfoLevel, apiVersion, serviceName, time.Now().String(), commit, env, os.Stdout)

	// note: in production, you should use gin.ReleaseMode
	gin.SetMode(gin.DebugMode)

	// create a new gin r with no middleware; we'll add our own
	r := gin.New()

	r.Use(gin.Recovery(), logs.GinLoggingMiddleware())

	r.GET("/ping", pingHandler)

	if err := r.Run(":8080"); err != nil {
		slog.Error("failed to start the server", "error", err)
		os.Exit(5) // initialize the exit code
	}
}

var invocations = 0

func pingHandler(c *gin.Context) {
	if invocations == 1 {
		// slog.Log(c, logs.LevelFatal, "invocations reached", "invocations", invocations)
		log.Fatal("invocations reached: ", invocations)
	}

	time.Sleep(time.Duration(sleepTime(5, 100)) * time.Millisecond)
	c.JSON(200, gin.H{
		"message": "pong",
	})
}

func sleepTime(min, max int) int {
	invocations++
	slog.Debug("sleeping", "min", min, "max", max)
	return rand.Intn(max-min) + min
}
