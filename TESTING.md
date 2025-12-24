# Testing strategy

## Goals
- Validate correctness of stage outputs.
- Confirm orchestrator scheduling, retries, and error handling.
- Verify gRPC contract compatibility between services.
- Measure end-to-end performance for feasibility.

## Test types
- Unit tests (Go)
  - Stage logic, graph validation, scheduler decisions.
  - Image transforms with golden images.
- Contract tests
  - Orchestrator <-> stage service gRPC payloads.
  - Stage service <-> model runner API payloads.
- Integration tests
  - docker-compose based tests with real containers.
  - Artifact store read/write behavior and reference validation.
- End-to-end tests
  - UI submits a workflow JSON graph and renders output.
  - Smoke tests for job status, queue, and cancellation.
- Performance tests
  - Measure per-stage latency and overall graph duration.
  - Track memory and CPU per service group.
  - Compare storage backends (local vs shared) when possible.

## Tooling
- `go test ./...` for unit tests.
- Lightweight test harness container for integration runs.
- Optional load testing via k6 or vegeta (TBD).

## Test data
- Use small sample images and deterministic seeds.
- Store golden outputs under `testdata/`.
