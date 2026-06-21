package backend

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestRecordProbeUsesThresholds(t *testing.T) {
	b := New("one", &url.URL{Scheme: "http", Host: "example.test"})
	b.RecordProbe(false, 2, 2)
	if !b.Alive() {
		t.Fatal("backend changed after one failure")
	}
	b.RecordProbe(false, 2, 2)
	if b.Alive() {
		t.Fatal("backend stayed alive after threshold")
	}
	b.RecordProbe(true, 2, 2)
	if b.Alive() {
		t.Fatal("backend recovered too early")
	}
	b.RecordProbe(true, 2, 2)
	if !b.Alive() {
		t.Fatal("backend did not recover")
	}
}

func TestCheckerMarksFailedEndpointDead(t *testing.T) {
	target, _ := url.Parse("http://backend.test")
	b := New("one", target)
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusServiceUnavailable, Body: io.NopCloser(strings.NewReader("unavailable"))}, nil
	})}
	checker := Checker{Backends: []*Backend{b}, Client: client, Policy: HealthPolicy{Path: "/health", Timeout: time.Second, HealthyThreshold: 1, UnhealthyThreshold: 1}}
	checker.CheckOnce(context.Background())
	if b.Alive() {
		t.Fatal("failed backend remained alive")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
