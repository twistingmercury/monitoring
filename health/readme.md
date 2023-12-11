# go-healthcheck

This package is designed to meet the healthcheck requirements as defined in our "core principles" at my current employer, [MCG Health](https://www.mcg.com/).

## Usage

:information_source: This package uses the [Gin Web Framework](https://github.com/gin-gonic/gin).

The package provides two means of checking a dependency for a health check:

1. By providing a URL that can be called using an `http.Client`:
   ```Go
        descriptor := healthcheck.DependencyDescriptor{
            Connection: "https://golang.org/",
            Name:       "Golang Site", // <-- anything meaningful to you
            Type:       "Website",     // <-- 
        }
   ```
2. By providing assigning a func to `DependencyDescriptor.HandlerFunc` that will return a [healthcheck.HealthStatusResult](./dependencies.go):
   ```Go
        descriptor := healthcheck.DependencyDescriptor{
            Name:       "My custom dependency check", // <-- anything meaningful to you
            HandlerFunc: func() (hcs healthcheck.HealthStatusResult){ 
                // do whatever is needed...
                // set the return values
                hcs.Status = healthcheck.HealthStatusOK // | healthcheck.HealthStatusWarning | healthcheck.HealthStatusCritical
                hcs.Message = "OK" // <-- or whatever is appropriate, can be empty
                return 
            },
        }
   ```

For an example of using both, see [example.go](examples/example.go).  

Get the latest package: `go get -u github.com/twistingmercury/go-healthcheck`

