package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRegisterDependency(t *testing.T) {
	tests := []struct {
		dependency Dependency

		expectedErr    error
		expectedHealth bool
	}{
		// Passing - healthy
		{
			dependency: Dependency{
				Name:  "healthy service",
				Level: LevelHard,

				check: func() bool { return true },
			},

			expectedErr:    nil,
			expectedHealth: true,
		},

		// Passing - unhealthy
		{
			dependency: Dependency{
				Name:  "unhealthy service",
				Level: LevelHard,

				check: func() bool { return false },
			},

			expectedErr:    nil,
			expectedHealth: false,
		},
		// Passing - unhealthy soft
		{
			dependency: Dependency{
				Name:  "unhealthy service",
				Level: LevelSoft,

				check: func() bool { return false },
			},

			expectedErr:    nil,
			expectedHealth: true,
		},

		// Failing - no name
		{Dependency{}, ErrNoDependency, true},
	}

	for i, test := range tests {
		check, err := InitialiseServiceCheck("test", 50*time.Millisecond)
		if err != nil {
			t.Errorf("expected nil got %v", err)
		}

		err = check.RegisterDependency(test.dependency.Name, test.dependency.Level, test.dependency.check)
		if err != test.expectedErr {
			t.Errorf("expected %v got %v", test.expectedErr, err)
		}

		check.updateStatus()
		if check.IsHealthy() != test.expectedHealth {
			t.Errorf("expected %v got %v on test case #%d", test.expectedHealth, check.Healthy, i)
		}
	}
}

func TestInitialiseServiceCheck(t *testing.T) {
	check, err := InitialiseServiceCheck("", 50*time.Millisecond)
	if err == nil {
		t.Errorf("expecting %v got %v", ErrNoServiceNameSupplied, err)
	}
	if check != nil {
		t.Errorf("expected nil got %v", check)
	}
}

func TestDependency(t *testing.T) {
	tests := []struct {
		dependency  *Dependency
		expectedErr error
	}{
		// Passing
		{
			dependency: &Dependency{
				Name:  "test",
				Level: LevelHard,
				check: func() bool { return false },
			},
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		check := &ServiceCheck{}
		if err := check.RegisterDependency(test.dependency.Name,
			test.dependency.Level, test.dependency.check); err != nil {
			t.Errorf("expected nil got %v", err)
		}

		dep, err := check.Dependency(test.dependency.Name)
		if err != nil {
			t.Errorf("expected %v got %v", test.expectedErr, err)
		}

		if dep.Name != test.dependency.Name {
			t.Errorf("expected %v got %v", test.dependency.Name, dep.Name)
		}
		if dep.Level != test.dependency.Level {
			t.Errorf("expected %v got %v", test.dependency.Level, dep.Level)
		}

	}
}

func TestEndpointHelper(t *testing.T) {
	tests := []struct {
		status      int
		expected    bool
		expectedErr error
	}{
		// Passing
		{http.StatusOK, true, nil},
		{http.StatusInternalServerError, false, nil},
	}

	for _, test := range tests {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(test.status)
		}))

		resp, err := Check200Helper(server.URL)
		if err != test.expectedErr {
			t.Errorf("expected %v got %v", test.expectedErr, err)
		}
		if resp != test.expected {
			t.Errorf("expected %v got %v", test.expected, resp)
		}

	}
}

func TestHTTPHandler(t *testing.T) {
	healthCheck := &ServiceCheck{
		Name:     "test",
		Healthy:  true,
		duration: 0,
	}

	ts := httptest.NewServer(http.HandlerFunc(healthCheck.HTTPHandler))
	defer ts.Close()

	res, err := http.Get(ts.URL)
	if err != nil {
		t.Error(err)
	}

	if res.StatusCode != 200 {
		t.Errorf("Unexpected status code returned, expected %d, found %d", 200, res.StatusCode)
	}

	healthCheck.Healthy = false

	res, err = http.Get(ts.URL)
	if err != nil {
		t.Error(err)
	}

	if res.StatusCode != 503 {
		t.Errorf("Unexpected status code returned, expected %d, found %d", 503, res.StatusCode)
	}
}

func TestGet(t *testing.T) {
	tests := []struct {
		healthy  bool
		expected bool
	}{
		{true, true},
		{false, false},
	}

	for _, test := range tests {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(&ServiceCheck{
				Name:    "test",
				Healthy: test.healthy,
			})
		}))

		healthy, err := Get(server.URL)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if healthy != test.expected {
			t.Errorf("expected %v, got %v", test.expected, healthy)
		}
	}
}

func TestWaitForDependencies(t *testing.T) {
	t.Run("healthy", func(t *testing.T) {
		healthCheck := &ServiceCheck{
			Name:     "test",
			Healthy:  true,
			duration: 0,
		}

		returned := make(chan struct{})

		healthCheck.RegisterDependency("redis", LevelHard, func() bool {
			return true
		})

		go func() {
			healthCheck.WaitForDependencies(10 * time.Second)
			returned <- struct{}{}
		}()
		select {
		case <-returned:
			break
		case <-time.After(time.Second * 2):
			t.Error("Timeout even though the health check passes")
		}
	})

	t.Run("unhealthy", func(t *testing.T) {
		healthCheck := &ServiceCheck{
			Name:     "test",
			Healthy:  true,
			duration: 0,
		}

		returned := make(chan struct{})

		healthCheck.RegisterDependency("redis", LevelHard, func() bool {
			return false
		})

		go func() {
			healthCheck.WaitForDependencies(2 * time.Second)
			returned <- struct{}{}
		}()
		select {
		case <-returned:
			break
		case <-time.After(time.Second * 3):
			t.Error("Context wasn't cancelled")
		}
	})
}
