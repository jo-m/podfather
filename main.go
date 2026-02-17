// Package main implements podfather, a simple web dashboard for rootless Podman.
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
const csrfTokenKey ctxKey = 1

func reqID(ctx context.Context) string {
	if id, ok := ctx.Value(reqIDKey).(string); ok {
		return id
	}
	return "-"
}

// Server holds all per-instance state for the podfather web server.
type Server struct {
	basePath         string
	enableAutoUpdate bool
	externalApps     []App
	podmanClient     *http.Client
	podmanBaseURL    string
}

func (s *Server) newMux(podmanBin string) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handleRoot)
	mux.HandleFunc("GET /apps", s.handleApps)
	mux.HandleFunc("GET /containers", s.handleContainers)
	mux.HandleFunc("GET /container/{id}", s.handleContainer)
	mux.HandleFunc("GET /images", s.handleImages)
	mux.HandleFunc("GET /image/{id}", s.handleImage)
	mux.HandleFunc("POST /auto-update", s.handleAutoUpdate(podmanBin))
	return mux
}

func main() {
	sock := socketPath()

	addr := "127.0.0.1:8080"
	if a := os.Getenv("LISTEN_ADDR"); a != "" {
		addr = a
	}

	s := &Server{
		basePath:         strings.TrimRight(os.Getenv("BASE_PATH"), "/"),
		enableAutoUpdate: os.Getenv("ENABLE_AUTOUPDATE_BUTTON") == "true",
		externalApps:     parseExternalApps(),
		podmanClient:     newPodmanClient(sock),
		podmanBaseURL:    "http://d/v4.0.0/libpod",
	}

	mux := s.newMux("podman")

	var handler http.Handler = mux
	if s.basePath != "" {
		handler = http.StripPrefix(s.basePath, mux)
	}

	host := addr
	if strings.HasPrefix(host, ":") {
		host = "localhost" + host
	}
	log.Printf("podfather listening on http://%s%s (socket: %s)", host, s.basePath, sock)
	handler = s.csrfProtect(handler)
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
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; img-src data:; form-action 'self'")

		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Printf("[%s] %s %s %d %s", id, r.Method, r.URL.Path, sw.status, time.Since(start).Round(time.Millisecond))
	})
}
