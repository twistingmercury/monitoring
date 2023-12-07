package handlers

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/twistingmercury/monitoring/metrics"
)

func MetricInfos() []metrics.MetricInfo {
	return []metrics.MetricInfo{
		{Name: "ping_handler", Path: "/ping", Method: http.MethodGet},
		{Name: "pong_handler", Path: "/pong", Method: http.MethodGet},
	}
}

func PingHandler(c *gin.Context) {
	time.Sleep(time.Duration(sleepTime(5, 250)) * time.Millisecond)
	c.JSON(200, gin.H{
		"message": "pong",
	})
}

func PongHandler(c *gin.Context) {
	time.Sleep(time.Duration(sleepTime(10, 500)) * time.Millisecond)
	c.JSON(200, gin.H{
		"message": "ping",
	})
}

func sleepTime(min, max int) int {
	return rand.Intn(max-min) + min
}
