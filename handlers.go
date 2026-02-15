package main

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"time"
)

//go:embed templates
var templateFS embed.FS

var funcMap = template.FuncMap{
	"shortID":     shortID,
	"humanSize":   humanSize,
	"formatUnix":  formatUnix,
	"formatTime":  formatTime,
	"formatPorts": formatPorts,
	"firstName":   firstName,
	"join":        strings.Join,
}

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

func handleContainers(w http.ResponseWriter, r *http.Request) {
	var list []Container
	if err := podmanGet("/containers/json?all=true", &list); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Created.After(list[j].Created)
	})
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
