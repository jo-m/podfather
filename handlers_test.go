package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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
	if jellyfin.Labels["ch.jo-m.go.podfather.app.name"] != "Jellyfin" {
		t.Errorf("jellyfin app name label = %q", jellyfin.Labels["ch.jo-m.go.podfather.app.name"])
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

func loadTestFixture(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// newMockPodmanAPI creates an httptest.Server that mocks the Podman REST API,
// serving test fixtures from testdata/.
func newMockPodmanAPI(t *testing.T) *httptest.Server {
	t.Helper()
	containers := loadTestFixture(t, "testdata/containers.json")
	containerInspect := loadTestFixture(t, "testdata/container_inspect.json")
	images := loadTestFixture(t, "testdata/images.json")
	imageInspect := loadTestFixture(t, "testdata/image_inspect.json")

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		w.Header().Set("Content-Type", "application/json")

		switch {
		case p == "/v4.0.0/libpod/containers/json":
			w.Write(containers)
		case strings.HasSuffix(p, "/json") && strings.HasPrefix(p, "/v4.0.0/libpod/containers/"):
			// /v4.0.0/libpod/containers/{id}/json
			id := strings.TrimPrefix(p, "/v4.0.0/libpod/containers/")
			id = strings.TrimSuffix(id, "/json")
			if id == "nonexistent" {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{}`))
				return
			}
			w.Write(containerInspect)
		case p == "/v4.0.0/libpod/images/json":
			w.Write(images)
		case strings.HasSuffix(p, "/json") && strings.HasPrefix(p, "/v4.0.0/libpod/images/"):
			// /v4.0.0/libpod/images/{id}/json
			id := strings.TrimPrefix(p, "/v4.0.0/libpod/images/")
			id = strings.TrimSuffix(id, "/json")
			if id == "nonexistent" {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{}`))
				return
			}
			w.Write(imageInspect)
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{}`))
		}
	}))
}

func TestEndToEnd(t *testing.T) {
	// Save and restore globals.
	origClient := podman
	origBaseURL := podmanBaseURL
	origBasePath := basePath
	origAutoUpdate := enableAutoUpdate
	origExtApps := externalApps
	t.Cleanup(func() {
		podman = origClient
		podmanBaseURL = origBaseURL
		basePath = origBasePath
		enableAutoUpdate = origAutoUpdate
		externalApps = origExtApps
	})

	// Start mock Podman API.
	mock := newMockPodmanAPI(t)
	defer mock.Close()

	podman = mock.Client()
	podmanBaseURL = mock.URL + "/v4.0.0/libpod"
	basePath = ""
	enableAutoUpdate = false
	externalApps = nil

	// Start app server.
	app := httptest.NewServer(newMux("podman"))
	defer app.Close()

	// Client that does not follow redirects.
	noRedirect := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   string // substring to look for in body (empty = just check non-empty)
	}{
		{"root redirects to apps", "GET", "/", http.StatusTemporaryRedirect, ""},
		{"apps page", "GET", "/apps", http.StatusOK, "Jellyfin"},
		{"containers page", "GET", "/containers", http.StatusOK, "jellyfin"},
		{"container detail", "GET", "/container/jellyfin", http.StatusOK, "jellyfin"},
		{"container not found", "GET", "/container/nonexistent", http.StatusNotFound, ""},
		{"container invalid id", "GET", "/container/!!!invalid", http.StatusBadRequest, ""},
		{"images page", "GET", "/images", http.StatusOK, "nginx"},
		{"image detail", "GET", "/image/b76de378d572", http.StatusOK, "nginx"},
		{"image not found", "GET", "/image/nonexistent", http.StatusNotFound, ""},
		{"auto-update disabled", "POST", "/auto-update", http.StatusNotFound, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp *http.Response
			var err error

			url := app.URL + tt.path
			if tt.method == "POST" {
				resp, err = noRedirect.Post(url, "", nil)
			} else {
				resp, err = noRedirect.Get(url)
			}
			if err != nil {
				t.Fatalf("request %s %s: %v", tt.method, tt.path, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			if tt.wantBody != "" {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("read body: %v", err)
				}
				if !strings.Contains(string(body), tt.wantBody) {
					t.Errorf("body does not contain %q (len=%d)", tt.wantBody, len(body))
				}
			}
		})
	}
}

func TestEndToEndAutoUpdate(t *testing.T) {
	// Save and restore globals.
	origClient := podman
	origBaseURL := podmanBaseURL
	origBasePath := basePath
	origAutoUpdate := enableAutoUpdate
	origExtApps := externalApps
	t.Cleanup(func() {
		podman = origClient
		podmanBaseURL = origBaseURL
		basePath = origBasePath
		enableAutoUpdate = origAutoUpdate
		externalApps = origExtApps
	})

	mock := newMockPodmanAPI(t)
	defer mock.Close()

	podman = mock.Client()
	podmanBaseURL = mock.URL + "/v4.0.0/libpod"
	basePath = ""
	enableAutoUpdate = true
	externalApps = nil

	// Pass "true" as podman binary — a no-op that exits 0.
	app := httptest.NewServer(newMux("true"))
	defer app.Close()

	resp, err := http.Post(app.URL+"/auto-update", "", nil)
	if err != nil {
		t.Fatalf("POST /auto-update: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Error("empty response body")
	}
}

func TestParseExternalApps(t *testing.T) {
	envs := map[string]string{
		"PODFATHER_APP_ROUTER_NAME":        "Router",
		"PODFATHER_APP_ROUTER_URL":         "http://192.168.1.1",
		"PODFATHER_APP_ROUTER_ICON":        "📡",
		"PODFATHER_APP_ROUTER_CATEGORY":    "Infrastructure",
		"PODFATHER_APP_ROUTER_SORT_INDEX":  "5",
		"PODFATHER_APP_ROUTER_SUBTITLE":    "Network Router",
		"PODFATHER_APP_ROUTER_DESCRIPTION": "Router admin interface",
		"PODFATHER_APP_NAS_NAME":           "NAS",
		"PODFATHER_APP_NAS_URL":            "http://192.168.1.2",
		// Key with underscores.
		"PODFATHER_APP_MY_APP_NAME": "My App",
		"PODFATHER_APP_MY_APP_URL":  "http://example.com",
		// Missing NAME — should be skipped.
		"PODFATHER_APP_NONAME_URL": "http://skip.me",
		// Empty key — should be skipped.
		"PODFATHER_APP__NAME": "BadKey",
	}
	for k, v := range envs {
		t.Setenv(k, v)
	}

	apps := parseExternalApps()

	// Should have 3 apps: Router, NAS, My App (NONAME and empty key skipped).
	if len(apps) != 3 {
		t.Fatalf("got %d apps, want 3", len(apps))
	}

	byName := make(map[string]App)
	for _, a := range apps {
		byName[a.Name] = a
	}

	router, ok := byName["Router"]
	if !ok {
		t.Fatal("Router app not found")
	}
	if router.URL != "http://192.168.1.1" {
		t.Errorf("Router URL = %q", router.URL)
	}
	if router.Icon != "📡" {
		t.Errorf("Router Icon = %q", router.Icon)
	}
	if router.Category != "Infrastructure" {
		t.Errorf("Router Category = %q", router.Category)
	}
	if router.SortIndex != 5 {
		t.Errorf("Router SortIndex = %d, want 5", router.SortIndex)
	}
	if router.Subtitle != "Network Router" {
		t.Errorf("Router Subtitle = %q", router.Subtitle)
	}
	if router.Description != "Router admin interface" {
		t.Errorf("Router Description = %q", router.Description)
	}
	if len(router.Containers) != 0 {
		t.Errorf("Router Containers = %d, want 0", len(router.Containers))
	}

	nas, ok := byName["NAS"]
	if !ok {
		t.Fatal("NAS app not found")
	}
	if nas.URL != "http://192.168.1.2" {
		t.Errorf("NAS URL = %q", nas.URL)
	}

	myApp, ok := byName["My App"]
	if !ok {
		t.Fatal("My App not found (key with underscores)")
	}
	if myApp.URL != "http://example.com" {
		t.Errorf("My App URL = %q", myApp.URL)
	}
}

func TestBuildAppCategoriesWithExternalApps(t *testing.T) {
	origExtApps := externalApps
	t.Cleanup(func() { externalApps = origExtApps })

	externalApps = []App{
		{
			Name:     "Router",
			Icon:     "📡",
			Category: "Infrastructure",
			URL:      "http://192.168.1.1",
		},
		{
			Name:     "Wiki",
			Icon:     "📖",
			Category: "Docs",
			URL:      "http://wiki.example.com",
		},
	}

	list := loadTestContainers(t)
	categories := buildAppCategories(list)

	// Should now have: Docs, Infrastructure, Media, Monitoring, Uncategorized.
	wantCats := []string{"Docs", "Infrastructure", "Media", "Monitoring", "Uncategorized"}
	if len(categories) != len(wantCats) {
		var got []string
		for _, c := range categories {
			got = append(got, c.Name)
		}
		t.Fatalf("got categories %v, want %v", got, wantCats)
	}
	for i, want := range wantCats {
		if categories[i].Name != want {
			t.Errorf("category[%d] = %q, want %q", i, categories[i].Name, want)
		}
	}

	// Infrastructure should have Traefik, Gitea, Router.
	var infra AppCategory
	for _, c := range categories {
		if c.Name == "Infrastructure" {
			infra = c
			break
		}
	}
	if len(infra.Apps) != 3 {
		t.Fatalf("Infrastructure has %d apps, want 3", len(infra.Apps))
	}
	// Router has no containers.
	var router *App
	for i := range infra.Apps {
		if infra.Apps[i].Name == "Router" {
			router = &infra.Apps[i]
			break
		}
	}
	if router == nil {
		t.Fatal("Router not found in Infrastructure")
	}
	if len(router.Containers) != 0 {
		t.Errorf("Router has %d containers, want 0", len(router.Containers))
	}
	if router.URL != "http://192.168.1.1" {
		t.Errorf("Router URL = %q", router.URL)
	}

	// Wiki should be in Docs category.
	var docs AppCategory
	for _, c := range categories {
		if c.Name == "Docs" {
			docs = c
			break
		}
	}
	if len(docs.Apps) != 1 || docs.Apps[0].Name != "Wiki" {
		t.Errorf("Docs category = %v, want [Wiki]", docs.Apps)
	}
}

func TestExternalAppContainerPriority(t *testing.T) {
	origExtApps := externalApps
	t.Cleanup(func() { externalApps = origExtApps })

	// External app with same name as a container app — container should take priority.
	externalApps = []App{
		{
			Name:     "Jellyfin",
			URL:      "http://external.example.com",
			Category: "External",
		},
	}

	list := loadTestContainers(t)
	categories := buildAppCategories(list)

	// Jellyfin should still be in Media (from container labels), not External.
	var jellyfin *App
	for _, cat := range categories {
		for i := range cat.Apps {
			if cat.Apps[i].Name == "Jellyfin" {
				jellyfin = &cat.Apps[i]
			}
		}
	}
	if jellyfin == nil {
		t.Fatal("Jellyfin not found")
	}
	if jellyfin.Category != "Media" {
		t.Errorf("Jellyfin category = %q, want Media (container takes priority)", jellyfin.Category)
	}
	if jellyfin.URL != "http://localhost:8096" {
		t.Errorf("Jellyfin URL = %q, want container URL", jellyfin.URL)
	}
	if len(jellyfin.Containers) == 0 {
		t.Error("Jellyfin should have containers (from container labels)")
	}

	// No "External" category should exist.
	for _, cat := range categories {
		if cat.Name == "External" {
			t.Error("External category should not exist (container takes priority)")
		}
	}
}

func TestEndToEndExternalApps(t *testing.T) {
	// Save and restore globals.
	origClient := podman
	origBaseURL := podmanBaseURL
	origBasePath := basePath
	origAutoUpdate := enableAutoUpdate
	origExtApps := externalApps
	t.Cleanup(func() {
		podman = origClient
		podmanBaseURL = origBaseURL
		basePath = origBasePath
		enableAutoUpdate = origAutoUpdate
		externalApps = origExtApps
	})

	mock := newMockPodmanAPI(t)
	defer mock.Close()

	podman = mock.Client()
	podmanBaseURL = mock.URL + "/v4.0.0/libpod"
	basePath = ""
	enableAutoUpdate = false
	externalApps = []App{
		{
			Name:     "Router",
			Icon:     "📡",
			Category: "Infrastructure",
			URL:      "http://192.168.1.1",
			Subtitle: "Network Router",
		},
	}

	app := httptest.NewServer(newMux("podman"))
	defer app.Close()

	noRedirect := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Root should redirect to /apps (external apps present).
	resp, err := noRedirect.Get(app.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusTemporaryRedirect {
		t.Errorf("GET / status = %d, want %d", resp.StatusCode, http.StatusTemporaryRedirect)
	}

	// Apps page should contain the external app.
	resp, err = http.Get(app.URL + "/apps")
	if err != nil {
		t.Fatalf("GET /apps: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /apps status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "Router") {
		t.Error("apps page does not contain external app 'Router'")
	}
	if !strings.Contains(bodyStr, "📡") {
		t.Error("apps page does not contain Router icon")
	}
	if !strings.Contains(bodyStr, "http://192.168.1.1") {
		t.Error("apps page does not contain Router URL")
	}
	// Should also still contain container-based apps.
	if !strings.Contains(bodyStr, "Jellyfin") {
		t.Error("apps page does not contain container app 'Jellyfin'")
	}
}
