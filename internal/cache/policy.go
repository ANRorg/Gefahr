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
	if headerPresent(r.Header, "Authorization") {
		return false, "authorization"
	}
	if headerPresent(r.Header, "Cookie") {
		return false, "cookie"
	}
	if headerPresent(r.Header, "Range") {
		return false, "range"
	}
	for _, name := range []string{"If-Match", "If-None-Match", "If-Modified-Since", "If-Unmodified-Since", "If-Range"} {
		if headerPresent(r.Header, name) {
			return false, "conditional"
		}
	}
	cacheControl := cacheControlValue(r.Header)
	if hasDirective(cacheControl, "no-cache") || hasDirective(cacheControl, "no-store") || hasDirective(cacheControl, "max-age") || hasDirective(cacheControl, "min-fresh") || strings.Contains(strings.ToLower(strings.Join(r.Header.Values("Pragma"), ",")), "no-cache") {
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
	if headerPresent(header, "Set-Cookie") {
		return Decision{Reason: "set_cookie"}
	}
	if headerPresent(header, "Vary") {
		return Decision{Reason: "vary"}
	}
	cacheControl := cacheControlValue(header)
	for _, directive := range []string{"no-store", "private", "no-cache"} {
		if hasDirective(cacheControl, directive) {
			return Decision{Reason: directive}
		}
	}
	ttl := defaultTTL
	if seconds, present, valid := directiveSeconds(cacheControl, "s-maxage"); present {
		if !valid || !durationSecondsValid(seconds) {
			return Decision{Reason: "invalid_freshness"}
		}
		ttl = time.Duration(seconds) * time.Second
	} else if seconds, present, valid := directiveSeconds(cacheControl, "max-age"); present {
		if !valid || !durationSecondsValid(seconds) {
			return Decision{Reason: "invalid_freshness"}
		}
		ttl = time.Duration(seconds) * time.Second
	} else if headerPresent(header, "Expires") {
		expires, err := http.ParseTime(header.Get("Expires"))
		if err != nil {
			return Decision{Reason: "invalid_expires"}
		}
		ttl = time.Until(expires)
	}
	if age, err := strconv.ParseInt(strings.TrimSpace(header.Get("Age")), 10, 64); headerPresent(header, "Age") {
		if len(header.Values("Age")) != 1 {
			return Decision{Reason: "invalid_age"}
		}
		if err != nil || age < 0 || !durationSecondsValid(age) {
			return Decision{Reason: "invalid_age"}
		}
		ttl -= time.Duration(age) * time.Second
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

// directiveSeconds extracts and parses a numeric directive value from a comma-separated directive string. It locates the directive matching the given name and returns its parsed integer value. Returns (seconds, present, valid) where present is true if the directive was found, and valid is true if the directive was found and has a valid non-negative integer value.
func directiveSeconds(value, name string) (seconds int64, present, valid bool) {
	var found bool
	for _, part := range strings.Split(value, ",") {
		pieces := strings.SplitN(part, "=", 2)
		if strings.TrimSpace(pieces[0]) != name {
			continue
		}
		if found {
			return 0, true, false
		}
		found = true
		if len(pieces) != 2 {
			return 0, true, false
		}
		parsed, err := strconv.ParseInt(strings.Trim(strings.TrimSpace(pieces[1]), `"`), 10, 64)
		if err != nil || parsed < 0 {
			return 0, true, false
		}
		seconds = parsed
	}
	return seconds, found, found
}

func cacheControlValue(header http.Header) string {
	return strings.ToLower(strings.Join(header.Values("Cache-Control"), ","))
}

func headerPresent(header http.Header, name string) bool {
	return len(header.Values(name)) > 0
}

func durationSecondsValid(seconds int64) bool {
	const maxDurationSeconds = int64((1<<63)-1) / int64(time.Second)
	return seconds <= maxDurationSeconds
}
