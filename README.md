# health

## tl;dr

Health checks for the services

## Description

Package `health` allows a health check to be quickly enabled for a service. It
allows the service to monitor any dependents and based upon the health of those,
report it's own health.

## Installation

`campaigns` is a [gb style project](https://github.com/constabulary/gb) and can be
imported by running
```
gb vendor fetch github.com/connectedventures/f8-pkg/health
```

Once imported into your project start referencing it by adding the line
```go
import "github.com/connectedventures/f8-pkg/health"
```

## Examples

```go
// Initialise a new health check, with service name and health check duration
healthCheck, err := health.InitialiseServiceCheck(serviceName, 10*time.Second)
if err != nil {
	logger.Crit(err)
}

// register any dependents
// dependentCheck should be a function which takes no parameters and returns
// a bool indicating the health of the dependent
healthCheck.RegisterDependent(dependentName, dependentCheck)

// Register the health check with an external endpoint
http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
	healthCheck.WriteStatus(w)
})
```

## Tests

In the package root directory run `go test` to run all tests.


## Help

Package maintained by @augier

`#f8-backend` is your best bet for support.
