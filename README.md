# podfather

A simple web dashboard for Podman.
Single binary, no JavaScript, no external dependencies.
Ideal as landing page on your self-hosted server.

## Features

- Apps dashboard - start page showing containers as application cards, grouped by category
- List all containers with state, image, ports, and creation time
- Inspect container details (command, mounts, ports, labels, restart policy, health)
- Read-only except for `podman auto-update` (off by default).
- List images with tags and sizes
- Trigger `podman auto-update`
- Environment variables and secrets are never displayed
- **No auth, handle this by running behind a reverse proxy if you host it publicly.**

## App labels

Containers with labels prefixed `ch.jo-m.go.podfather.app.` appear as apps on the start page.
Multiple containers sharing the same `name` displayed on the same app card.

| Label | Required | Description | Example |
|---|---|---|---|
| `ch.jo-m.go.podfather.app.name` | **yes** | App name (used for grouping) | `Nextcloud` |
| `ch.jo-m.go.podfather.app.icon` | no | Emoji icon | `☁️` |
| `ch.jo-m.go.podfather.app.category` | no | Category heading (default: "Uncategorized") | `Productivity` |
| `ch.jo-m.go.podfather.app.sort-index` | no | Sort order within category (default: 0) | `10` |
| `ch.jo-m.go.podfather.app.subtitle` | no | Short subtitle | `Cloud storage` |
| `ch.jo-m.go.podfather.app.description` | no | Longer description | `Self-hosted file sync and share` |
| `ch.jo-m.go.podfather.app.url` | no | URL opened when clicking the card | `https://cloud.example.com` |

Example:

```
podman run -d \
  --label ch.jo-m.go.podfather.app.name=Nextcloud \
  --label ch.jo-m.go.podfather.app.icon=☁️ \
  --label ch.jo-m.go.podfather.app.category=Productivity \
  --label ch.jo-m.go.podfather.app.sort-index=10 \
  --label ch.jo-m.go.podfather.app.subtitle="Cloud storage" \
  --label ch.jo-m.go.podfather.app.url=https://cloud.example.com \
  nextcloud:latest
```

## External apps

You can also add apps to the dashboard that are not running as Podman containers (e.g. a network router, NAS, or external service). Define them via environment variables using the pattern `PODFATHER_APP_<KEY>_<FIELD>`, where `<KEY>` is any unique identifier (may contain underscores) and `<FIELD>` is one of:

| Field | Required | Description | Example |
|---|---|---|---|
| `NAME` | **yes** | App name | `Router` |
| `ICON` | no | Emoji icon | `📡` |
| `CATEGORY` | no | Category heading (default: "Uncategorized") | `Infrastructure` |
| `SORT_INDEX` | no | Sort order within category (default: 0) | `10` |
| `SUBTITLE` | no | Short subtitle | `Network Router` |
| `DESCRIPTION` | no | Longer description | `Router admin interface` |
| `URL` | no | URL opened when clicking the card | `http://192.168.1.1` |

Example:

```bash
export PODFATHER_APP_ROUTER_NAME=Router
export PODFATHER_APP_ROUTER_ICON=📡
export PODFATHER_APP_ROUTER_CATEGORY=Infrastructure
export PODFATHER_APP_ROUTER_URL=http://192.168.1.1
export PODFATHER_APP_ROUTER_SUBTITLE="Network Router"
```

If an external app has the same name as a container-based app, the container-based app takes priority.

## Building, Installation, and Usage

```bash
# For rootless Podman, enable the socket with
systemctl --user enable --now podman.socket

# Build and run
export ENABLE_AUTOUPDATE_BUTTON=true
go build -o podfather .
./podfather
```

The server starts on `127.0.0.1:8080` (localhost only) by default and connects to the rootless Podman socket.

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `LISTEN_ADDR` | `127.0.0.1:8080` | HTTP listen address |
| `PODMAN_SOCKET` | `$XDG_RUNTIME_DIR/podman/podman.sock` | Path to the Podman API socket |
| `BASE_PATH` | _(none)_ | URL path prefix for hosting at a subpath (e.g. `/podfather`), no trailing slash |
| `ENABLE_AUTOUPDATE_BUTTON` | _(none)_ | Set to `true` to allow triggering `podman auto-update` from the web UI |

### Running as a systemd user service

A sample unit file is provided in [`support/podfather.service`](support/podfather.service). Install it with:

```bash
cp support/podfather.service ~/.config/systemd/user/
systemctl --user enable --now podfather
```

### Running with Docker (Compose)

A sample compose file is provided:

```bash
cd support
podman-compose up -d
```

A demo compose file with fake app containers is also available:

```bash
cd support
podman-compose -f docker-compose.demo.yml up -d
```

## Creating releases

1. Go to the [GitHub Releases page](https://github.com/jo-m/podfather/releases) and click **Draft a new release**.
2. Create a new tag (e.g. `v1.2.0`).
3. Fill in the release title and description, then click **Publish release**.
4. The [Release workflow](.github/workflows/release.yml) will automatically build binaries, Docker images, and attach them to the release.
