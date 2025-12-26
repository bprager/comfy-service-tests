# Lessons learned

## Model compatibility
- Not every file in `models/checkpoints` is a diffusion checkpoint; CLIP, VAE, or video weights will fail to load.
- A lightweight filename-based filter prevents most incompatible assets from surfacing in the UI.
- Large SDXL checkpoints require explicit size limits (`MAX_CHECKPOINT_BYTES`) and sufficient host memory.

## Reliability and UX
- Failed stages should not leave the graph in a locked or unusable state; reset node status on edits.
- Provide a secondary editing surface (Node Inspector) when in-node widget input is flaky.
- Keep status streaming resilient with polling fallback so UI remains responsive on disconnects.

## Ops and observability
- Centralized, timestamped logs across services make pipeline failures actionable.
- Capture artifacts and metadata per job to enable fast replays and debugging.

## Production prep
- Maintain a strict catalog of allowed checkpoints (validated hashes, formats, and sizes).
- Add model provenance metadata and store compatibility tags (sd15/sdxl/video).
- Plan for GPU/CPU split early: memory limits and device flags should be explicit.
