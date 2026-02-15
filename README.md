# podview

A simple web dashboard for rootless Podman. Single binary, no JavaScript, no external dependencies.

Connects to the local Podman API socket and renders container and image information server-side using Go templates. Designed to run as an unprivileged user behind a reverse proxy (no built-in auth).

## Features

- List all containers with state, image, ports, and creation time
- Inspect container details (command, mounts, ports, labels, restart policy, health)
- List images with tags and sizes
- Trigger `podman auto-update`
- Environment variables and secrets are never displayed

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
