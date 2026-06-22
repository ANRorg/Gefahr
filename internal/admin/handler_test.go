package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOperationalProbes(t *testing.T) {
	ready := false
	handler := NewHandler(func() bool { return true }, func() bool { return ready }, nil)
	assertStatus(t, handler, "/livez", http.StatusOK)
	assertStatus(t, handler, "/readyz", http.StatusServiceUnavailable)
	ready = true
	assertStatus(t, handler, "/readyz", http.StatusOK)
}

func TestMetricsIsNotExposedWhenHandlerIsNil(t *testing.T) {
	handler := NewHandler(func() bool { return true }, func() bool { return true }, nil)
	assertStatus(t, handler, "/metrics", http.StatusNotFound)
}

func assertStatus(t *testing.T, handler http.Handler, path string, want int) {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	if recorder.Code != want {
		t.Fatalf("%s status = %d; want %d", path, recorder.Code, want)
	}
}
