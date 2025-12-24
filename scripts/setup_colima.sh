#!/usr/bin/env bash
set -euo pipefail

if [ "$(uname -s)" != "Darwin" ]; then
  echo "This setup script only supports macOS (Darwin)." >&2
  exit 1
fi

COLIMA_CPU="${COLIMA_CPU:-6}"
COLIMA_MEM="${COLIMA_MEM:-12}"
COLIMA_DISK="${COLIMA_DISK:-80}"

require_cmd() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    return 1
  fi
  return 0
}

install_if_missing() {
  local name="$1"
  local cmd="$2"
  if require_cmd "$cmd"; then
    echo "ok: $name is already installed"
    return 0
  fi
  echo "installing: $name"
  brew install "$name"
}

if ! require_cmd brew; then
  echo "Homebrew is required but not installed." >&2
  echo "Install it from https://brew.sh and re-run this script." >&2
  exit 1
fi

install_if_missing docker docker
install_if_missing colima colima
install_if_missing docker-compose docker-compose

if ! docker compose version >/dev/null 2>&1; then
  if require_cmd docker-compose; then
    mkdir -p "$HOME/.docker/cli-plugins"
    plugin_path="$(brew --prefix)/lib/docker/cli-plugins/docker-compose"
    if [ -f "$plugin_path" ]; then
      ln -sf "$plugin_path" "$HOME/.docker/cli-plugins/docker-compose"
      echo "linked docker compose plugin"
    fi
  fi
fi

if colima status >/dev/null 2>&1; then
  if colima status 2>/dev/null | grep -qi "running"; then
    echo "colima already running"
  else
    colima start --cpu "$COLIMA_CPU" --memory "$COLIMA_MEM" --disk "$COLIMA_DISK"
  fi
else
  colima start --cpu "$COLIMA_CPU" --memory "$COLIMA_MEM" --disk "$COLIMA_DISK"
fi

docker context use colima >/dev/null 2>&1 || true

echo "colima is ready"
echo "next: docker compose up --build"
