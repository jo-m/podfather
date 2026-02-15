// Package main implements podview, a simple web dashboard for rootless Podman.
// It connects to the Podman API socket and renders container/image information
// server-side using Go templates. No JavaScript, no external dependencies.
package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type ctxKey int

const reqIDKey ctxKey = 0

func reqID(ctx context.Context) string {
	if id, ok := ctx.Value(reqIDKey).(string); ok {
		return id
	}
	return "-"
}

func main() {
	sock := socketPath()
	initPodmanClient(sock)

	addr := ":8080"
	if a := os.Getenv("LISTEN_ADDR"); a != "" {
		addr = a
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", handleContainers)
	mux.HandleFunc("GET /container/{id}", handleContainer)
	mux.HandleFunc("GET /images", handleImages)
	mux.HandleFunc("GET /image/{id}", handleImage)
	mux.HandleFunc("POST /auto-update", handleAutoUpdate)

	log.Printf("podview listening on %s (socket: %s)", addr, sock)
	log.Fatal(http.ListenAndServe(addr, logRequests(mux)))
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf [4]byte
		rand.Read(buf[:])
		id := fmt.Sprintf("%x", buf)
		ctx := context.WithValue(r.Context(), reqIDKey, id)
		r = r.WithContext(ctx)

		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Printf("[%s] %s %s %d %s", id, r.Method, r.URL.Path, sw.status, time.Since(start).Round(time.Millisecond))
	})
}
