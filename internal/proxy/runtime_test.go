package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadyRequiresHealthyBackendInEveryPool(t *testing.T) {
	h, err := NewHandler(proxyConfig(), roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, nil }))
	if err != nil {
		t.Fatal(err)
	}
	if !h.Ready() {
		t.Fatal("new runtime should be ready")
	}
	h.pools["api"].backends[0].SetAlive(false)
	if h.Ready() {
		t.Fatal("runtime stayed ready with no healthy backend")
	}
}

func TestDynamicSwapsCompleteHandler(t *testing.T) {
	first, _ := NewHandler(proxyConfig(), staticResponse("first"))
	second, _ := NewHandler(proxyConfig(), staticResponse("second"))
	dynamic := NewDynamic(first)
	if body := serveDynamic(dynamic); body != "first" {
		t.Fatalf("body = %q", body)
	}
	dynamic.Swap(second)
	if body := serveDynamic(dynamic); body != "second" {
		t.Fatalf("body = %q", body)
	}
}

func TestInheritBackendHealthPreservesOnlyUnchangedBackend(t *testing.T) {
	previous, _ := NewHandler(proxyConfig(), staticResponse("old"))
	previous.pools["api"].backends[0].SetAlive(false)
	next, _ := NewHandler(proxyConfig(), staticResponse("new"))
	next.InheritBackendHealth(previous)
	if next.pools["api"].backends[0].Alive() {
		t.Fatal("unchanged backend lost its unhealthy state")
	}

	changedConfig := proxyConfig()
	pool := changedConfig.Pools["api"]
	pool.Backends[0].URL = "http://replacement.test"
	changedConfig.Pools["api"] = pool
	changed, _ := NewHandler(changedConfig, staticResponse("changed"))
	changed.InheritBackendHealth(previous)
	if !changed.pools["api"].backends[0].Alive() {
		t.Fatal("changed backend inherited stale health")
	}
}

func TestHealthClientDoesNotFollowRedirects(t *testing.T) {
	calls := 0
	client := newHealthClient(roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": {"http://other.test/health"}}, Body: http.NoBody}, nil
	}))
	response, err := client.Get("http://backend.test/health")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if calls != 1 || response.StatusCode != http.StatusFound {
		t.Fatalf("calls=%d status=%d", calls, response.StatusCode)
	}
}

func staticResponse(body string) roundTripFunc {
	return func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	}
}

func serveDynamic(handler http.Handler) string {
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "http://api.test/api", nil))
	return recorder.Body.String()
}
