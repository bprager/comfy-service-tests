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
  - Checkpoint dropdown should reflect `models/checkpoints` contents.
- Performance tests
  - Measure per-stage latency and overall graph duration.
  - Track memory and CPU per service group.
  - Compare storage backends (local vs shared) when possible.

## Tooling
- `go test ./...` for unit tests.
- Lightweight test harness container for integration runs.
- `node --test ui/tests/*.test.js` for UI widget configuration checks.
- Optional load testing via k6 or vegeta (TBD).

### Coverage targets
- Go unit coverage: 90%+ for `internal/orchestrator` and `internal/imaging` (excludes generated protobufs and `cmd/` entrypoints).
- Python unit coverage: 90%+ for `services/stage-sampler/app_core.py` helpers.

Use the helper script to run the scoped coverage checks:
```sh
./scripts/test.sh
```

## Manual smoke workflow
```sh
curl -s http://localhost:8084/v1/checkpoints
curl -s -X POST http://localhost:8084/v1/workflows \
  -H "Content-Type: application/json" \
  --data-binary @ui/workflows/default.json
```

Then poll `GET /v1/jobs/:id` and fetch `GET /v1/jobs/:id/output` for the image.

## Test data
- Use small sample images and deterministic seeds.
- Store golden outputs under `testdata/`.
