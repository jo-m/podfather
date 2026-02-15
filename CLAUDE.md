# CLAUDE.md

## Project overview

podview is a simple web dashboard for rootless Podman. Single Go binary, no JavaScript, no external dependencies. Module path: `jo-m.ch/go/podview`.

## Build and run

```
go build -o podview .
./podview
```

Requires the Podman API socket to be running (`systemctl --user start podman.socket` or `podman system service --time=0 &`).

## Architecture

The app connects to the Podman REST API (libpod) over a Unix socket using stdlib `net/http` — no podman Go module dependency.

## Key conventions

- **No JavaScript.** All rendering is server-side via Go templates.
- **No external dependencies.** Only Go stdlib. Do not add third-party modules.
- **No secrets in UI.** The `Env` field is structurally omitted from `ContainerConfig`. Do not add it or any other field that could expose secrets.
- **Podman API version** is `v4.0.0` in the URL path (compatible with Podman v4+).
- **Auto-update** is done via `exec.Command("podman", "auto-update")`, not the REST API.
- **CSS** is inline in `templates/base.html`. No CSS framework. Keep it minimal.

## Testing

No test suite. Smoke test manually:

```
LISTEN_ADDR=:18923 ./podview &
curl -s http://localhost:18923/
curl -s http://localhost:18923/images
curl -s http://localhost:18923/container/<ID>
```

## Environment variables

- `LISTEN_ADDR` — listen address (default `:8080`)
- `PODMAN_SOCKET` — override socket path (default `$XDG_RUNTIME_DIR/podman/podman.sock`)
