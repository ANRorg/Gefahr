// Command backend is an observable upstream fixture for integration testing.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

func main() {
	address := flag.String("address", ":9001", "listen address")
	name := flag.String("name", "backend-1", "response identity")
	flag.Parse()
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		delay, _ := time.ParseDuration(r.URL.Query().Get("delay"))
		if delay <= 0 {
			delay = time.Second
		}
		time.Sleep(delay)
		fmt.Fprintln(w, *name)
	})
	mux.HandleFunc("/cache", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=30")
		fmt.Fprintln(w, *name)
	})
	mux.HandleFunc("/cookie", func(w http.ResponseWriter, _ *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "fixture", HttpOnly: true})
		fmt.Fprintln(w, *name)
	})
	mux.HandleFunc("/chunked", func(w http.ResponseWriter, _ *http.Request) {
		flusher, _ := w.(http.Flusher)
		for i := range 3 {
			fmt.Fprintln(w, strconv.Itoa(i)+":"+*name)
			if flusher != nil {
				flusher.Flush()
			}
		}
	})
	mux.HandleFunc("/fail", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "fixture failure", http.StatusServiceUnavailable)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Fixture-Backend", *name)
		fmt.Fprintf(w, "%s %s %s\n", *name, r.Method, r.URL.RequestURI())
	})
	log.Printf("fixture %s listening on %s", *name, *address)
	log.Fatal(http.ListenAndServe(*address, mux))
}
