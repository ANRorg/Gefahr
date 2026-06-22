package proxy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anouar/goproxy/internal/config"
)

func proxyConfig() config.Config {
	cfg := validProxyConfig()
	return cfg
}

func validProxyConfig() config.Config {
	cfg := config.Default()
	cfg.Routes = []config.Route{{Name: "api", Host: "api.test", PathPrefix: "/api", Pool: "api", Strategy: "round_robin"}}
	cfg.Pools["api"] = config.Pool{
		Backends: []config.Backend{{Name: "one", URL: "http://backend.test/base"}},
		Health:   config.Health{Path: "/health", Interval: config.Duration(1), Timeout: config.Duration(1), HealthyThreshold: 1, UnhealthyThreshold: 1},
		Retry:    config.Retry{MaxAttempts: 1},
	}
	return cfg
}

func TestHandlerForwardsMatchedRequest(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "backend.test" || r.URL.Path != "/base/api/users" {
			t.Fatalf("upstream URL = %s", r.URL)
		}
		if r.Host != "api.test" {
			t.Fatalf("upstream Host = %q", r.Host)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("proxied"))}, nil
	})
	h, err := NewHandler(proxyConfig(), transport)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://api.test/api/users", nil)
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK || recorder.Body.String() != "proxied" {
		t.Fatalf("response = %d %q", recorder.Code, recorder.Body.String())
	}
}

func TestHandlerReturnsNotFoundForUnmatchedRequest(t *testing.T) {
	h, err := NewHandler(proxyConfig(), roundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("transport called"); return nil, nil }))
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "http://other.test/api", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestHandlerReplacesUntrustedForwardingHeaders(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("X-Forwarded-For"); got != "192.0.2.10" {
			t.Fatalf("X-Forwarded-For = %q", got)
		}
		if got := r.Header.Get("X-Forwarded-Host"); got != "api.test" {
			t.Fatalf("X-Forwarded-Host = %q", got)
		}
		if got := r.Header.Get("X-Forwarded-Proto"); got != "http" {
			t.Fatalf("X-Forwarded-Proto = %q", got)
		}
		if got := r.Header.Get("Forwarded"); got != `for="192.0.2.10";host="api.test";proto=http` {
			t.Fatalf("Forwarded = %q", got)
		}
		return &http.Response{StatusCode: http.StatusNoContent, Header: make(http.Header), Body: http.NoBody}, nil
	})
	h, err := NewHandler(proxyConfig(), transport)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "http://api.test/api", nil)
	req.RemoteAddr = "192.0.2.10:4321"
	req.Header.Set("X-Forwarded-For", "attacker.test")
	req.Header.Set("Forwarded", "for=attacker.test")
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestHandlerRejectsDeclaredOversizedBody(t *testing.T) {
	cfg := proxyConfig()
	cfg.Limits.MaxBodyBytes = 4
	h, err := NewHandler(cfg, roundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("transport called"); return nil, nil }))
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://api.test/api", strings.NewReader("oversized"))
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestHandlerMapsUpstreamTimeoutWithoutLeakingError(t *testing.T) {
	h, err := NewHandler(proxyConfig(), roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, context.DeadlineExceeded }))
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "http://api.test/api", nil))
	if recorder.Code != http.StatusGatewayTimeout {
		t.Fatalf("status = %d", recorder.Code)
	}
	if recorder.Body.String() != "{\"code\":\"upstream_timeout\",\"message\":\"upstream timed out\"}\n" {
		t.Fatalf("body = %q", recorder.Body.String())
	}
}

func TestHandlerRetriesSafeTransportFailure(t *testing.T) {
	cfg := proxyConfig()
	cfg.Pools["api"] = config.Pool{
		Backends: []config.Backend{{Name: "one", URL: "http://one.test"}, {Name: "two", URL: "http://two.test"}},
		Health:   config.Health{Path: "/health", Interval: config.Duration(1), Timeout: config.Duration(1), HealthyThreshold: 1, UnhealthyThreshold: 1},
		Retry:    config.Retry{MaxAttempts: 2},
	}
	calls := 0
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return nil, context.DeadlineExceeded
		}
		if r.URL.Host != "two.test" {
			t.Fatalf("retry host = %q", r.URL.Host)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("retried"))}, nil
	})
	h, err := NewHandler(cfg, transport)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "http://api.test/api", nil))
	if recorder.Code != http.StatusOK || recorder.Body.String() != "retried" || calls != 2 {
		t.Fatalf("response = %d %q, calls=%d", recorder.Code, recorder.Body.String(), calls)
	}
}

func TestHandlerServesEligibleResponseFromCache(t *testing.T) {
	cfg := proxyConfig()
	cfg.Routes[0].Cache.Enabled = true
	calls := 0
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Cache-Control": {"public, max-age=60"}}, Body: io.NopCloser(strings.NewReader("cached"))}, nil
	})
	h, err := NewHandler(cfg, transport)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		recorder := httptest.NewRecorder()
		h.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "http://api.test/api/items?q=one", nil))
		if recorder.Code != http.StatusOK || recorder.Body.String() != "cached" {
			t.Fatalf("response %d = %d %q", i, recorder.Code, recorder.Body.String())
		}
	}
	if calls != 1 {
		t.Fatalf("upstream calls = %d", calls)
	}
}

func TestClientBodyErrorDoesNotEjectBackend(t *testing.T) {
	calls := 0
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return nil, &http.MaxBytesError{Limit: 10}
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("healthy"))}, nil
	})
	h, err := NewHandler(proxyConfig(), transport)
	if err != nil {
		t.Fatal(err)
	}
	first := httptest.NewRecorder()
	h.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "http://api.test/api", nil))
	if first.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("first status = %d", first.Code)
	}
	second := httptest.NewRecorder()
	h.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "http://api.test/api", nil))
	if second.Code != http.StatusOK || second.Body.String() != "healthy" {
		t.Fatalf("second response = %d %q", second.Code, second.Body.String())
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
