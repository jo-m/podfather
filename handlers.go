package main

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed templates
var templateFS embed.FS

var funcMap = template.FuncMap{
	"shortID":            shortID,
	"humanSize":          humanSize,
	"formatUnix":         formatUnix,
	"formatTime":         formatTime,
	"formatPorts":        formatPorts,
	"formatExposedPorts": formatExposedPorts,
	"firstName":          firstName,
	"join":               joinStrings,
	"mapKeys":            mapKeys,
	"envName":            envName,
	"envValue":           envValue,
	"basePath":           func() string { return basePath },
	"enableAutoUpdate":   func() bool { return enableAutoUpdate },
	"appState":           appState,
}

func joinStrings(elems any, sep string) string {
	switch v := elems.(type) {
	case []string:
		return strings.Join(v, sep)
	case StringOrSlice:
		return strings.Join([]string(v), sep)
	default:
		return ""
	}
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

func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func formatUnix(ts int64) template.HTML {
	if ts == 0 {
		return "-"
	}
	t := time.Unix(ts, 0)
	return template.HTML(fmt.Sprintf(`<span title="%s">%s</span>`, t.Format("2006-01-02 15:04:05 MST"), timeAgo(t)))
}

func formatTime(t time.Time) template.HTML {
	if t.IsZero() {
		return "-"
	}
	return template.HTML(fmt.Sprintf(`<span title="%s">%s</span>`, t.Format("2006-01-02 15:04:05 MST"), timeAgo(t)))
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

func formatExposedPorts(ep map[string][]string) string {
	if len(ep) == 0 {
		return ""
	}
	var parts []string
	for port, protos := range ep {
		for _, proto := range protos {
			parts = append(parts, port+"/"+proto)
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

func envName(s string) string {
	if i := strings.IndexByte(s, '='); i >= 0 {
		return s[:i]
	}
	return s
}

func envValue(s string) string {
	if i := strings.IndexByte(s, '='); i >= 0 {
		return s[i+1:]
	}
	return ""
}

func mapKeys(m map[string]struct{}) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func firstName(names []string) string {
	if len(names) > 0 {
		return names[0]
	}
	return ""
}

// validID matches container and image IDs (hex, sha256: prefix, or name-like identifiers).
var validID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.:-]*$`)

var pageTemplates map[string]*template.Template

func init() {
	pages := []string{
		"apps.html",
		"autoupdate.html",
		"container.html",
		"containers.html",
		"image.html",
		"images.html",
	}
	pageTemplates = make(map[string]*template.Template, len(pages))
	for _, page := range pages {
		t, err := template.New("").Funcs(funcMap).ParseFS(
			templateFS, "templates/base.html", "templates/"+page,
		)
		if err != nil {
			log.Fatalf("parse template %s: %v", page, err)
		}
		pageTemplates[page] = t
	}
}

func render(w http.ResponseWriter, r *http.Request, page string, data any) {
	t := pageTemplates[page]
	if t == nil {
		log.Printf("[%s] unknown template %s", reqID(r.Context()), page)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "base", data); err != nil {
		log.Printf("[%s] render %s: %v", reqID(r.Context()), page, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf.Bytes())
}

func appState(containers []Container) string {
	for _, c := range containers {
		if c.State == "running" {
			return "running"
		}
	}
	if len(containers) > 0 {
		return containers[0].State
	}
	return "unknown"
}

func buildAppCategories(containers []Container) []AppCategory {
	appMap := make(map[string]*App)

	for _, c := range containers {
		name := c.Labels[appLabelPrefix+"name"]
		if name == "" {
			continue
		}

		app, exists := appMap[name]
		if !exists {
			sortIdx := 0
			if s := c.Labels[appLabelPrefix+"sort-index"]; s != "" {
				if v, err := strconv.Atoi(s); err == nil {
					sortIdx = v
				}
			}
			app = &App{
				Name:        name,
				Icon:        c.Labels[appLabelPrefix+"icon"],
				Category:    c.Labels[appLabelPrefix+"category"],
				SortIndex:   sortIdx,
				Subtitle:    c.Labels[appLabelPrefix+"subtitle"],
				Description: c.Labels[appLabelPrefix+"description"],
				URL:         c.Labels[appLabelPrefix+"url"],
			}
			appMap[name] = app
		}
		app.Containers = append(app.Containers, c)
	}

	catMap := make(map[string][]App)
	for _, app := range appMap {
		cat := app.Category
		if cat == "" {
			cat = "Uncategorized"
		}
		catMap[cat] = append(catMap[cat], *app)
	}

	for cat := range catMap {
		apps := catMap[cat]
		sort.Slice(apps, func(i, j int) bool {
			if apps[i].SortIndex != apps[j].SortIndex {
				return apps[i].SortIndex < apps[j].SortIndex
			}
			return apps[i].Name < apps[j].Name
		})
		catMap[cat] = apps
	}

	var categories []AppCategory
	for cat, apps := range catMap {
		categories = append(categories, AppCategory{Name: cat, Apps: apps})
	}
	sort.Slice(categories, func(i, j int) bool {
		if categories[i].Name == "Uncategorized" {
			return false
		}
		if categories[j].Name == "Uncategorized" {
			return true
		}
		return categories[i].Name < categories[j].Name
	})

	return categories
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	var list []Container
	if err := podmanGet("/containers/json?all=true", &list); err != nil {
		log.Printf("[%s] podman API error: %v", reqID(r.Context()), err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	for _, c := range list {
		if c.Labels[appLabelPrefix+"name"] != "" {
			http.Redirect(w, r, basePath+"/apps", http.StatusTemporaryRedirect)
			return
		}
	}
	http.Redirect(w, r, basePath+"/containers", http.StatusTemporaryRedirect)
}

func handleApps(w http.ResponseWriter, r *http.Request) {
	var list []Container
	if err := podmanGet("/containers/json?all=true", &list); err != nil {
		log.Printf("[%s] podman API error: %v", reqID(r.Context()), err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	categories := buildAppCategories(list)
	render(w, r, "apps.html", map[string]any{
		"Title":      "Apps",
		"Categories": categories,
	})
}

func handleContainers(w http.ResponseWriter, r *http.Request) {
	var list []Container
	if err := podmanGet("/containers/json?all=true", &list); err != nil {
		log.Printf("[%s] podman API error: %v", reqID(r.Context()), err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Created.After(list[j].Created)
	})
	render(w, r, "containers.html", map[string]any{
		"Title":      "Containers",
		"Containers": list,
	})
}

func handleContainer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID.MatchString(id) {
		http.Error(w, "Invalid container ID", http.StatusBadRequest)
		return
	}
	var c ContainerInspect
	if err := podmanGet("/containers/"+id+"/json", &c); err != nil {
		if errors.Is(err, errNotFound) {
			http.Error(w, "Container Not Found", http.StatusNotFound)
			return
		}
		log.Printf("[%s] podman API error: %v", reqID(r.Context()), err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	name := c.Name
	if name == "" {
		name = shortID(c.ID)
	}
	render(w, r, "container.html", map[string]any{
		"Title":     "Container: " + name,
		"Container": c,
	})
}

func handleImages(w http.ResponseWriter, r *http.Request) {
	var list []ImageSummary
	if err := podmanGet("/images/json", &list); err != nil {
		log.Printf("[%s] podman API error: %v", reqID(r.Context()), err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	render(w, r, "images.html", map[string]any{
		"Title":  "Images",
		"Images": list,
	})
}

func handleImage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID.MatchString(id) {
		http.Error(w, "Invalid image ID", http.StatusBadRequest)
		return
	}
	var img ImageInspect
	if err := podmanGet("/images/"+id+"/json", &img); err != nil {
		if errors.Is(err, errNotFound) {
			http.Error(w, "Image Not Found", http.StatusNotFound)
			return
		}
		log.Printf("[%s] podman API error: %v", reqID(r.Context()), err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	name := ""
	if len(img.RepoTags) > 0 {
		name = img.RepoTags[0]
	}
	if name == "" {
		name = shortID(img.ID)
	}
	render(w, r, "image.html", map[string]any{
		"Title": "Image: " + name,
		"Image": img,
	})
}

func handleAutoUpdate(w http.ResponseWriter, r *http.Request) {
	if !enableAutoUpdate {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "podman", "auto-update")
	out, err := cmd.CombinedOutput()

	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}
	render(w, r, "autoupdate.html", map[string]any{
		"Title":  "Auto Update",
		"Output": string(out),
		"Error":  errMsg,
	})
}
