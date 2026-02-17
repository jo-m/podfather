# CLAUDE.md

## Project overview

podview is a simple web dashboard for rootless Podman. Single Go binary, no JavaScript, no external dependencies. Module path: `jo-m.ch/go/podview`.

## Build and run

```bash
go build -o podview .
./podview
```

Requires the Podman API socket to be running (`systemctl --user start podman.socket` or `podman system service`).

## Architecture

The app connects to the Podman REST API over a Unix socket using Go stdlib `net/http`.

- `main.go` — Entry point: server setup and routing.
- `types.go` — Podman API response structs and app-layer types (`App`, `AppCategory`). `ContainerConfig.Env` is intentionally omitted so env vars are never parsed.
- `podman.go` — Podman API client: socket path resolution, HTTP-over-Unix-socket client, `podmanGet` helper.
- `handlers.go` — Template embed/rendering, template helper functions, all HTTP handlers. `buildAppCategories` extracts containers with `ch.jo-m.go.podview.app.*` labels, groups by name, and sorts by category/sort-index.
- `templates/` — Go `html/template` files embedded via `go:embed`. `base.html` defines the layout with a `{{block "content"}}` slot; page templates define `"content"`.

## Key conventions

- **No JavaScript.** All rendering is server-side via Go templates.
- **No external dependencies.** Only Go stdlib. Do not add third-party modules.
- **No secrets in UI.** The `Env` field is omitted from `ContainerConfig`. Do not add it or any other field that could expose secrets.
- **Podman API version** is `v4.0.0` in the URL path (compatible with Podman v4+).
- **Apps view** at (`GET /apps`). Containers with `ch.jo-m.go.podview.app.*` (`const appLabelPrefix` in `types.go`) labels are grouped into app cards by name, organized by category.
- **Auto-update** is done via `exec.Command("podman", "auto-update")`, not the REST API. Disabled by default; enable with `ENABLE_AUTOUPDATE_BUTTON=true`.
- **CSS** is inline in `templates/base.html`. No CSS framework. Keep it minimal.
- **Error handling.** Log errors server-side with `log.Printf` and return minimal error messages (e.g. "Internal Server Error") to the client without exposing details.
- **Formatting.** Always run `gofmt -w` on all edited `.go` files after making changes
- **Tests.** Run with `go test ./...` after making changes.
- **Config and usage changes:** When changing ENV vars, CLI flags, etc. always update accordingly 1. README.md 2. systemd unit file 3. this CLAUDE.md.

## Testing

Test fixtures live in `testdata/` (raw Podman API JSON responses). Tests cover data transformation functions (`buildAppCategories`, `appState`, `formatPorts`, etc.) using real API data.
