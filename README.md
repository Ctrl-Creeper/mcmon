# mcmon

[English](README.md) | [简体中文](README.zh-CN.md)

mcmon is a lightweight desktop monitor for Minecraft Java Edition
servers. It probes servers with the vanilla status/ping protocol, stores
server status history in SQLite, and shows the data in a bundled desktop UI.

The app can also run as a small background service so monitoring continues
after the window is closed.

## Related Projects

This desktop app is the standalone/local member of the mcmon project family.

- `mcmon`: this app. Use it when you want a local desktop monitor
  that works without any server.
- [mcmon-host](https://github.com/Ctrl-Creeper/mcmon-host): Linux-only central dashboard and API for managed monitoring.
  It configures nodes and generates one-line `mcmon-agent` install commands.
- [mcmon-agent](https://github.com/Ctrl-Creeper/mcmon-agent): lightweight cross-platform node process with no UI. It reports
  to `mcmon-host`.

You do not need `mcmon-host` to use this desktop app. Configure a remote host
only when you want the app to view or proxy data from a central deployment.

## Features

- Native desktop app powered by Wails.
- Minecraft Java Edition status/ping probing.
- Per-metric enable switches and probe intervals, with latency-specific burst settings.
- SQLite history storage.
- Metric charts for online state, players, latency, and packet loss.
- Local-only default bind address: `127.0.0.1:8090`.
- Optional remote host integration for pairing with `mcmon-host`.
- Background mode:
  - macOS: user `launchd` agent.
  - Linux: user `systemd` service.
  - Windows: Scheduled Task at logon.

## Requirements

For development and local builds:

- Go 1.25.4 or newer compatible Go toolchain.
- Node.js/npm, required by Wails even though this project has no frontend build
  step.
- Wails v2.10.2, invoked through `go run` by the build scripts.

Platform-specific desktop build requirements:

- macOS: Xcode Command Line Tools.
- Windows: WebView2 runtime. The Windows build script uses Wails'
  `-webview2 download` option.
- Linux: GTK/WebKitGTK development packages. On Ubuntu 24.04 or newer:

```sh
sudo apt-get update
sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.1-dev pkg-config
```

On Ubuntu 22.04 or older Debian-based systems, use the WebKitGTK 4.0 package
instead:

```sh
sudo apt-get update
sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.0-dev pkg-config
```

The build script detects the installed WebKitGTK pkg-config package and passes
the matching Wails build tag automatically.

## Run From Source

Desktop app:

```sh
go run .
```

CLI/server mode:

```sh
go run ./cmd/mcmon
```

The CLI server listens on `127.0.0.1:8090` by default. Open:

```text
http://127.0.0.1:8090
```

## Build The Desktop App

### macOS and Linux

```sh
./scripts/build-desktop.sh
```

On macOS, the script also performs local ad-hoc signing so the generated app can
be opened on your own machine:

```text
build/bin/mcmon.app
```

### Windows

Run in PowerShell:

```powershell
.\scripts\build-desktop.ps1
```

The output is written under:

```text
build/bin/
```

### Build Options

Both scripts pass extra arguments through to `wails build`.

Examples:

```sh
./scripts/build-desktop.sh -debug
./scripts/build-desktop.sh -platform darwin/arm64
```

```powershell
.\scripts\build-desktop.ps1 -debug
```

## Build The CLI Binary

The CLI binary is useful for server-only deployments or manual background
service installation.

```sh
go build -o dist/mcmon ./cmd/mcmon
```

Cross-compile examples:

```sh
GOOS=windows GOARCH=amd64 go build -o dist/mcmon.exe ./cmd/mcmon
GOOS=linux   GOARCH=amd64 go build -o dist/mcmon-linux ./cmd/mcmon
GOOS=darwin  GOARCH=arm64 go build -o dist/mcmon-mac ./cmd/mcmon
```

## Background Mode

In the desktop app, open Settings and enable "Run in background".

From the CLI:

```sh
mcmon install -config /path/to/config.json
mcmon uninstall
```

The installed background service starts the same binary in lightweight mode:

```sh
serve -config /path/to/config.json
```

That means the GUI is not started by the background service.

Platform details:

- macOS: `~/Library/LaunchAgents/com.mcmon.plist`
- Linux: `~/.config/systemd/user/mcmon.service`
- Windows: Scheduled Task named `mcmon`

On headless Linux systems, enable user services after logout:

```sh
loginctl enable-linger "$USER"
```

## Remote Host Integration

The desktop app can be used standalone. It can also connect to an `mcmon-host`
instance from the Remote view.

Remote settings support:

- Host URL.
- Optional admin token.
- Bearer token forwarding to the host API.

If no remote host is configured, local monitoring continues to work normally.

## API

The local server exposes:

- `GET /api/targets`
- `POST /api/targets`
- `PUT /api/targets/{id}`
- `DELETE /api/targets/{id}`
- `GET /api/series?target={id}&range={1h|6h|12h|1d|7d|30d}`
- `GET /api/settings/background`
- `POST /api/settings/background`
- `GET /api/remote/config`
- `POST /api/remote/config`
- `GET /api/remote/*`

## Development Checks

Run tests:

```sh
go test ./...
```

Check Wails environment:

```sh
go run github.com/wailsapp/wails/v2/cmd/wails@v2.10.2 doctor
```

Cross-platform compile checks from macOS:

```sh
GOOS=linux GOARCH=amd64 go test -exec /usr/bin/true ./...
GOOS=windows GOARCH=amd64 go test -exec /usr/bin/true ./...
```

Full desktop packaging should be done on the native target platform. Wails does
not currently support Linux desktop packaging from macOS.

## CI

The GitHub Actions workflow in `.github/workflows/desktop.yml` builds on native
platform runners:

- macOS app bundle.
- Windows desktop app.
- Linux desktop binary.
- Go tests on Ubuntu.
