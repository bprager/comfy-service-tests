<!-- markdownlint-disable MD024 -->
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.1] - 2025-12-26

### Added
- Lessons learned steering doc under `.vibe/`.

### Changed
- Filter checkpoint catalog to diffusion-compatible assets in the gateway.
- Raise `MAX_CHECKPOINT_BYTES` to 16GiB for larger checkpoints.
- Reset failed node state on edits to keep the UI usable after errors.
- Documentation refreshed for the updated workflow.

## [0.2.0] - 2025-12-26

### Added
- Checkpoint catalog endpoint (`GET /v1/checkpoints`) and UI dropdown integration.
- Node Inspector sidebar for reliable widget editing.
- UI link coloring plus node status highlighting for running/failed nodes.
- Polling fallback when event streams disconnect.
- Stage timeout and checkpoint size guard environment knobs.

### Changed
- Gateway now mounts the models volume to expose checkpoint listings.
- Documentation updated for the first fully functional workflow.

## [0.1.0] - 2025-12-24

### Added

- Architecture and steering docs to capture the control/data plane split.
- Roadmap and testing strategy documentation.
- gRPC protobufs plus Go orchestrator and gateway services.
- Stage sampler service with a diffusers-based text-to-image pipeline.
- UI scaffold with a ComfyUI-style workflow JSON and queue/status preview.
- Docker Compose stack and service Dockerfiles.
- Helper scripts for model syncing, frontend syncing, and Colima setup.
