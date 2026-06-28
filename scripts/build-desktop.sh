#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

ARGS=("$@")

add_wails_tag() {
  local tag="$1"
  local i
  for ((i = 0; i < ${#ARGS[@]}; i++)); do
    case "${ARGS[$i]}" in
      -tags)
        if (( i + 1 < ${#ARGS[@]} )); then
          ARGS[$((i + 1))]="${ARGS[$((i + 1))]},${tag}"
          return
        fi
        ;;
      -tags=*)
        ARGS[$i]="${ARGS[$i]},${tag}"
        return
        ;;
    esac
  done
  ARGS+=("-tags" "$tag")
}

if [[ "$(uname -s)" == "Linux" ]]; then
  if pkg-config --exists webkit2gtk-4.1; then
    add_wails_tag "webkit2_41"
  elif ! pkg-config --exists webkit2gtk-4.0; then
    cat >&2 <<'EOF'
Missing WebKitGTK development files.

Install one of these package sets, then run the build again:

  Ubuntu 24.04+:
    sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.1-dev pkg-config

  Ubuntu 22.04/Debian with WebKitGTK 4.0:
    sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.0-dev pkg-config
EOF
    exit 1
  fi
fi

go run github.com/wailsapp/wails/v2/cmd/wails@v2.10.2 build -clean "${ARGS[@]}"

if [[ "$(uname -s)" == "Darwin" ]]; then
  APP="$ROOT/build/bin/mc-latency-monitor.app"
  if [[ ! -d "$APP" ]]; then
    APP="$ROOT/build/bin/MC Server Monitor.app"
  fi
  /usr/bin/codesign --force --deep --sign - "$APP"
  /usr/bin/codesign --verify --deep --strict --verbose=2 "$APP"
fi
