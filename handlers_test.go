package main

import (
	"encoding/json"
	"os"
	"testing"
)

func loadTestContainers(t *testing.T) []Container {
	t.Helper()
	data, err := os.ReadFile("testdata/containers.json")
	if err != nil {
		t.Fatal(err)
	}
	var list []Container
	if err := json.Unmarshal(data, &list); err != nil {
		t.Fatal(err)
	}
	return list
}

func TestLoadContainers(t *testing.T) {
	list := loadTestContainers(t)
	if len(list) != 11 {
		t.Fatalf("expected 11 containers, got %d", len(list))
	}

	// Verify fields are parsed correctly for a known container.
	var jellyfin *Container
	for i := range list {
		if len(list[i].Names) > 0 && list[i].Names[0] == "jellyfin" {
			jellyfin = &list[i]
			break
		}
	}
	if jellyfin == nil {
		t.Fatal("jellyfin container not found")
	}
	if jellyfin.State != "running" {
		t.Errorf("jellyfin state = %q, want running", jellyfin.State)
	}
	if jellyfin.Image != "docker.io/library/nginx:alpine" {
		t.Errorf("jellyfin image = %q", jellyfin.Image)
	}
	if len(jellyfin.Ports) != 1 {
		t.Fatalf("jellyfin ports = %d, want 1", len(jellyfin.Ports))
	}
	if jellyfin.Ports[0].HostPort != 8096 || jellyfin.Ports[0].ContainerPort != 80 {
		t.Errorf("jellyfin port = %d->%d, want 8096->80", jellyfin.Ports[0].HostPort, jellyfin.Ports[0].ContainerPort)
	}
	if jellyfin.Labels["ch.jo-m.go.podview.app.name"] != "Jellyfin" {
		t.Errorf("jellyfin app name label = %q", jellyfin.Labels["ch.jo-m.go.podview.app.name"])
	}
}

func TestBuildAppCategories(t *testing.T) {
	list := loadTestContainers(t)
	categories := buildAppCategories(list)

	// Expected categories in order: Infrastructure, Media, Monitoring, Uncategorized.
	wantCats := []string{"Infrastructure", "Media", "Monitoring", "Uncategorized"}
	if len(categories) != len(wantCats) {
		t.Fatalf("got %d categories, want %d", len(categories), len(wantCats))
	}
	for i, want := range wantCats {
		if categories[i].Name != want {
			t.Errorf("category[%d] = %q, want %q", i, categories[i].Name, want)
		}
	}

	// Infrastructure: Traefik (idx=0), Gitea (idx=1).
	infra := categories[0]
	if len(infra.Apps) != 2 {
		t.Fatalf("Infrastructure has %d apps, want 2", len(infra.Apps))
	}
	if infra.Apps[0].Name != "Traefik" {
		t.Errorf("Infrastructure[0] = %q, want Traefik", infra.Apps[0].Name)
	}
	if infra.Apps[1].Name != "Gitea" {
		t.Errorf("Infrastructure[1] = %q, want Gitea", infra.Apps[1].Name)
	}

	// Gitea has 2 containers (gitea-web + gitea-db).
	if len(infra.Apps[1].Containers) != 2 {
		t.Errorf("Gitea has %d containers, want 2", len(infra.Apps[1].Containers))
	}

	// Media: Jellyfin (idx=1), Navidrome (idx=2).
	media := categories[1]
	if len(media.Apps) != 2 {
		t.Fatalf("Media has %d apps, want 2", len(media.Apps))
	}
	if media.Apps[0].Name != "Jellyfin" {
		t.Errorf("Media[0] = %q, want Jellyfin", media.Apps[0].Name)
	}
	if media.Apps[1].Name != "Navidrome" {
		t.Errorf("Media[1] = %q, want Navidrome", media.Apps[1].Name)
	}

	// Monitoring: Grafana (idx=1), Prometheus (idx=2).
	mon := categories[2]
	if len(mon.Apps) != 2 {
		t.Fatalf("Monitoring has %d apps, want 2", len(mon.Apps))
	}
	if mon.Apps[0].Name != "Grafana" {
		t.Errorf("Monitoring[0] = %q, want Grafana", mon.Apps[0].Name)
	}
	if mon.Apps[1].Name != "Prometheus" {
		t.Errorf("Monitoring[1] = %q, want Prometheus", mon.Apps[1].Name)
	}

	// Uncategorized: Whoami (no category label).
	uncat := categories[3]
	if len(uncat.Apps) != 1 {
		t.Fatalf("Uncategorized has %d apps, want 1", len(uncat.Apps))
	}
	if uncat.Apps[0].Name != "Whoami" {
		t.Errorf("Uncategorized[0] = %q, want Whoami", uncat.Apps[0].Name)
	}
}

