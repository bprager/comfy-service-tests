# Roadmap

## Phase 0: Repo scaffold
- Architecture and steering docs aligned to Design.md.
- Docker Compose with service placeholders.
- UI scaffold that will be replaced by the ComfyUI frontend assets.

## Phase 1: Control-plane foundation
- gRPC/protobuf definitions for workflow execution and status.
- Orchestrator API with in-memory job queue and DAG validation.
- HTTP/gRPC-web gateway for the UI.

## Phase 2: Data-plane and artifacts
- Artifact reference format (URI + shape/dtype metadata).
- Local volume storage implementation and content hashing.
- First stage service using Imagick for image transforms.

## Phase 3: Multi-stage graphs
- Stage grouping strategy (sampler + VAE decode, etc.).
- Parallel scheduling and backpressure in orchestrator.
- Node catalog and parameter schema surfaced to UI.

## Phase 4: Model runner integration
- Containerized model server (first inference model).
- Stage service invokes model runner and persists outputs.
- Model and artifact volume management.

## Phase 5: Feasibility evaluation
- Latency and throughput benchmarks.
- Operational complexity review (logs, configs, scaling).
- Decision on continuing toward production or pivoting.
