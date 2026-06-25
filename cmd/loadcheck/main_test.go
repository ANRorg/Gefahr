package main

import (
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestRunLoadCountsSuccessAndFailure(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ok.Close()
	failures, elapsed := runLoad(ok.Client(), ok.URL, 20, 4)
	if failures != 0 || elapsed <= 0 {
		t.Fatalf("success failures=%d elapsed=%s", failures, elapsed)
	}

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "failed", http.StatusBadGateway)
	}))
	defer failing.Close()
	failures, _ = runLoad(failing.Client(), failing.URL, 7, 3)
	if failures != 7 {
		t.Fatalf("failure count = %d", failures)
	}
}

func TestRunLoadCountsInvalidRequestTargets(t *testing.T) {
	failures, _ := runLoad(http.DefaultClient, "://bad-url", 5, 2)
	if failures != 5 {
		t.Fatalf("invalid target failures = %d", failures)
	}
}

func TestScrapeParsesOnlyPlainNumericMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "# comment\nmetric_with_label{route=\"api\"} 1\ngo_goroutines 7\nbad line\nnot_number nope\n")
	}))
	defer server.Close()

	values, err := scrape(server.Client(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if values["go_goroutines"] != 7 || len(values) != 1 {
		t.Fatalf("values = %v", values)
	}
}

func TestScrapeRejectsNonOKMetricsResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "no", http.StatusUnauthorized)
	}))
	defer server.Close()

	if _, err := scrape(server.Client(), server.URL); err == nil {
		t.Fatal("expected scrape error")
	}
}

func TestMainRunsSuccessfulLoadCheck(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()
	metrics := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "go_goroutines 1\ngo_memstats_alloc_bytes 2\n")
	}))
	defer metrics.Close()

	previousArgs, previousFlags, previousStdout := os.Args, flag.CommandLine, os.Stdout
	t.Cleanup(func() {
		os.Args, flag.CommandLine, os.Stdout = previousArgs, previousFlags, previousStdout
	})
	flag.CommandLine = flag.NewFlagSet("loadcheck", flag.ContinueOnError)
	os.Args = []string{"loadcheck", "-target", target.URL, "-metrics", metrics.URL, "-requests", "1", "-concurrency", "1"}
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = write
	main()
	write.Close()
	output, err := io.ReadAll(read)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), "requests=1 failures=0") {
		t.Fatalf("output = %s", output)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