func TestBuildAppCategoriesMetadata(t *testing.T) {
	list := loadTestContainers(t)
	categories := buildAppCategories(list)

	// Find Jellyfin and check all metadata fields are extracted.
	var jellyfin *App
	for _, cat := range categories {
		for i := range cat.Apps {
			if cat.Apps[i].Name == "Jellyfin" {
				jellyfin = &cat.Apps[i]
			}
		}
	}
	if jellyfin == nil {
		t.Fatal("Jellyfin app not found")
	}
	if jellyfin.Icon != "🎬" {
		t.Errorf("icon = %q, want 🎬", jellyfin.Icon)
	}
	if jellyfin.Category != "Media" {
		t.Errorf("category = %q, want Media", jellyfin.Category)
	}
	if jellyfin.SortIndex != 1 {
		t.Errorf("sort-index = %d, want 1", jellyfin.SortIndex)
	}
	if jellyfin.Subtitle != "Media Server" {
		t.Errorf("subtitle = %q, want Media Server", jellyfin.Subtitle)
	}
	if jellyfin.Description != "Stream your media library" {
		t.Errorf("description = %q", jellyfin.Description)
	}
	if jellyfin.URL != "http://localhost:8096" {
		t.Errorf("url = %q", jellyfin.URL)
	}
}

func TestBuildAppCategoriesNoApps(t *testing.T) {
	// Containers without app labels produce no categories.
	containers := []Container{
		{ID: "aaa", Names: []string{"redis"}, State: "running", Labels: map[string]string{}},
		{ID: "bbb", Names: []string{"backup"}, State: "running"},
	}
	categories := buildAppCategories(containers)
	if len(categories) != 0 {
		t.Errorf("got %d categories, want 0", len(categories))
	}
}

func TestBuildAppCategoriesEmpty(t *testing.T) {
	categories := buildAppCategories(nil)
	if len(categories) != 0 {
		t.Errorf("got %d categories, want 0", len(categories))
	}
}

func TestAppState(t *testing.T) {
	list := loadTestContainers(t)
	categories := buildAppCategories(list)

	// All demo containers are running, so appState should return "running".
	for _, cat := range categories {
		for _, app := range cat.Apps {
			got := appState(app.Containers)
			if got != "running" {
				t.Errorf("appState(%s) = %q, want running", app.Name, got)
			}
		}
	}

	// Mixed states: running wins.
	mixed := []Container{
		{State: "exited"},
		{State: "running"},
		{State: "exited"},
	}
	if got := appState(mixed); got != "running" {
		t.Errorf("appState(mixed) = %q, want running", got)
	}

	// All exited: returns first container's state.
	exited := []Container{
		{State: "exited"},
		{State: "created"},
	}
	if got := appState(exited); got != "exited" {
		t.Errorf("appState(exited) = %q, want exited", got)
	}

	// Empty: returns "unknown".
	if got := appState(nil); got != "unknown" {
		t.Errorf("appState(nil) = %q, want unknown", got)
	}
}

func TestFormatPortsFromFixture(t *testing.T) {
	list := loadTestContainers(t)

	// Jellyfin: 0.0.0.0:8096->80/tcp.
	var jellyfin Container
	for _, c := range list {
		if len(c.Names) > 0 && c.Names[0] == "jellyfin" {
			jellyfin = c
			break
		}
	}
	got := formatPorts(jellyfin.Ports)
	want := "0.0.0.0:8096->80/tcp"
	if got != want {
		t.Errorf("formatPorts(jellyfin) = %q, want %q", got, want)
	}

	// Traefik: two ports.
	var traefik Container
	for _, c := range list {
		if len(c.Names) > 0 && c.Names[0] == "traefik" {
			traefik = c
			break
		}
	}
	got = formatPorts(traefik.Ports)
	// Port order depends on API response.
	if got == "" {
		t.Error("formatPorts(traefik) is empty")
	}

	// Container without ports.
	var redis Container
	for _, c := range list {
		if len(c.Names) > 0 && c.Names[0] == "redis" {
			redis = c
			break
		}
	}
	if got := formatPorts(redis.Ports); got != "" {
		t.Errorf("formatPorts(redis) = %q, want empty", got)
	}
}

func TestFormatExposedPortsFromFixture(t *testing.T) {
	list := loadTestContainers(t)

	// Jellyfin has ExposedPorts: {"80": ["tcp"]}.
	var jellyfin Container
	for _, c := range list {
		if len(c.Names) > 0 && c.Names[0] == "jellyfin" {
			jellyfin = c
			break
		}
	}
	got := formatExposedPorts(jellyfin.ExposedPorts)
	if got != "80/tcp" {
		t.Errorf("formatExposedPorts(jellyfin) = %q, want 80/tcp", got)
	}

	// Redis has no exposed ports.
	var redis Container
	for _, c := range list {
		if len(c.Names) > 0 && c.Names[0] == "redis" {
			redis = c
			break
		}
	}
	if got := formatExposedPorts(redis.ExposedPorts); got != "" {
		t.Errorf("formatExposedPorts(redis) = %q, want empty", got)
	}
}

func TestShortIDFromFixture(t *testing.T) {
	list := loadTestContainers(t)
	for _, c := range list {
		got := shortID(c.ID)
		if len(got) != 12 {
			t.Errorf("shortID(%s) length = %d, want 12", c.Names, len(got))
		}
		if got != c.ID[:12] {
			t.Errorf("shortID(%s) = %q, want %q", c.Names, got, c.ID[:12])
		}
	}

	// With sha256: prefix.
	got := shortID("sha256:" + list[0].ID)
	if got != list[0].ID[:12] {
		t.Errorf("shortID(sha256:...) = %q, want %q", got, list[0].ID[:12])
	}
}
