package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

// errNotFound is returned when the Podman API responds with 404.
var errNotFound = errors.New("not found")

func socketPath() string {
	if s := os.Getenv("PODMAN_SOCKET"); s != "" {
		return s
	}
	xdg := os.Getenv("XDG_RUNTIME_DIR")
	if xdg == "" {
		xdg = fmt.Sprintf("/run/user/%d", os.Getuid())
	}
	return xdg + "/podman/podman.sock"
}

func newPodmanClient(sock string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", sock)
			},
		},
		Timeout: 30 * time.Second,
	}
}

func (s *Server) podmanGet(path string, result any) error {
	resp, err := s.podmanClient.Get(s.podmanBaseURL + path)
	if err != nil {
		return fmt.Errorf("podman API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		io.Copy(io.Discard, resp.Body)
		return errNotFound
	}
	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return fmt.Errorf("podman API %s: %s", path, resp.Status)
	}
	err = json.NewDecoder(resp.Body).Decode(result)
	io.Copy(io.Discard, resp.Body)
	return err
}
