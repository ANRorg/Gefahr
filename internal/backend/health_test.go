package backend

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
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

func TestCheckerRunProbesImmediatelyAndStops(t *testing.T) {
	target, _ := url.Parse("http://backend.test")
	b := New("one", target)
	b.SetAlive(false)
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusNoContent, Body: http.NoBody}, nil
	})}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	changed := make(chan struct{})
	var closeOnce sync.Once
	checker := Checker{
		Backends: []*Backend{b},
		Client:   client,
		Policy:   HealthPolicy{Path: "/health", Interval: time.Hour, Timeout: time.Second, HealthyThreshold: 1, UnhealthyThreshold: 1},
		OnChange: func(*Backend, bool) {
			closeOnce.Do(func() { close(changed) })
		},
	}
	done := make(chan struct{})
	go func() {
		checker.Run(ctx)
		close(done)
	}()
	select {
	case <-changed:
	case <-time.After(time.Second):
		t.Fatal("checker did not probe immediately")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("checker did not stop")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
