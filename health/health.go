package health

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// DependencyDescriptor defines a resource to be checked during a heartbeat request.
type DependencyDescriptor struct {
	Name        string                          `json:"name"`
	Type        string                          `json:"type"`
	Connection  string                          `json:"connection"`
	HandlerFunc func() (hsr HealthStatusResult) `json:"-"`
}

func (d *DependencyDescriptor) String() string {
	text, _ := json.MarshalIndent(d, "", "  ")
	return string(text)
}

// HealthStatusResult represents another process or API that this service relies upon to be considered healthy.
type HealthStatusResult struct {
	Status          HealthStatus `json:"status"`
	Name            string       `json:"name,omitempty"`
	Resource        string       `json:"resource"`
	RequestDuration float64      `json:"request_duration_ms"`
	StatusCode      int          `json:"http_status_code"`
	Message         string       `json:"message,omitempty"`
}

func (dep *HealthStatusResult) String() string {
	text, _ := json.MarshalIndent(dep, "", "  ")
	return string(text)
}

// HealthResponse is the response to be returned to the caller.
type HealthResponse struct {
	Status          HealthStatus         `json:"status"`
	Name            string               `json:"name,omitempty"`
	Resource        string               `json:"resource"`
	Machine         string               `json:"machine,omitempty"`
	UtcDateTime     time.Time            `json:"utc_DateTime"`
	RequestDuration float64              `json:"request_duration_ms"`
	Message         string               `json:"message,omitempty"`
	Dependencies    []HealthStatusResult `json:"dependencies,omitempty"`
}

func (h *HealthResponse) String() string {
	text, _ := json.Marshal(h)
	return string(text)
}

var (
	dependencies []DependencyDescriptor
)

// Handler returns the health of the app as a HealthResponse object.
func Handler(svcName string, deps ...DependencyDescriptor) gin.HandlerFunc {
	dependencies = deps
	return func(c *gin.Context) {
		st := time.Now()

		hb := HealthResponse{
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

func checkDeps(deps []DependencyDescriptor) (status HealthStatus, hbl []HealthStatusResult) {
	for _, desc := range deps {
		hsr := HealthStatusResult{Status: HealthStatusOK}
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

func checkURL(url string) HealthStatusResult {
	hsr := HealthStatusResult{
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
	case elapsed > time.Duration(3*time.Second) && r.StatusCode >= 200 && r.StatusCode <= 299:
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
