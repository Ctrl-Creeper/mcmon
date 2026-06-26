# mc-latency-monitor

A Minecraft Java Edition server latency monitor. Pings target servers using
the vanilla status/ping protocol, stores min/median/max latency and packet
loss per probe burst in SQLite, and serves a small built-in web UI with
graphs. Servers, their ping period, and probe settings are fully editable at
runtime from the UI — no config file editing required.

This is a rewrite of an earlier PHP + RRDtool prototype, in Go, as a single
self-contained binary with no external dependencies (no PHP, no rrdtool,
pure-Go SQLite driver). The binary cross-compiles for macOS, Windows, and
Linux, and the UI is just a browser tab, so "the app" runs anywhere Go can
target.

## Run

```sh
go run ./cmd/mcmon
```

On first run it creates `config.json` next to the binary with a placeholder
target. Then open http://localhost:8090 and use the **Servers** panel to add,
edit, or delete monitored servers — each one has its own ping period
("interval_sec"), probe burst size, timeout, and inter-probe gap. Changes are
applied immediately (the prober for that target restarts) and persisted back
to `config.json`.

## Run in the background

To keep the monitor running across reboots/logins without a terminal open:

```sh
go run ./cmd/mcmon install     # registers + starts a background service
go run ./cmd/mcmon uninstall   # stops + removes it
```

This uses the platform's native mechanism, working from the directory the
binary is in (so `config.json`/`mcmon.db` live there):

- **macOS** — a `launchd` user agent (`~/Library/LaunchAgents/com.lewiswu.mcmon.plist`), `RunAtLoad` + `KeepAlive`.
- **Linux** — a `systemd --user` unit (`~/.config/systemd/user/mcmon.service`). On a headless box, also run `loginctl enable-linger $USER` so it starts without an active login session.
- **Windows** — a Scheduled Task (`schtasks`) that runs at logon.

Run `install`/`uninstall` using the actual built binary (not `go run`, which
would register the temporary build wrapper) once you've built one — see below.

## Build for other platforms

```sh
GOOS=windows GOARCH=amd64 go build -o dist/mcmon.exe ./cmd/mcmon
GOOS=linux   GOARCH=amd64 go build -o dist/mcmon-linux ./cmd/mcmon
GOOS=darwin  GOARCH=arm64 go build -o dist/mcmon-mac ./cmd/mcmon
```

## API

- `GET /api/targets` — list configured servers
- `POST /api/targets` — add a server (JSON body: id, name, host, port, interval_sec, probes_per_burst, timeout_ms, probe_gap_ms, protocol_version)
- `PUT /api/targets/{id}` — edit a server (same body; restarts its probe loop)
- `DELETE /api/targets/{id}` — remove a server (stops probing; history stays in `mcmon.db`)
- `GET /api/series?target={id}&range={1h|6h|12h|1d|7d|30d}` — latency/loss series for graphing

## Status

Draft / proof of concept:

- [x] MC status+ping protocol round trip
- [x] Configurable per-server ping period + burst sampling (min/median/max/loss)
- [x] SQLite storage
- [x] Web UI: add/edit/delete servers, Chart.js graphs (min/max band, median, loss %)
- [x] Background running via launchd / systemd / Scheduled Task
- [ ] Native desktop wrapper (currently a server + browser UI)
- [ ] Mobile app
- [ ] Alerting / thresholds
