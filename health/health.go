package health

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// DependencyDescriptor defines a resource to be checked during a heartbeat request.
type DependencyDescriptor struct {
	Name        string                    `json:"name"`
	Type        string                    `json:"type"`
	Connection  string                    `json:"connection"`
	HandlerFunc func() (hsr StatusResult) `json:"-"`
}

func (d *DependencyDescriptor) String() string {
	text, _ := json.MarshalIndent(d, "", "  ")
	return string(text)
}

// StatusResult represents another process or API that this service relies upon to be considered healthy.
type StatusResult struct {
	Status          HealthStatus `json:"status"`
	Name            string       `json:"name,omitempty"`
	Resource        string       `json:"resource"`
	RequestDuration float64      `json:"request_duration_ms"`
	StatusCode      int          `json:"http_status_code"`
	Message         string       `json:"message,omitempty"`
}

func (dep *StatusResult) String() string {
	text, _ := json.MarshalIndent(dep, "", "  ")
	return string(text)
}

// Response is the response to be returned to the caller.
type Response struct {
	Status          HealthStatus   `json:"status"`
	Name            string         `json:"name,omitempty"`
	Resource        string         `json:"resource"`
	Machine         string         `json:"machine,omitempty"`
	UtcDateTime     time.Time      `json:"utc_DateTime"`
	RequestDuration float64        `json:"request_duration_ms"`
	Message         string         `json:"message,omitempty"`
	Dependencies    []StatusResult `json:"dependencies,omitempty"`
}

func (h *Response) String() string {
	text, _ := json.Marshal(h)
	return string(text)
}

var (
	dependencies []DependencyDescriptor
)

// Handler returns the health of the app as a Response object.
func Handler(svcName string, deps ...DependencyDescriptor) gin.HandlerFunc {
	dependencies = deps
	return func(c *gin.Context) {
		st := time.Now()

		hb := Response{
			Resource:    svcName,
			UtcDateTime: time.Now().UTC(),
		}
		status, deps := checkDeps(dependencies)
		hb.Dependencies = deps
		hb.Status = status

		hb.RequestDuration = float64(time.Since(st).Microseconds()) / 1000

		c.JSON(http.StatusOK, hb)
	}
}

func checkDeps(deps []DependencyDescriptor) (status HealthStatus, hbl []StatusResult) {
	for _, desc := range deps {
		hsr := StatusResult{Status: HealthStatusOK}
		switch {
		case desc.HandlerFunc != nil:
			hsr = desc.HandlerFunc()
		default:
			hsr = checkURL(desc.Connection)
		}
		if hsr.Status > status {
			status = hsr.Status
		}
		hsr.Name = desc.Name
		hbl = append(hbl, hsr)
	}
	return
}

func checkURL(url string) StatusResult {
	hsr := StatusResult{
		Resource: url,
		Status:   HealthStatusNotSet,
	}

	st := time.Now()
	r, err := http.Get(url)
	elapsed := time.Since(st)
	hsr.RequestDuration = float64(elapsed.Microseconds()) / 1000
	if err != nil {
		hsr.Status = HealthStatusCritical
		return hsr
	}

	defer r.Body.Close()
	hsr.StatusCode = r.StatusCode

	switch {
	case elapsed > 3*time.Second && r.StatusCode >= 200 && r.StatusCode <= 299:
		hsr.Status = HealthStatusWarning
	case r.StatusCode >= 100 && r.StatusCode <= 299:
		hsr.Status = HealthStatusOK
		hsr.Message = "ok"
	case r.StatusCode >= 300 && r.StatusCode <= 399:
		hsr.Status = HealthStatusWarning
	case r.StatusCode >= 200 && r.StatusCode <= 299:
	default:
		hsr.Status = HealthStatusCritical
	}
	return hsr
}
