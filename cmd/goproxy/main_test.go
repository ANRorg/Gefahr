package main

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/anouar/goproxy/internal/config"
)

func TestRunHealthcheckRequiresOKResponse(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		transport  error
		wantErr    bool
		wantClosed bool
	}{
		{name: "ready", status: http.StatusOK, wantClosed: true},
		{name: "not ready", status: http.StatusServiceUnavailable, wantErr: true, wantClosed: true},
		{name: "transport failure", transport: errors.New("dial failed"), wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := &trackingBody{Reader: strings.NewReader("status")}
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				if test.transport != nil {
					return nil, test.transport
				}
				return &http.Response{StatusCode: test.status, Body: body, Header: make(http.Header)}, nil
			})}
			err := runHealthcheck(client, "http://127.0.0.1:9090/readyz", "")
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v", err)
			}
			if body.closed != test.wantClosed {
				t.Fatalf("body closed = %t", body.closed)
			}
		})
	}
}

func TestRunHealthcheckSendsBearerToken(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if got := request.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("Authorization = %q", got)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("ok")), Header: make(http.Header)}, nil
	})}
	if err := runHealthcheck(client, "http://127.0.0.1:9090/readyz", "secret"); err != nil {
		t.Fatal(err)
	}
}

func TestAdminTokenReadsConfiguredEnvironment(t *testing.T) {
	t.Setenv("GOPROXY_ADMIN_TOKEN", "secret")
	cfg := config.Default()
	cfg.Admin.AuthTokenEnv = "GOPROXY_ADMIN_TOKEN"
	token, err := adminToken(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if token != "secret" {
		t.Fatalf("token = %q", token)
	}
}

func TestAdminTokenRequiresConfiguredEnvironment(t *testing.T) {
	cfg := config.Default()
	cfg.Admin.AuthTokenEnv = "GOPROXY_ADMIN_TOKEN"
	if _, err := adminToken(cfg); err == nil {
		t.Fatal("expected missing token error")
	}
}

func TestVersionStringIncludesBuildMetadata(t *testing.T) {
	previousVersion, previousCommit := version, commit
	t.Cleanup(func() { version, commit = previousVersion, previousCommit })
	version, commit = "v1.0.0", "abc1234"
	if got := versionString(); got != "goproxy version=v1.0.0 commit=abc1234" {
		t.Fatalf("version = %q", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type trackingBody struct {
	io.Reader
	closed bool
}

func (b *trackingBody) Close() error {
	b.closed = true
	return nil
}
