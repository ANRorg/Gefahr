// Package admin exposes operational endpoints on a separate listener.
package admin

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// NewHandler builds liveness, readiness, and optional metrics routes.
func NewHandler(live, ready func() bool, metrics http.Handler) http.Handler {
	router := chi.NewRouter()
	router.Get("/livez", probe(live))
	router.Get("/readyz", probe(ready))
	if metrics != nil {
		router.Handle("/metrics", metrics)
	}
	return router
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
