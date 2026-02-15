// Package main implements podview, a simple web dashboard for rootless Podman.
// It connects to the Podman API socket and renders container/image information
// server-side using Go templates. No JavaScript, no external dependencies.
package main

import (
	"log"
	"net/http"
	"os"
)

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
	log.Fatal(http.ListenAndServe(addr, mux))
}
