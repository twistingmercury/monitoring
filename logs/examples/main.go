package main

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

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
		log.Logger.Fatal().Err(err).Msg("failed to start server")
	}
}

var invocations = 0

func pingHandler(c *gin.Context) {
	if invocations == 1 {
		log.Error().Msg("invocations reached")
	}

	time.Sleep(time.Duration(sleepTime(5, 100)) * time.Millisecond)
	c.JSON(200, gin.H{
		"message": "pong",
	})
}

func sleepTime(min, max int) int {
	invocations++
	log.Debug().Int("invocations", invocations).Msg("sleepTime")
	return rand.Intn(max-min) + min
}
