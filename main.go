// Package main implements podview, a simple web dashboard for rootless Podman.
// It connects to the Podman API socket and renders container/image information
// server-side using Go templates. No JavaScript, no external dependencies.
package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

//go:embed templates
var templateFS embed.FS

// --- Podman API response types ---
// Fields like Config.Env are intentionally omitted so secrets and
// environment variables are never parsed or displayed.

type Container struct {
	ID      string    `json:"Id"`
	Names   []string  `json:"Names"`
	Image   string    `json:"Image"`
	Command []string  `json:"Command"`
	Created time.Time `json:"Created"`
	State   string    `json:"State"`
	Status  string    `json:"Status"`
	Ports   []Port    `json:"Ports"`
}

type Port struct {
	HostIP        string `json:"host_ip"`
	HostPort      uint16 `json:"host_port"`
	ContainerPort uint16 `json:"container_port"`
	Protocol      string `json:"protocol"`
}

type ContainerInspect struct {
	ID              string           `json:"Id"`
	Name            string           `json:"Name"`
	Created         time.Time        `json:"Created"`
	ImageName       string           `json:"ImageName"`
	State           ContainerState   `json:"State"`
	Config          ContainerConfig  `json:"Config"`
	Mounts          []Mount          `json:"Mounts"`
	NetworkSettings *NetworkSettings `json:"NetworkSettings"`
	RestartCount    int32            `json:"RestartCount"`
	HostConfig      *HostConfig      `json:"HostConfig"`
}

type ContainerState struct {
	Status     string    `json:"Status"`
	Running    bool      `json:"Running"`
	StartedAt  time.Time `json:"StartedAt"`
	FinishedAt time.Time `json:"FinishedAt"`
	ExitCode   int32     `json:"ExitCode"`
	Health     *Health   `json:"Health,omitempty"`
}

type Health struct {
	Status string `json:"Status"`
}

type ContainerConfig struct {
	Hostname string            `json:"Hostname"`
	Image    string            `json:"Image"`
	Cmd      []string          `json:"Cmd"`
	Labels   map[string]string `json:"Labels"`
	// Env is intentionally omitted — never show environment variables.
}

type HostConfig struct {
	RestartPolicy RestartPolicy `json:"RestartPolicy"`
}

type RestartPolicy struct {
	Name              string `json:"Name"`
	MaximumRetryCount uint   `json:"MaximumRetryCount"`
}

type Mount struct {
	Type        string `json:"Type"`
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
	RW          bool   `json:"RW"`
}

type NetworkSettings struct {
	Ports map[string][]HostPort `json:"Ports"`
}

type HostPort struct {
	HostIP   string `json:"HostIp"`
	HostPort string `json:"HostPort"`
}

type ImageSummary struct {
	ID       string   `json:"Id"`
	RepoTags []string `json:"RepoTags"`
	Created  int64    `json:"Created"`
	Size     int64    `json:"Size"`
}

// --- Template helpers ---

func shortID(id string) string {
	id = strings.TrimPrefix(id, "sha256:")
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func humanSize(b int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case b >= GB:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func formatUnix(ts int64) string {
	if ts == 0 {
		return "-"
	}
	return time.Unix(ts, 0).Format("2006-01-02 15:04:05")
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02 15:04:05")
}

func formatPorts(ports []Port) string {
	if len(ports) == 0 {
		return ""
	}
	var parts []string
	for _, p := range ports {
		s := fmt.Sprintf("%d/%s", p.ContainerPort, p.Protocol)
		if p.HostPort > 0 {
			host := fmt.Sprintf("0.0.0.0:%d", p.HostPort)
			if p.HostIP != "" {
				host = fmt.Sprintf("%s:%d", p.HostIP, p.HostPort)
			}
			s = host + "->" + s
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, ", ")
}

func firstName(names []string) string {
	if len(names) > 0 {
		return names[0]
	}
	return ""
}

var funcMap = template.FuncMap{
	"shortID":     shortID,
	"humanSize":   humanSize,
	"formatUnix":  formatUnix,
	"formatTime":  formatTime,
	"formatPorts": formatPorts,
	"firstName":   firstName,
	"join":        strings.Join,
}

// --- Template rendering ---

func render(w http.ResponseWriter, page string, data any) {
	t, err := template.New("").Funcs(funcMap).ParseFS(
		templateFS, "templates/base.html", "templates/"+page,
	)
	if err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "base", data); err != nil {
		log.Printf("render %s: %v", page, err)
	}
}

// --- Podman API client ---

var podman *http.Client

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

func podmanGet(path string, result any) error {
	resp, err := podman.Get("http://d/v4.0.0/libpod" + path)
	if err != nil {
		return fmt.Errorf("podman API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("podman API %s: %s", path, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(result)
}

// --- HTTP handlers ---

func handleContainers(w http.ResponseWriter, r *http.Request) {
	var list []Container
	if err := podmanGet("/containers/json?all=true", &list); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	render(w, "containers.html", map[string]any{
		"Title":      "Containers",
		"Containers": list,
	})
}

func handleContainer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var c ContainerInspect
	if err := podmanGet("/containers/"+id+"/json", &c); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	name := c.Name
	if name == "" {
		name = shortID(c.ID)
	}
	render(w, "container.html", map[string]any{
		"Title":     "Container: " + name,
		"Container": c,
	})
}

func handleImages(w http.ResponseWriter, r *http.Request) {
	var list []ImageSummary
	if err := podmanGet("/images/json", &list); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	render(w, "images.html", map[string]any{
		"Title":  "Images",
		"Images": list,
	})
}

func handleAutoUpdate(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "podman", "auto-update")
	out, err := cmd.CombinedOutput()

	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}
	render(w, "autoupdate.html", map[string]any{
		"Title":  "Auto Update",
		"Output": string(out),
		"Error":  errMsg,
	})
}

// --- Main ---

func main() {
	sock := socketPath()
	podman = &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", sock)
			},
		},
		Timeout: 30 * time.Second,
	}

	addr := ":8080"
	if a := os.Getenv("LISTEN_ADDR"); a != "" {
		addr = a
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", handleContainers)
	mux.HandleFunc("GET /container/{id}", handleContainer)
	mux.HandleFunc("GET /images", handleImages)
	mux.HandleFunc("POST /auto-update", handleAutoUpdate)

	log.Printf("podview listening on %s (socket: %s)", addr, sock)
	log.Fatal(http.ListenAndServe(addr, mux))
}
