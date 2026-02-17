# podview

A simple web dashboard for rootless Podman. Single binary, no JavaScript, no external dependencies.

Connects to the local Podman API socket and renders container and image information server-side using Go templates. Designed to run as an unprivileged user behind a reverse proxy (no built-in auth).

## Features

- **Apps dashboard** — start page showing containers as application cards, grouped by category
- List all containers with state, image, ports, and creation time
- Inspect container details (command, mounts, ports, labels, restart policy, health)
- List images with tags and sizes
- Trigger `podman auto-update`
- Environment variables and secrets are never displayed

## App labels

Containers with labels prefixed `ch.jo-m.go.podview.app.` appear as apps on the start page. Multiple containers sharing the same `name` are grouped into one app.

| Label | Required | Description | Example |
|---|---|---|---|
| `ch.jo-m.go.podview.app.name` | **yes** | App name (used for grouping) | `Nextcloud` |
| `ch.jo-m.go.podview.app.icon` | no | Emoji icon | `☁️` |
| `ch.jo-m.go.podview.app.category` | no | Category heading (default: "Uncategorized") | `Productivity` |
| `ch.jo-m.go.podview.app.sort-index` | no | Sort order within category (default: 0) | `10` |
| `ch.jo-m.go.podview.app.subtitle` | no | Short subtitle | `Cloud storage` |
| `ch.jo-m.go.podview.app.description` | no | Longer description | `Self-hosted file sync and share` |
| `ch.jo-m.go.podview.app.url` | no | URL opened when clicking the card | `https://cloud.example.com` |

Example:

```
podman run -d \
  --label ch.jo-m.go.podview.app.name=Nextcloud \
  --label ch.jo-m.go.podview.app.icon=☁️ \
  --label ch.jo-m.go.podview.app.category=Productivity \
  --label ch.jo-m.go.podview.app.sort-index=10 \
  --label ch.jo-m.go.podview.app.subtitle="Cloud storage" \
  --label ch.jo-m.go.podview.app.url=https://cloud.example.com \
  nextcloud:latest
```

## Requirements

- Go 1.23+
- Podman with the API socket enabled (`podman system service`)

## Build

```
go build -o podview .
```

## Usage

```
./podview
```

The server starts on `:8080` by default and connects to the rootless Podman socket.

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `LISTEN_ADDR` | `:8080` | HTTP listen address |
| `PODMAN_SOCKET` | `$XDG_RUNTIME_DIR/podman/podman.sock` | Path to the Podman API socket |
| `BASE_PATH` | _(none)_ | URL path prefix for hosting at a subpath (e.g. `/podview`), no trailing slash |

### Running as a systemd user service

A sample unit file is provided in [`podview.service`](podview.service). Install it with:

```
cp podview.service ~/.config/systemd/user/
systemctl --user enable --now podview
```

### Enabling the Podman socket

For rootless Podman, enable the socket with:

```
systemctl --user enable --now podman.socket
```
