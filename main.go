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
	"strings"
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

var basePath string

func main() {
	sock := socketPath()
	initPodmanClient(sock)

	addr := "127.0.0.1:8080"
	if a := os.Getenv("LISTEN_ADDR"); a != "" {
		addr = a
	}

	basePath = strings.TrimRight(os.Getenv("BASE_PATH"), "/")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", handleRoot)
	mux.HandleFunc("GET /apps", handleApps)
	mux.HandleFunc("GET /containers", handleContainers)
	mux.HandleFunc("GET /container/{id}", handleContainer)
	mux.HandleFunc("GET /images", handleImages)
	mux.HandleFunc("GET /image/{id}", handleImage)
	mux.HandleFunc("POST /auto-update", handleAutoUpdate)

	var handler http.Handler = mux
	if basePath != "" {
		handler = http.StripPrefix(basePath, mux)
	}

	host := addr
	if strings.HasPrefix(host, ":") {
		host = "localhost" + host
	}
	log.Printf("podview listening on http://%s%s (socket: %s)", host, basePath, sock)
	log.Fatal(http.ListenAndServe(addr, logRequests(handler)))
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

		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")

		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Printf("[%s] %s %s %d %s", id, r.Method, r.URL.Path, sw.status, time.Since(start).Round(time.Millisecond))
	})
}
