#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

go run github.com/wailsapp/wails/v2/cmd/wails@v2.10.2 build -clean "$@"

if [[ "$(uname -s)" == "Darwin" ]]; then
  APP="$ROOT/build/bin/mc-latency-monitor.app"
  /usr/bin/codesign --force --deep --sign - "$APP"
  /usr/bin/codesign --verify --deep --strict --verbose=2 "$APP"
fi
