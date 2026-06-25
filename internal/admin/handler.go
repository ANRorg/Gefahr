// Package admin exposes operational endpoints on a separate listener.
package admin

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// NewHandler builds liveness, readiness, and optional metrics routes.
func NewHandler(live, ready func() bool, metrics http.Handler, bearerToken string) http.Handler {
	router := chi.NewRouter()
	router.Get("/livez", probe(live))
	router.Get("/readyz", probe(ready))
	if metrics != nil {
		router.Handle("/metrics", metrics)
	}
	if bearerToken != "" {
		return requireBearerToken(router, bearerToken)
	}
	return router
}

func requireBearerToken(next http.Handler, token string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const prefix = "Bearer "
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, prefix) || subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(header, prefix)), []byte(token)) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="goproxy-admin"`)
			http.Error(w, "unauthorized\n", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func probe(check func() bool) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if check == nil || !check() {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("not ready\n"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	}
}
