# ComfyUI service tests

A feasibility test for splitting ComfyUI node execution into Dockerized services with a Go-based control plane and a litegraph.js UI that preserves ComfyUI workflows.

## Goals

- Validate a control-plane + data-plane split for ComfyUI-style execution.
- Use Go + Imagick (ImageMagick MagickWand) for image operations.
- Keep large tensors local and exchange references between services.
- Preserve the existing ComfyUI frontend behavior and workflow JSON.

## Quick start

```sh
docker compose up --build
```

- UI: <http://localhost:8080>
- Gateway: <http://localhost:8084>
- Orchestrator (gRPC): <http://localhost:9090>
- Stage sampler (gRPC): <http://localhost:9091>

The stage sampler runs a diffusers-based Stable Diffusion pipeline (sampler + VAE decode). It writes outputs to the artifacts volume for UI preview.

### Checkpoints
- Place checkpoints in `./models/checkpoints` (`.safetensors`, `.ckpt`, `.pt`, `.bin`).
- The UI loads the list from `GET /v1/checkpoints` and shows it as a dropdown in the Checkpoint Loader node.
- The gateway filters obvious non-diffusion assets (CLIP, VAE, video, etc.) by filename heuristics; adjust the filter in `cmd/gateway/main.go` if needed.
- Use the Node Inspector (right sidebar) if you prefer editing values outside the node widgets.

Environment knobs in `docker-compose.yml`:
- `orchestrator`
  - `STAGE_TIMEOUT` timeout for stage calls
- `stage-sampler`
  - `CHECKPOINTS_DIR` path to checkpoints (default `/models/checkpoints`)
  - `DEFAULT_CHECKPOINT` checkpoint filename to load
  - `PIPELINE_KIND` `auto` | `sdxl` | `sd15`
  - `TORCH_DEVICE` `cpu` | `cuda`
  - `TORCH_DTYPE` `float32` | `float16` | `bfloat16`
  - `MAX_CHECKPOINT_BYTES` size guard for checkpoints

## Logging
Service logs are written under `.log/` when running via Docker Compose:
- `.log/orchestrator/orchestrator.log`
- `.log/gateway/gateway.log`
- `.log/stage-sampler/stage-sampler.log`
- `.log/nginx/` (nginx access/error logs)

## Optional: sync ComfyUI frontend

```sh
./scripts/sync_comfyui_frontend.sh /path/to/ComfyUI
```

This overwrites the contents of `ui/` with the ComfyUI frontend assets.

## Project structure

- `architecture.md` system overview and boundaries
- `.vibe/Design.md` source design notes and microservice inventory
- `docker-compose.yml` local POC stack
- `ui/` litegraph.js UI scaffold (replace with ComfyUI frontend assets via `scripts/sync_comfyui_frontend.sh`)
- `proto/` gRPC/protobuf definitions for the control plane and stage services
- `cmd/` Go service entrypoints (gateway + orchestrator)
- `scripts/` helper scripts for syncing models and UI assets
- `.vibe/` steering docs for vibe coding
- `ROADMAP.md` progress plan
- `TESTING.md` testing strategy

## Next steps

- Swap in the ComfyUI frontend assets and map UI events to the gateway.
- Add scheduler mappings and node params beyond the minimal text-to-image flow.
- Expand stage services and storage backends per `.vibe/Design.md`.
