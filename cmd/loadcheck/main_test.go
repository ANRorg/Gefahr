package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestScrapeSendsBearerToken(t *testing.T) {
	t.Setenv("GOPROXY_ADMIN_TOKEN", "secret")
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if got := request.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("Authorization = %q", got)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("go_goroutines 1\n")), Header: make(http.Header)}, nil
	})}
	values, err := scrape(client, "http://127.0.0.1:9090/metrics")
	if err != nil {
		t.Fatal(err)
	}
	if values["go_goroutines"] != 1 {
		t.Fatalf("values = %v", values)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
