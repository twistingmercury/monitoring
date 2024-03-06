package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/twistingmercury/monitoring/metrics/examples/data"
)

// func MetricsInfo() []metrics.MetricInfo {
// 	return []metrics.MetricInfo{
// 		{Name: "getPerson", Path: "/person", Method: "GET"},
// 		{Name: "addPerson", Path: "/person", Method: "POST"},
// 		{Name: "updatePerson", Path: "/person/id", Method: "PUT"},
// 		{Name: "deletePerson", Path: "/person/id", Method: "DELETE"},
// 	}
// }

func GetPersonHandler(c *gin.Context) {
	err := data.DoDatabaseStuff()
	if !handleErr(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"name": "John"})
}

func AddPersonHandler(c *gin.Context) {
	err := data.DoDatabaseStuff()
	if !handleErr(c, err) {
		return
	}
	c.Writer.WriteHeader(http.StatusAccepted)
}

func UpdatePersonHandler(c *gin.Context) {
	err := data.DoDatabaseStuff()
	if !handleErr(c, err) {
		return
	}
	c.Writer.WriteHeader(http.StatusNoContent)
}

func DeletePersonHandler(c *gin.Context) {
	err := data.DoDatabaseStuff()
	if !handleErr(c, err) {
		return
	}
	c.Writer.WriteHeader(http.StatusNoContent)
}

func handleErr(c *gin.Context, err error) (ok bool) {
	ok = true
	if err != nil {
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		ok = false
	}
	return
}
