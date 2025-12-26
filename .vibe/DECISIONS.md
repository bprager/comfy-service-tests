# Decisions

## 2024-XX-XX
- Adopt a control-plane + data-plane split with gRPC in the control plane.
- Avoid sending large tensors over the network; use references instead.
- Preserve the ComfyUI frontend behavior, layouts, and workflow JSON format.
- Prefer coarse-grained stage services over per-node microservices.
- Compose-based deployment is the default for the feasibility study.

## 2025-12-26
- Use a lightweight HTTP gateway for UI integration (workflows, status, events).
- Expose checkpoints via `/v1/checkpoints` and keep models on a shared volume.
- Provide a Node Inspector sidebar to edit node widget values reliably.
- Baseline pipeline uses diffusers (sampler + VAE decode) for the first end-to-end workflow.
