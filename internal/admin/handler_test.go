package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOperationalProbes(t *testing.T) {
	ready := false
	handler := NewHandler(func() bool { return true }, func() bool { return ready }, nil, "")
	assertStatus(t, handler, "/livez", http.StatusOK)
	assertStatus(t, handler, "/readyz", http.StatusServiceUnavailable)
	ready = true
	assertStatus(t, handler, "/readyz", http.StatusOK)
}

func TestMetricsIsNotExposedWhenHandlerIsNil(t *testing.T) {
	handler := NewHandler(func() bool { return true }, func() bool { return true }, nil, "")
	assertStatus(t, handler, "/metrics", http.StatusNotFound)
}

func TestBearerTokenProtectsOperationalEndpoints(t *testing.T) {
	handler := NewHandler(func() bool { return true }, func() bool { return true }, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), "secret")
	assertStatus(t, handler, "/readyz", http.StatusUnauthorized)
	assertStatus(t, handler, "/metrics", http.StatusUnauthorized)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	request.Header.Set("Authorization", "Bearer secret")
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("authenticated status = %d; want %d", recorder.Code, http.StatusOK)
	}
}

func assertStatus(t *testing.T, handler http.Handler, path string, want int) {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	if recorder.Code != want {
		t.Fatalf("%s status = %d; want %d", path, recorder.Code, want)
	}
}
