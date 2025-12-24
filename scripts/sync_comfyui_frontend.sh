#!/usr/bin/env bash
set -euo pipefail

SRC_ROOT="${1:-}"
DEST_ROOT="${2:-$(pwd)/ui}"

if [ -z "$SRC_ROOT" ]; then
  echo "usage: $0 /path/to/ComfyUI/repo [dest]" >&2
  exit 1
fi

if [ ! -d "$SRC_ROOT" ]; then
  echo "source repo not found: $SRC_ROOT" >&2
  exit 1
fi

if [ -d "$SRC_ROOT/web" ]; then
  SRC="$SRC_ROOT/web"
elif [ -d "$SRC_ROOT/webroot" ]; then
  SRC="$SRC_ROOT/webroot"
else
  echo "could not find web or webroot directory under $SRC_ROOT" >&2
  exit 1
fi

mkdir -p "$DEST_ROOT"
rsync -a --delete "$SRC/" "$DEST_ROOT/"

echo "synced ComfyUI frontend from $SRC to $DEST_ROOT"
