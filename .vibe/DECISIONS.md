# Decisions

## 2024-XX-XX
- Adopt a control-plane + data-plane split with gRPC in the control plane.
- Avoid sending large tensors over the network; use references instead.
- Preserve the ComfyUI frontend behavior, layouts, and workflow JSON format.
- Prefer coarse-grained stage services over per-node microservices.
- Compose-based deployment is the default for the feasibility study.
