package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFixtureHandlerEndpoints(t *testing.T) {
	handler := newHandler("fixture")
	tests := []struct {
		path       string
		status     int
		body       string
		headerName string
		header     string
	}{
		{path: "/health", status: http.StatusOK},
		{path: "/cache", status: http.StatusOK, body: "fixture\n", headerName: "Cache-Control", header: "public, max-age=30"},
		{path: "/cookie", status: http.StatusOK, body: "fixture\n", headerName: "Set-Cookie", header: "session=fixture"},
		{path: "/chunked", status: http.StatusOK, body: "0:fixture\n1:fixture\n2:fixture\n"},
		{path: "/fail", status: http.StatusServiceUnavailable, body: "fixture failure\n"},
		{path: "/items?q=<x>", status: http.StatusOK, body: "fixture GET /items?q=&lt;x&gt;\n", headerName: "X-Fixture-Backend", header: "fixture"},
		{path: "/slow?delay=1ms", status: http.StatusOK, body: "fixture\n"},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
			response := recorder.Result()
			defer response.Body.Close()
			body, _ := io.ReadAll(response.Body)
			if response.StatusCode != test.status || string(body) != test.body {
				t.Fatalf("response = %d %q", response.StatusCode, body)
			}
			if test.headerName != "" && !strings.Contains(response.Header.Get(test.headerName), test.header) {
				t.Fatalf("%s = %q", test.headerName, response.Header.Get(test.headerName))
			}
		})
	}
}

func TestRunRejectsInvalidFlags(t *testing.T) {
	if err := run([]string{"-unknown"}); err == nil {
		t.Fatal("expected flag error")
	}
}
