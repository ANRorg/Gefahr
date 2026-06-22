//go:build integration

package integration_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

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
