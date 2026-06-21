package cache

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Decision explains whether and for how long a response may be stored.
type Decision struct {
	Cacheable bool
	TTL       time.Duration
	Reason    string
}

// Key returns the cache key for a safe request. The host and full request URI
// prevent collisions between virtual hosts and query variants.
func Key(r *http.Request) string {
	return http.MethodGet + "\x00" + strings.ToLower(r.Host) + "\x00" + r.URL.RequestURI()
}

// RequestEligible reports whether a request can use the shared cache.
func RequestEligible(r *http.Request) (bool, string) {
	if r.Method != http.MethodGet {
		return false, "method"
	}
	if r.Header.Get("Authorization") != "" {
		return false, "authorization"
	}
	if r.Header.Get("Cookie") != "" {
		return false, "cookie"
	}
	cacheControl := strings.ToLower(r.Header.Get("Cache-Control"))
	if hasDirective(cacheControl, "no-cache") || hasDirective(cacheControl, "no-store") || strings.EqualFold(r.Header.Get("Pragma"), "no-cache") {
		return false, "request_cache_control"
	}
	return true, "eligible"
}

// Evaluate applies conservative shared-cache response rules.
func Evaluate(r *http.Request, status int, header http.Header, defaultTTL time.Duration) Decision {
	if ok, reason := RequestEligible(r); !ok {
		return Decision{Reason: reason}
	}
	if status != http.StatusOK {
		return Decision{Reason: "status"}
	}
	if header.Get("Set-Cookie") != "" {
		return Decision{Reason: "set_cookie"}
	}
	if header.Get("Vary") != "" {
		return Decision{Reason: "vary"}
	}
	cacheControl := strings.ToLower(header.Get("Cache-Control"))
	for _, directive := range []string{"no-store", "private", "no-cache"} {
		if hasDirective(cacheControl, directive) {
			return Decision{Reason: directive}
		}
	}
	ttl := defaultTTL
	if seconds, ok := directiveSeconds(cacheControl, "s-maxage"); ok {
		ttl = time.Duration(seconds) * time.Second
	} else if seconds, ok := directiveSeconds(cacheControl, "max-age"); ok {
		ttl = time.Duration(seconds) * time.Second
	}
	if ttl <= 0 {
		return Decision{Reason: "zero_ttl"}
	}
	return Decision{Cacheable: true, TTL: ttl, Reason: "cacheable"}
}

func hasDirective(value, name string) bool {
	for _, part := range strings.Split(value, ",") {
		if strings.TrimSpace(strings.SplitN(part, "=", 2)[0]) == name {
			return true
		}
	}
	return false
}

func directiveSeconds(value, name string) (int64, bool) {
	for _, part := range strings.Split(value, ",") {
		pieces := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(pieces) != 2 || pieces[0] != name {
			continue
		}
		seconds, err := strconv.ParseInt(strings.Trim(strings.TrimSpace(pieces[1]), `"`), 10, 64)
		return seconds, err == nil && seconds >= 0
	}
	return 0, false
}
