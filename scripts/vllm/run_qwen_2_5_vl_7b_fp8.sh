#!/usr/bin/env bash
set -euo pipefail

VLLM_IMAGE="${VLLM_IMAGE:-vllm/vllm-openai:v0.4.2}"
PORT="${PORT:-8001}"
MODEL_DIR="${MODEL_DIR:-$(pwd)/models/llm/qwen_2_5_vl_7b_fp8}"
VLLM_ARGS="${VLLM_ARGS:-}"

if [ ! -d "$MODEL_DIR" ]; then
  echo "model directory not found: $MODEL_DIR" >&2
  echo "vLLM requires a full Hugging Face model directory (config.json, tokenizer files)." >&2
  exit 1
fi

if [ ! -f "$MODEL_DIR/config.json" ]; then
  echo "missing $MODEL_DIR/config.json" >&2
  echo "copy the full model repo into $MODEL_DIR before starting vLLM." >&2
  exit 1
fi

docker run --rm --gpus all \
  -p "${PORT}:8000" \
  -v "$MODEL_DIR:/model:ro" \
  "$VLLM_IMAGE" \
  --model /model \
  --host 0.0.0.0 \
  --port 8000 \
  $VLLM_ARGS
