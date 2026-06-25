package admin

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestScopedCredentialsAuthorizeAdminEndpoints(t *testing.T) {
	handler := NewHandler(func() bool { return true }, func() bool { return true }, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), "", WithCredentials([]Credential{
		{Name: "health", Token: "health-secret", Scopes: []string{"health"}},
		{Name: "metrics", Token: "metrics-secret", Scopes: []string{"metrics"}},
		{Name: "reader", Token: "read-secret", Scopes: []string{"read"}},
		{Name: "operator", Token: "admin-secret", Scopes: []string{"admin"}},
	}))

	assertBearerStatus(t, handler, "/readyz", "health-secret", http.StatusOK)
	assertBearerStatus(t, handler, "/metrics", "health-secret", http.StatusForbidden)
	assertBearerStatus(t, handler, "/metrics", "metrics-secret", http.StatusNoContent)
	assertBearerStatus(t, handler, "/readyz", "read-secret", http.StatusOK)
	assertBearerStatus(t, handler, "/metrics", "read-secret", http.StatusNoContent)
	assertBearerStatus(t, handler, "/missing", "read-secret", http.StatusForbidden)
	assertBearerStatus(t, handler, "/missing", "admin-secret", http.StatusNotFound)
}

func TestAuditLoggerRecordsAdminAccessWithoutToken(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	handler := NewHandler(func() bool { return true }, func() bool { return true }, nil, "secret", WithAuditLogger(logger))

	unauthorized := httptest.NewRecorder()
	unauthorizedRequest := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	unauthorizedRequest.RemoteAddr = "192.0.2.10:1234"
	unauthorizedRequest.Header.Set("Authorization", "Bearer wrong-secret")
	handler.ServeHTTP(unauthorized, unauthorizedRequest)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d", unauthorized.Code)
	}

	authorized := httptest.NewRecorder()
	authorizedRequest := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	authorizedRequest.RemoteAddr = "192.0.2.10:1234"
	authorizedRequest.Header.Set("Authorization", "Bearer secret")
	handler.ServeHTTP(authorized, authorizedRequest)
	if authorized.Code != http.StatusOK {
		t.Fatalf("authorized status = %d", authorized.Code)
	}

	logs := output.String()
	for _, expected := range []string{`"msg":"admin request completed"`, `"path":"/readyz"`, `"remote_addr":"192.0.2.10"`, `"status":401`, `"auth":"unauthorized"`, `"status":200`, `"auth":"authorized"`} {
		if !strings.Contains(logs, expected) {
			t.Fatalf("audit logs missing %s: %s", expected, logs)
		}
	}
	if strings.Contains(logs, "wrong-secret") || strings.Contains(logs, "Bearer") {
		t.Fatalf("audit logs leaked authorization material: %s", logs)
	}
}

func TestAuditLoggerRecordsScopedPrincipalAndForbidden(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	handler := NewHandler(func() bool { return true }, func() bool { return true }, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), "", WithCredentials([]Credential{{Name: "health", Token: "health-secret", Scopes: []string{"health"}}}), WithAuditLogger(logger))

	assertBearerStatus(t, handler, "/metrics", "health-secret", http.StatusForbidden)

	logs := output.String()
	for _, expected := range []string{`"status":403`, `"auth":"forbidden"`, `"principal":"health"`} {
		if !strings.Contains(logs, expected) {
			t.Fatalf("audit logs missing %s: %s", expected, logs)
		}
	}
	if strings.Contains(logs, "health-secret") || strings.Contains(logs, "Bearer") {
		t.Fatalf("audit logs leaked authorization material: %s", logs)
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

func assertBearerStatus(t *testing.T, handler http.Handler, path, token string, want int) {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(recorder, request)
	if recorder.Code != want {
		t.Fatalf("%s with token %q status = %d; want %d", path, token, recorder.Code, want)
	}
}
