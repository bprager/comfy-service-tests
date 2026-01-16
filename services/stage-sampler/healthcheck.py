import sys

import grpc

import orchestrator_pb2
import orchestrator_pb2_grpc


def main() -> int:
    try:
        channel = grpc.insecure_channel("127.0.0.1:9091")
        stub = orchestrator_pb2_grpc.StageRunnerStub(channel)
        stub.Health(orchestrator_pb2.HealthRequest(), timeout=2.0)
        return 0
    except Exception as exc:
        print(f"stage-sampler healthcheck failed: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
