package cache

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEvaluateAcceptsSharedCacheResponse(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://api.test/items?page=2", nil)
	decision := Evaluate(req, http.StatusOK, http.Header{"Cache-Control": {"public, s-maxage=60, max-age=30"}}, time.Second)
	if !decision.Cacheable || decision.TTL != time.Minute {
		t.Fatalf("decision = %#v", decision)
	}
	if Key(req) == Key(httptest.NewRequest(http.MethodGet, "http://api.test/items?page=3", nil)) {
		t.Fatal("query was omitted from key")
	}
}

func TestEvaluateRejectsUnsafeResponses(t *testing.T) {
	base := httptest.NewRequest(http.MethodGet, "http://api.test/items", nil)
	tests := []struct {
		name    string
		request *http.Request
		status  int
		header  http.Header
		reason  string
	}{
		{"authorization", withHeader(base.Clone(base.Context()), "Authorization", "Bearer secret"), 200, nil, "authorization"},
		{"cookie", withHeader(base.Clone(base.Context()), "Cookie", "session=x"), 200, nil, "cookie"},
		{"range", withHeader(base.Clone(base.Context()), "Range", "bytes=0-10"), 200, nil, "range"},
		{"conditional", withHeader(base.Clone(base.Context()), "If-None-Match", `"abc"`), 200, nil, "conditional"},
		{"set-cookie", base, 200, http.Header{"Set-Cookie": {"session=x"}}, "set_cookie"},
		{"set-cookie-second-field", base, 200, http.Header{"Set-Cookie": {"", "session=x"}}, "set_cookie"},
		{"private", base, 200, http.Header{"Cache-Control": {"private"}}, "private"},
		{"private-second-field", base, 200, http.Header{"Cache-Control": {"public", "private"}}, "private"},
		{"invalid-freshness", base, 200, http.Header{"Cache-Control": {"max-age=invalid"}}, "invalid_freshness"},
		{"duplicate-freshness", base, 200, http.Header{"Cache-Control": {"max-age=60, max-age=0"}}, "invalid_freshness"},
		{"overflow-freshness", base, 200, http.Header{"Cache-Control": {"max-age=9223372036854775807"}}, "invalid_freshness"},
		{"stale-age", base, 200, http.Header{"Cache-Control": {"max-age=10"}, "Age": {"10"}}, "zero_ttl"},
		{"expired", base, 200, http.Header{"Expires": {time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat)}}, "zero_ttl"},
		{"vary", base, 200, http.Header{"Vary": {"Accept-Encoding"}}, "vary"},
		{"status", base, 404, nil, "status"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Evaluate(tc.request, tc.status, tc.header, time.Minute)
			if got.Cacheable || got.Reason != tc.reason {
				t.Fatalf("decision = %#v", got)
			}
		})
	}
}

func withHeader(r *http.Request, name, value string) *http.Request {
	r.Header.Set(name, value)
	return r
}
