package health

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Level is used to outline an acceptable level of dependency failure
type Level uint32

const (
	// LevelSoft defines a soft dependency, one that the service can continue
	// functionality if it's down
	LevelSoft Level = 0
	// LevelHard defines a hard dependency, one that's crucial to the service
	LevelHard Level = 1
)

var (
	// HTTPClient is used to make requests, it comes with sensible, pre-defined
	// timeouts.
	HTTPClient = &http.Client{
		Timeout:   500 * time.Millisecond,
		Transport: http.DefaultTransport,
	}
)

// ServiceCheck is the main struct in the package. Use InitialiseHealthCheck to
// instantiate one
type ServiceCheck struct {
	Name         string        `json:"name"`
	Healthy      bool          `json:"healthy"`
	Dependencies []*Dependency `json:"dependencies"`

	duration time.Duration
	mu       sync.RWMutex
}

// Dependency defines a dependency and it's status
type Dependency struct {
	Name    string `json:"name"`
	Healthy bool   `json:"healthy"`
	Level   Level  `json:"level"`

	check func() bool
}

// Check200Helper is a helper for checking a service's health endpoint.
func Check200Helper(rawURL string) (bool, error) {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return false, err
	}

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return false, err
	}

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	return true, nil
}

// InitialiseServiceCheck returns an initialised check for the service `name`.
// It's dependencies will be polled every `duration`.
//
// Since v2.0.0 the user is required to start the check themselves by calling
// StartCheck once all dependancies are registered
func InitialiseServiceCheck(name string, duration time.Duration) (*ServiceCheck, error) {
	if name == "" {
		return nil, ErrNoServiceNameSupplied
	}

	check := &ServiceCheck{
		Name:     name,
		duration: duration,
	}

	return check, nil
}

// WaitForDependencies blocks current thread until all dependencies are healthy
// if it takes longer than `timeout` to ensure that all dependencies are
// healthy it will return false
func (s *ServiceCheck) WaitForDependencies(timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	go func() {
		for {
			s.updateStatus()
			if s.getHealth() {
				cancel()
				break
			}
			<-time.After(1 * time.Second)
		}
	}()
	<-ctx.Done()
	return s.getHealth()
}

// StartCheck will start checking the dependencies
func (s *ServiceCheck) StartCheck() {
	go func() {
		for {
			s.updateStatus()
			<-time.After(s.duration)
		}
	}()
}

// RegisterDependency registers a new dependency on the service. It checks that
// dependency isn't a duplicate, performs an initial health check, and adds it
// to be continually checked.
func (s *ServiceCheck) RegisterDependency(name string, level Level, check func() bool) error {
	if name == "" {
		return ErrNoDependency
	}

	for _, dependency := range s.Dependencies {
		if dependency.Name == name {
			return ErrDependencyAlreadyRegistered
		}
	}

	dep := &Dependency{
		Name:    name,
		Level:   level,
		Healthy: check(),

		check: check,
	}

	s.mu.Lock()
	s.Dependencies = append(s.Dependencies, dep)
	s.mu.Unlock()
	return nil
}

// Dependency finds and returns the named dependency
func (s *ServiceCheck) Dependency(name string) (*Dependency, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, dependency := range s.Dependencies {
		if dependency.Name == name {
			return dependency, nil
		}
	}

	return nil, ErrNoDependency
}

func (s *ServiceCheck) getHealth() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Healthy
}

func (s *ServiceCheck) updateStatus() {
	s.mu.Lock()
	defer s.mu.Unlock()
	// loop through and change to unhealthy if any dependents are unhealthy
	for _, dependency := range s.Dependencies {
		dependency.Healthy = dependency.check()

		if !dependency.Healthy && dependency.Level == LevelHard {
			s.Healthy = false
			return
		}
	}

	s.Healthy = true
}

// WriteStatus writes the status to any io.Writer
func (s *ServiceCheck) WriteStatus(w io.Writer) error {
	return json.NewEncoder(w).Encode(s)
}

// HTTPHandler outputs the status with the relevant response code to a ResponseWriter
func (s *ServiceCheck) HTTPHandler(w http.ResponseWriter, r *http.Request) {
	if s.getHealth() {
		w.WriteHeader(200)
	} else {
		w.WriteHeader(503)
	}

	s.WriteStatus(w)
}

// IsHealthy returns a bool whether this ServiceCheck is healthy
func (s *ServiceCheck) IsHealthy() bool {
	return s.getHealth()
}

// Get is a wrapper which checks whether the URL is healthy
func Get(url string) (bool, error) {
	var (
		response ServiceCheck
	)

	r, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}

	resp, err := HTTPClient.Do(r)
	if err != nil {
		return false, err
	}

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return false, err
	}

	return response.Healthy, nil
}

// Errors
var (
	ErrNoServiceNameSupplied       = errors.New("no service name supplied")
	ErrDependencyAlreadyRegistered = errors.New("dependent already registered")
	ErrNoDependency                = errors.New("no dependency registered")
)
