# health [![Godoc](https://img.shields.io/badge/godoc-reference-5272B4.svg?style=round)](https://godoc.org/github.com/fresh8/health/)
---
A package to help your service check the health of others. (For when your load balancers can't.)

## Requirements

Due to the use of the `context` package, `health` is only compatible with go 1.7+

## Installation
Health is vendorable, use your favourite tools to vendor the package via the import path `github.com/fresh8/health`, example:
```bash
go get -u github.com/fresh8/health
```

## Usage

`health.LevelHard` references a hard dependency. This describes a dependency in which the service requires, and will become unhealthy if one of its hard dependencies is down.

`health.LevelSoft` references a soft dependency. This describes a dependency in which the service can run without, remaining healthy.

#### Initialise your healthcheck:
```go
check, err := health.InitialiseServiceCheck("name", 5 * time.Second)
```

#### Register some dependencies:
```go
// Register a hard dependency
check.RegisterDependency("google", health.LevelHard, func() bool {
	resp, err := http.Get("http://google.com")
	if err != nil {
		return false
	}

	return resp.StatusCode == http.StatusOK
})
```

#### Start the check
```go
check.StartCheck()
```

#### Serve the healthchecks
```go
router.Handle("/health", health.HTTPHandler)
```

#### Check a dependency's health
```go
dep, err := check.Dependency("google")
if err != nil {
	// Handle errors
	...
}

if !dep.Health  {
	// Enter a degraded state
	...
}
```
