package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandlerExposesRequestAndBackendMetrics(t *testing.T) {
	m := New()
	m.ObserveRequest("api", http.MethodGet, 200, 2, "hit", 25*time.Millisecond)
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
