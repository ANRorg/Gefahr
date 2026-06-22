package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/anouar/goproxy/internal/config"
)

func TestHandlerExposesRequestAndBackendMetrics(t *testing.T) {
	m := New()
	m.ObserveRequest("request-id", "api", http.MethodGet, "/items", "one", 200, 2, "hit", 25*time.Millisecond)
	m.SetBackendHealth("api", "one", true)
	m.SetBackendActive("api", "one", 3)
	recorder := httptest.NewRecorder()
	m.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := recorder.Body.String()
	for _, expected := range []string{`goproxy_requests_total{method="GET",route="api",status="200"} 1`, `goproxy_retries_total{route="api"} 1`, `goproxy_backend_healthy{backend="one",pool="api"} 1`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("metrics missing %q", expected)
		}
	}
}

func TestObserveRequestBoundsAttackerControlledMethods(t *testing.T) {
	m := New()
	for _, method := range []string{"BREW", "CUSTOM", "X-UNIQUE"} {
		m.ObserveRequest("", "api", method, "", "", 200, 1, "bypass", time.Second)
	}
	if len(m.requests) != 1 || m.requests[requestKey{"api", "OTHER", 200}] != 3 {
		t.Fatalf("request series were not bounded: %#v", m.requests)
	}
}

func TestReconcileConfigBoundsLabelsAcrossReloads(t *testing.T) {
	m := New()
	cfg := config.Default()
	cfg.Routes = []config.Route{{Name: "current"}}
	cfg.Pools["pool"] = config.Pool{Backends: []config.Backend{{Name: "current"}}}
	m.ObserveRequest("", "old", http.MethodGet, "", "", 200, 1, "bypass", time.Second)
	m.SetBackendHealth("old", "old", true)
	m.ReconcileConfig(cfg)
	if len(m.requests) != 0 || len(m.health) != 0 {
		t.Fatalf("stale metrics survived: requests=%v health=%v", m.requests, m.health)
	}
	m.ObserveRequest("", "removed", http.MethodGet, "", "", 200, 1, "bypass", time.Second)
	m.SetBackendHealth("removed", "removed", true)
	if m.requests[requestKey{"retired", http.MethodGet, 200}] != 1 || len(m.health) != 0 {
		t.Fatalf("late labels were not bounded: requests=%v health=%v", m.requests, m.health)
	}
}
