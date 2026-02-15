# podview

A simple web dashboard for rootless Podman. Single binary, no JavaScript, no external dependencies.

Connects to the local Podman API socket and renders container and image information server-side using Go templates. Designed to run as an unprivileged user behind a reverse proxy (no built-in auth).

## Features

- List all containers with state, image, ports, and creation time
- Inspect container details (command, mounts, ports, labels, restart policy, health)
- List images with tags and sizes
- Trigger `podman auto-update`
- Environment variables and secrets are never displayed (structurally omitted from JSON parsing)

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

### Enabling the Podman socket

For rootless Podman, enable the socket with:

```
systemctl --user enable --now podman.socket
```

Or start it temporarily:

```
podman system service --time=0 &
```

## Architecture

- `main.go` — all Go code (~320 lines): HTTP handlers, Podman API client (HTTP over Unix socket), template helpers
- `templates/` — HTML templates embedded at compile time via `go:embed`
- Zero external dependencies — stdlib only
- Talks to Podman via its REST API (libpod) over the Unix socket

### Routes

| Method | Path | Description |
|---|---|---|
| GET | `/` | Container list |
| GET | `/container/{id}` | Container detail |
| GET | `/images` | Image list |
| POST | `/auto-update` | Run `podman auto-update` |
