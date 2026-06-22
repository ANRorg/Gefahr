//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anouar/goproxy/internal/app"
	"github.com/anouar/goproxy/internal/config"
	proxyhandler "github.com/anouar/goproxy/internal/proxy"
)

func TestRoutingBalancingAndCachingOverRealSockets(t *testing.T) {
	first, firstCalls := fixture("one")
	defer first.Close()
	second, secondCalls := fixture("two")
	defer second.Close()
	cfg := config.Default()
	cfg.Routes = []config.Route{{Name: "api", Host: "api.test", PathPrefix: "/", Pool: "api", Strategy: "round_robin", Cache: config.RouteCache{Enabled: true}}}
	cfg.Pools["api"] = config.Pool{
		Backends: []config.Backend{{Name: "one", URL: first.URL}, {Name: "two", URL: second.URL}},
		Health:   config.Health{Path: "/health", Interval: config.Duration(time.Second), Timeout: config.Duration(time.Second), HealthyThreshold: 1, UnhealthyThreshold: 1},
		Retry:    config.Retry{MaxAttempts: 1},
	}
	handler, err := proxyhandler.NewHandler(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	front := httptest.NewServer(handler)
	defer front.Close()

	if got := request(t, front.URL+"/first"); got != "one:/first" {
		t.Fatalf("first = %q", got)
	}
	if got := request(t, front.URL+"/second"); got != "two:/second" {
		t.Fatalf("second = %q", got)
	}
	before := firstCalls.Load() + secondCalls.Load()
	request(t, front.URL+"/cache")
	request(t, front.URL+"/cache")
	if delta := firstCalls.Load() + secondCalls.Load() - before; delta != 1 {
		t.Fatalf("cache upstream calls = %d", delta)
	}
}

func TestReloadPublishesValidConfigAndRetainsItAfterRejection(t *testing.T) {
	first, _ := fixture("one")
	defer first.Close()
	second, _ := fixture("two")
	defer second.Close()

	path := filepath.Join(t.TempDir(), "proxy.yaml")
	if err := os.WriteFile(path, []byte(proxyYAML(first.URL)), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := app.New(cfg, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	healthContext, stopHealth := context.WithCancel(context.Background())
	t.Cleanup(stopHealth)
	front := httptest.NewServer(runtime.Handler())
	defer front.Close()

	if got := request(t, front.URL+"/before"); got != "one:/before" {
		t.Fatalf("before reload = %q", got)
	}
	if err := os.WriteFile(path, []byte(proxyYAML(second.URL)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Reload(healthContext); err != nil {
		t.Fatal(err)
	}
	if got := request(t, front.URL+"/after"); got != "two:/after" {
		t.Fatalf("after reload = %q", got)
	}

	if err := os.WriteFile(path, []byte("unknown: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Reload(healthContext); err == nil {
		t.Fatal("invalid reload succeeded")
	}
	if got := request(t, front.URL+"/retained"); got != "two:/retained" {
		t.Fatalf("after rejected reload = %q", got)
	}
}

func TestSafeRequestRetriesRealTransportFailure(t *testing.T) {
	var failedCalls atomic.Int64
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		failedCalls.Add(1)
		connection, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Errorf("hijack: %v", err)
			return
		}
		connection.Close()
	}))
	defer failing.Close()
	healthy, healthyCalls := fixture("healthy")
	defer healthy.Close()

	cfg := proxyConfig(failing.URL, healthy.URL)
	cfg.Pools["api"] = config.Pool{
		Backends: cfg.Pools["api"].Backends,
		Health:   cfg.Pools["api"].Health,
		Retry:    config.Retry{MaxAttempts: 2},
	}
	handler, err := proxyhandler.NewHandler(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	front := httptest.NewServer(handler)
	defer front.Close()

	if got := request(t, front.URL+"/retry"); got != "healthy:/retry" {
		t.Fatalf("retry response = %q", got)
	}
	if failedCalls.Load() != 1 || healthyCalls.Load() != 1 {
		t.Fatalf("attempts = failed:%d healthy:%d", failedCalls.Load(), healthyCalls.Load())
	}
}

func fixture(name string) (*httptest.Server, *atomic.Int64) {
	calls := new(atomic.Int64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path == "/cache" {
			w.Header().Set("Cache-Control", "public, max-age=60")
		}
		fmt.Fprintf(w, "%s:%s", name, r.URL.Path)
	}))
	return server, calls
}

func proxyConfig(backendURLs ...string) config.Config {
	cfg := config.Default()
	cfg.Routes = []config.Route{{Name: "api", Host: "api.test", PathPrefix: "/", Pool: "api", Strategy: "round_robin"}}
	backends := make([]config.Backend, 0, len(backendURLs))
	for index, backendURL := range backendURLs {
		backends = append(backends, config.Backend{Name: fmt.Sprintf("backend-%d", index), URL: backendURL})
	}
	cfg.Pools["api"] = config.Pool{
		Backends: backends,
		Health:   config.Health{Path: "/health", Interval: config.Duration(time.Second), Timeout: config.Duration(time.Second), HealthyThreshold: 1, UnhealthyThreshold: 1},
		Retry:    config.Retry{MaxAttempts: 1},
	}
	return cfg
}

func proxyYAML(backendURL string) string {
	return fmt.Sprintf(`routes:
  - name: api
    host: api.test
    path_prefix: /
    pool: api
    strategy: round_robin
pools:
  api:
    backends:
      - name: backend
        url: %s
    health:
      path: /health
      interval: 1s
      timeout: 1s
      unhealthy_threshold: 1
      healthy_threshold: 1
    retry:
      max_attempts: 1
`, backendURL)
}

func request(t *testing.T, target string) string {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, target, nil)
	req.Host = "api.test"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.StatusCode, body)
	}
	return string(body)
}
