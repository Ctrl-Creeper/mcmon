# mc-latency-monitor

A Minecraft Java Edition server latency monitor. Pings target servers using
the vanilla status/ping protocol, stores per-minute min/median/max latency
and packet loss in SQLite, and serves a small built-in web UI with graphs.

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
target — edit it to point at real servers, then restart:

```json
{
  "listen_addr": ":8090",
  "db_path": "mcmon.db",
  "targets": [
    {
      "id": "hypixel",
      "name": "Hypixel",
      "host": "mc.hypixel.net",
      "port": 25565,
      "timeout_ms": 1500,
      "probes_per_minute": 5,
      "probe_interval_ms": 1500,
      "protocol_version": 760
    }
  ]
}
```

Then open http://localhost:8090.

## Build for other platforms

```sh
GOOS=windows GOARCH=amd64 go build -o dist/mcmon.exe ./cmd/mcmon
GOOS=linux   GOARCH=amd64 go build -o dist/mcmon-linux ./cmd/mcmon
GOOS=darwin  GOARCH=arm64 go build -o dist/mcmon-mac ./cmd/mcmon
```

## Status

Draft / proof of concept:

- [x] MC status+ping protocol round trip
- [x] Per-minute multi-probe sampling with min/median/max/loss
- [x] SQLite storage
- [x] Web UI with Chart.js graphs (min/max band, median, loss %)
- [ ] Native desktop wrapper (currently a server + browser UI)
- [ ] Mobile app
- [ ] Alerting / thresholds
