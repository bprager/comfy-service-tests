# Architecture

## Purpose
Validate whether ComfyUI node execution can be decomposed into Dockerized services while preserving the ComfyUI UI/graph model and keeping large tensor movement local.

## Goals
- Keep the UI behavior and workflow JSON compatible with the existing ComfyUI frontend.
- Use Go as the primary service language and Imagick for image operations.
- Split control-plane (small messages) from data-plane (large tensors).
- Prefer coarse-grained services that group tightly coupled stages.

## Non-goals
- Full ComfyUI feature parity.
- A production-ready, multi-tenant platform.

## Architectural principles
- Control-plane uses gRPC/protobuf for scheduling and status.
- Data-plane exchanges opaque references (URI + shape/dtype metadata), not tensors.
- Group latency-sensitive stages (e.g., sampler + VAE decode) in the same service.
- Keep model loading close to compute; avoid per-request reloads.

## High-level components
- UI service
  - ComfyUI-style frontend using litegraph.js and existing layouts/widgets.
  - Preserves workflow JSON format and default graph.
  - Talks to the orchestrator for workflow execution and status.
- Orchestrator (control-plane, Go)
  - Validates graphs, schedules DAG execution, tracks references.
  - Handles retries, backpressure, and status streaming.
  - Exposes gRPC APIs and an optional HTTP/gRPC-web gateway for the UI.
- Stage services (data-plane, Go)
  - Coarse-grained services that implement clusters of node stages.
  - Prefer local or shared storage for tensor artifacts.
  - Image transforms use Imagick where possible.
- Model runner services
  - Host model weights and expose a stable inference API.
- Artifact store
  - Local volume in compose; pluggable to network storage later.

## Data flow (happy path)
1. UI loads the node catalog and default workflow layout from the orchestrator gateway.
2. User submits a graph; UI posts to the orchestrator.
3. Orchestrator validates the graph and schedules stages in topological order.
4. Orchestrator dispatches stage execution via gRPC, passing input references.
5. Stage services read inputs from shared storage, execute, and write outputs.
6. Orchestrator updates job state and streams status to the UI.

## API boundaries
- Orchestrator gRPC API
  - `ExecuteWorkflow(WorkflowGraphRef)`
  - `ExecuteStage(StageRequest)`
  - `StreamStatus(StatusRequest)`
- Stage service gRPC API
  - `RunStage(StageRequest)`
  - `Health(HealthRequest)`
- Model runner API
  - `Infer(InferRequest)` or gRPC equivalent

## Storage tiers (fastest first)
- Local NVMe/hostPath for co-located stages.
- NFS/Filestore for low-latency shared storage.
- Redis for hot intermediates if size permits.
- GCS/S3 for durable outputs and checkpoints.

## Reliability and scaling
- Idempotent execution keyed by (node id, inputs, params).
- Cache reusable outputs by content hash when possible.
- Apply backpressure at the orchestrator rather than in stage services.

## Security
- Private network for service-to-service traffic.
- Scoped credentials or signed URLs for storage access.
- Audit logging for artifact writes.

## Deployment notes
- All services run on a shared Docker network via `docker-compose.yml`.
- A named volume stores models and artifacts in the local POC.
- The UI is exposed on port 8080 by default.

## Proposed directory structure
```
.
|-- cmd/                       # Go entrypoints (per service)
|   |-- orchestrator/
|   `-- gateway/
|-- internal/                  # Shared Go packages
|   |-- orchestrator/
|   |-- stages/
|   |-- storage/
|   `-- proto/
|-- proto/                     # Protobuf definitions
|-- services/                  # Service-specific Docker contexts
|   |-- orchestrator/
|   |-- stage-image/
|   |-- stage-sampling/
|   `-- model-runner/
|-- ui/                        # ComfyUI frontend assets (litegraph.js)
|-- .vibe/                     # Steering docs for vibe coding
|-- docker-compose.yml
|-- architecture.md
|-- ROADMAP.md
|-- TESTING.md
`-- README.md
```
