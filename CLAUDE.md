# CLAUDE.md

## Project overview

podfather is a simple web dashboard for rootless Podman. Single Go binary, no JavaScript, no external dependencies. Module path: `jo-m.ch/go/podfather`.

## Build and run

```bash
go build -o podfather .
./podfather
```

Requires the Podman API socket to be running (`systemctl --user start podman.socket` or `podman system service`).

## Architecture

The app connects to the Podman REST API over a Unix socket using Go stdlib `net/http`.

- `main.go` — Entry point: server setup and routing.
- `types.go` — Podman API response structs and app-layer types (`App`, `AppCategory`). `ContainerConfig.Env` is intentionally omitted so env vars are never parsed.
- `podman.go` — Podman API client: socket path resolution, HTTP-over-Unix-socket client, `podmanGet` helper.
- `handlers.go` — Template embed/rendering, template helper functions, all HTTP handlers. `buildAppCategories` extracts containers with `ch.jo-m.go.podfather.app.*` labels, groups by name, and sorts by category/sort-index.
- `templates/` — Go `html/template` files embedded via `go:embed`. `base.html` defines the layout with a `{{block "content"}}` slot; page templates define `"content"`.

## Key conventions

- **No JavaScript.** All rendering is server-side via Go templates.
- **No external dependencies.** Only Go stdlib. Do not add third-party modules.
- **No secrets in UI.** The `Env` field is omitted from `ContainerConfig`. Do not add it or any other field that could expose secrets.
- **Podman API version** is `v4.0.0` in the URL path (compatible with Podman v4+).
- **Apps view** at (`GET /apps`). Containers with `ch.jo-m.go.podfather.app.*` (`const appLabelPrefix` in `types.go`) labels are grouped into app cards by name, organized by category. External (non-container) apps can also be defined via `PODFATHER_APP_<KEY>_<FIELD>` env vars (parsed in `parseExternalApps()` in `handlers.go`).
- **Auto-update** is done via `exec.Command("podman", "auto-update")`, not the REST API. Disabled by default; enable with `ENABLE_AUTOUPDATE_BUTTON=true`.
- **CSS** is inline in `templates/base.html`. No CSS framework. Keep it minimal.
- **Error handling.** Log errors server-side with `log.Printf` and return minimal error messages (e.g. "Internal Server Error") to the client without exposing details.
- **Formatting.** Always run `gofmt -w` on all edited `.go` files after making changes
- **Tests.** Run with `go test ./...` after making changes.
- **Config and usage changes:** When changing ENV vars, CLI flags, etc. always update accordingly 1. README.md 2. systemd unit file (`support/podfather.service`) 3. docker-compose files (`support/docker-compose.yml`, `support/docker-compose.demo.yml`) 4. this CLAUDE.md.

## Testing

Test fixtures live in `testdata/` (raw Podman API JSON responses). Tests cover data transformation functions (`buildAppCategories`, `appState`, `formatPorts`, etc.) using real API data.

## Dependency pinning

All external dependencies are pinned to exact versions/digests for reproducibility and supply-chain security:

- **GitHub Actions** in `.github/workflows/*.yml` are pinned to full commit SHAs with a `# vX` tag comment. Renovate updates these automatically.
- **Docker images** in `Dockerfile` and `support/docker-compose.demo.yml` are pinned to `tag@sha256:digest`. The project's own image (`ghcr.io/jo-m/podfather:latest`) in `support/docker-compose.yml` is intentionally left unpinned.
- **GoReleaser** version in `release.yml` is pinned to an exact version (not `~> v2`).
- **govulncheck** runs in CI (`ci.yml`) to detect Go vulnerabilities. It is installed as a CI tool, not added to `go.mod`.
- **Renovate** (`renovate.json`) is configured to create weekly update PRs for all pinned dependencies. Requires the [Mend Renovate GitHub App](https://github.com/apps/renovate) to be installed on the repo.

When adding new GitHub Actions, Docker images, or tool versions, always pin them to an exact SHA/digest. Renovate will handle keeping them up to date.



You run on nixos use nix-shell -p to load programs if you need them.
