package main

import (
	// "database/sql"
	// "fmt"
	"log"

	// _ "github.com/denisenkom/go-mssqldb"

	"github.com/gin-gonic/gin"
	"github.com/twistingmercury/monitoring/health"
)

func main() {
	r := gin.Default()

	deps := []health.DependencyDescriptor{
		{
			Connection: "https://golang.org/",
			Name:       "Golang Site",
			Type:       "Website",
		},
		{
			Name:        "sql dB check",
			Type:        "database",
			HandlerFunc: checkMSSQL,
		},
	}

	r.GET("/healthcheck", health.Handler("example", deps...))
	if err := r.Run(); err != nil {
		log.Fatal(err)
	}
}

func checkMSSQL() (hsr health.StatusResult) {
	// connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d", "myServer", "user", "pwd", 1234)
	// db, err := sql.Open("mssql", connString)
	// if err != nil {
	// 	hsr.Status = healthcheck.HealthStatusCritical
	// 	hsr.Message = err.Error()
	// 	return
	// }
	// defer db.Close()

	// query, err := db.Prepare("SELECT 1;") // --> 'SELECT 1;' is the fastest query that can be returned from a working MSSQL database.
	// if err != nil {
	// 	hsr.Status = healthcheck.HealthStatusCritical
	// 	hsr.Message = err.Error()
	// 	return
	// }
	// defer query.Close()
	// r := query.QueryRow()
	// var ans int
	// err = r.Scan(&ans)
	// if err != nil {
	// 	hsr.Status = healthcheck.HealthStatusCritical
	// 	hsr.Message = err.Error()
	// 	return
	// }
	// if ans == 1 {
	// 	hsr.Status = healthcheck.HealthStatusOK
	// 	hsr.Message = "ok"
	// }

	hsr.Status = health.HealthStatusOK
	hsr.Message = "ok"
	return
}
