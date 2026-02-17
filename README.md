# podview

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

Containers with labels prefixed `ch.jo-m.go.podview.app.` appear as apps on the start page.
Multiple containers sharing the same `name` displayed on the same app card.

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

## Building, Installation, and Usage

```bash
# For rootless Podman, enable the socket with
systemctl --user enable --now podman.socket

# Build and run
go build -o podview .
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

```bash
cp podview.service ~/.config/systemd/user/
systemctl --user enable --now podview
```

### Running with Docker

```bash
docker run -it --rm \
  --publish 8080:8080 \
  --mount "type=bind,source=$XDG_RUNTIME_DIR/podman/podman.sock,target=/var/run/podman.sock" \
  --env PODMAN_SOCKET=/var/run/podman.sock \
  --env LISTEN_ADDR=:8080 \
  --env BASE_PATH= \
  --user $(id -u):$(id -g) \
  ghcr.io/jo-m/podview:latest-amd64
```

## Creating releases

1. Go to the [GitHub Releases page](https://github.com/jo-m/podview/releases) and click **Draft a new release**.
2. Create a new tag (e.g. `v1.2.0`).
3. Fill in the release title and description, then click **Publish release**.
4. The [Release workflow](.github/workflows/release.yml) will automatically build binaries, Docker images, and attach them to the release.
