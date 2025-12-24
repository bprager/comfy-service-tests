#!/usr/bin/env bash
set -euo pipefail

SRC_ROOT="${1:-/Users/bernd/Documents/ComfyUI/models}"
DEST_ROOT="${2:-$(pwd)/models}"

if [ ! -d "$SRC_ROOT" ]; then
  echo "source models dir not found: $SRC_ROOT" >&2
  exit 1
fi

mkdir -p "$DEST_ROOT"

mapfile -t MODEL_FILES < <(find "$SRC_ROOT" -type f -name "*.safetensors" -print)

if [ ${#MODEL_FILES[@]} -eq 0 ]; then
  echo "no .safetensors files found under $SRC_ROOT"
  exit 0
fi

for src in "${MODEL_FILES[@]}"; do
  rel="${src#${SRC_ROOT}/}"
  dest="$DEST_ROOT/$rel"
  mkdir -p "$(dirname "$dest")"
  if [ -f "$dest" ]; then
    echo "skip: $dest exists"
    continue
  fi
  cp -a "$src" "$dest"
  echo "copied: $rel"
done
