# Roadmap

## Current status
- First fully functional text-to-image workflow (checkpoint -> CLIP -> sampler -> VAE -> save).
- UI shows node status, live output preview, and checkpoint selection via the gateway.
- Logging and artifacts captured per service in `.log/` and `artifacts/`.

## Phase 0: Repo scaffold (done)
- Architecture and steering docs aligned to Design.md.
- Docker Compose with service placeholders.
- UI scaffold that will be replaced by the ComfyUI frontend assets.
- Defer publishing LFS model assets until storage/bandwidth planning is finalized.

## Phase 1: Control-plane foundation (done)
- gRPC/protobuf definitions for workflow execution and status.
- Orchestrator API with in-memory job queue and DAG validation.
- HTTP/gRPC-web gateway for the UI.

## Phase 2: Data-plane and artifacts (in progress)
- Artifact reference format (URI + shape/dtype metadata).
- Local volume storage implementation and content hashing.
- First stage service using diffusers for text-to-image output.
- Imagick-powered image stages (pending).

## Phase 3: Multi-stage graphs (in progress)
- Stage grouping strategy (sampler + VAE decode, etc.).
- Parallel scheduling and backpressure in orchestrator.
- Node catalog and parameter schema surfaced to UI.
- UI status streaming + checkpoint dropdown for the Checkpoint Loader node.

## Phase 4: Model runner integration (pending)
- Containerized model server (first inference model).
- Stage service invokes model runner and persists outputs.
- Model and artifact volume management.

## Phase 5: Feasibility evaluation (pending)
- Latency and throughput benchmarks.
- Operational complexity review (logs, configs, scaling).
- Decision on continuing toward production or pivoting.
