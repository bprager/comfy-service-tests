import json
import os
from concurrent import futures
from typing import Dict

import grpc
import torch
from diffusers import (
    DDIMScheduler,
    EulerAncestralDiscreteScheduler,
    EulerDiscreteScheduler,
    StableDiffusionPipeline,
    StableDiffusionXLPipeline,
)

import orchestrator_pb2
import orchestrator_pb2_grpc

ARTIFACTS_ROOT = os.getenv("ARTIFACTS_ROOT", "/artifacts")
CHECKPOINTS_DIR = os.getenv("CHECKPOINTS_DIR", "/models/checkpoints")
DEFAULT_CHECKPOINT = os.getenv("DEFAULT_CHECKPOINT", "")
PIPELINE_KIND = os.getenv("PIPELINE_KIND", "auto")
TORCH_DEVICE = os.getenv("TORCH_DEVICE", "cpu")
TORCH_DTYPE = os.getenv("TORCH_DTYPE", "float32")


class PipelineCache:
    def __init__(self) -> None:
        self._checkpoint: str | None = None
        self._kind: str | None = None
        self._pipe = None

    def get(self, checkpoint: str, kind: str):
        if self._pipe and checkpoint == self._checkpoint and kind == self._kind:
            return self._pipe

        if self._pipe is not None:
            del self._pipe
            self._pipe = None

        pipe = load_pipeline(checkpoint, kind)
        self._pipe = pipe
        self._checkpoint = checkpoint
        self._kind = kind
        return pipe


PIPELINE_CACHE = PipelineCache()


def resolve_checkpoint(name: str) -> str:
    if name:
        if os.path.isabs(name) and os.path.exists(name):
            return name
        candidate = os.path.join(CHECKPOINTS_DIR, name)
        if os.path.exists(candidate):
            return candidate
    if DEFAULT_CHECKPOINT:
        candidate = os.path.join(CHECKPOINTS_DIR, DEFAULT_CHECKPOINT)
        if os.path.exists(candidate):
            return candidate
    raise FileNotFoundError("checkpoint not found")


def detect_kind(checkpoint: str) -> str:
    name = os.path.basename(checkpoint).lower()
    if "xl" in name or "sdxl" in name:
        return "sdxl"
    return "sd15"


def resolve_device() -> str:
    if TORCH_DEVICE == "cuda" and not torch.cuda.is_available():
        return "cpu"
    return TORCH_DEVICE


def parse_dtype(value: str, device: str) -> torch.dtype:
    if device == "cpu":
        return torch.float32
    if value == "float16":
        return torch.float16
    if value == "bfloat16":
        return torch.bfloat16
    return torch.float32


def load_pipeline(checkpoint: str, kind: str):
    device = resolve_device()
    dtype = parse_dtype(TORCH_DTYPE, device)
    if kind == "auto":
        kind = detect_kind(checkpoint)

    if kind == "sdxl":
        pipe = StableDiffusionXLPipeline.from_single_file(
            checkpoint,
            torch_dtype=dtype,
            use_safetensors=True,
        )
    else:
        pipe = StableDiffusionPipeline.from_single_file(
            checkpoint,
            torch_dtype=dtype,
            use_safetensors=True,
        )

    pipe.to(device)
    if hasattr(pipe, "safety_checker"):
        pipe.safety_checker = None
    if hasattr(pipe, "requires_safety_checker"):
        pipe.requires_safety_checker = False
    return pipe


def apply_scheduler(pipe, sampler: str):
    sampler = (sampler or "").lower()
    if sampler in ("euler", "euler_discrete"):
        pipe.scheduler = EulerDiscreteScheduler.from_config(pipe.scheduler.config)
    elif sampler in ("euler_a", "euler_ancestral"):
        pipe.scheduler = EulerAncestralDiscreteScheduler.from_config(pipe.scheduler.config)
    elif sampler in ("ddim",):
        pipe.scheduler = DDIMScheduler.from_config(pipe.scheduler.config)


def parse_int(value: str, fallback: int) -> int:
    try:
        return int(value)
    except (TypeError, ValueError):
        return fallback


def parse_float(value: str, fallback: float) -> float:
    try:
        return float(value)
    except (TypeError, ValueError):
        return fallback


def clamp_dim(value: int) -> int:
    if value < 64:
        value = 64
    return value - (value % 8)


def build_metadata(params: Dict[str, str]) -> Dict[str, str]:
    return {
        "checkpoint": params.get("checkpoint", ""),
        "positive": params.get("positive", ""),
        "negative": params.get("negative", ""),
        "steps": params.get("steps", ""),
        "cfg": params.get("cfg", ""),
        "sampler": params.get("sampler", ""),
        "scheduler": params.get("scheduler", ""),
    }


def write_metadata(path: str, metadata: Dict[str, str]):
    with open(path, "w", encoding="utf-8") as handle:
        json.dump(metadata, handle, indent=2)


class StageRunner(orchestrator_pb2_grpc.StageRunnerServicer):
    def RunStage(self, request, context):
        try:
            checkpoint = resolve_checkpoint(request.params.get("checkpoint", ""))
        except FileNotFoundError as exc:
            return orchestrator_pb2.StageResult(
                stage_id=request.stage_id,
                status="failed",
                error_message=str(exc),
            )

        width = clamp_dim(parse_int(request.params.get("width", "512"), 512))
        height = clamp_dim(parse_int(request.params.get("height", "512"), 512))
        steps = parse_int(request.params.get("steps", "20"), 20)
        cfg = parse_float(request.params.get("cfg", "8"), 8.0)
        seed = parse_int(request.params.get("seed", "0"), 0)

        pipe = PIPELINE_CACHE.get(checkpoint, PIPELINE_KIND)
        apply_scheduler(pipe, request.params.get("sampler", ""))

        generator = None
        device = resolve_device()
        if seed > 0:
            generator = torch.Generator(device=device).manual_seed(seed)

        try:
            result = pipe(
                prompt=request.params.get("positive", ""),
                negative_prompt=request.params.get("negative", ""),
                width=width,
                height=height,
                num_inference_steps=steps,
                guidance_scale=cfg,
                generator=generator,
            )
        except Exception as exc:
            return orchestrator_pb2.StageResult(
                stage_id=request.stage_id,
                status="failed",
                error_message=str(exc),
            )

        output_dir = os.path.join(ARTIFACTS_ROOT, request.stage_id)
        os.makedirs(output_dir, exist_ok=True)

        output_path = os.path.join(output_dir, "output.png")
        result.images[0].save(output_path)

        metadata_path = os.path.join(output_dir, "metadata.json")
        write_metadata(metadata_path, build_metadata(request.params))

        output_ref = orchestrator_pb2.TensorRef(
            uri=output_path,
            shape=[height, width, 3],
            dtype="image/png",
        )

        return orchestrator_pb2.StageResult(
            stage_id=request.stage_id,
            status="completed",
            output_refs={"image": output_ref},
        )

    def Health(self, request, context):
        return orchestrator_pb2.HealthResponse(status="ok")


def serve() -> None:
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=1))
    orchestrator_pb2_grpc.add_StageRunnerServicer_to_server(StageRunner(), server)
    server.add_insecure_port("[::]:9091")
    server.start()
    server.wait_for_termination()


if __name__ == "__main__":
    serve()
