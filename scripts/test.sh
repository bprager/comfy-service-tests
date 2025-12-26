#!/usr/bin/env bash
set -euo pipefail

mkdir -p coverage

GO_COVERAGE_PROFILE="coverage/go.out"

go test ./internal/orchestrator ./internal/imaging -coverprofile "$GO_COVERAGE_PROFILE" -covermode=atomic

go tool cover -func "$GO_COVERAGE_PROFILE" | tail -n 1

PYTHONPATH=services/stage-sampler \
python -m pytest services/stage-sampler/tests \
  --cov=app_core \
  --cov-report=term-missing \
  --cov-fail-under=90

if command -v node >/dev/null 2>&1; then
  node --test ui/tests
else
  echo "node not found; skipping UI tests"
fi
