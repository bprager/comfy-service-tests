import logging
import os
import sys
import time
import types
from concurrent import futures

import grpc

import orchestrator_pb2
import orchestrator_pb2_grpc
from app_core import (
 build_metadata,
 clamp_dim,
 detect_kind,
 parse_float,
 parse_int,
 resolve_checkpoint,
 write_metadata,
)

ARTIFACTS_ROOT = os.getenv("ARTIFACTS_ROOT", "/artifacts")
CHECKPOINTS_DIR = os.getenv("CHECKPOINTS_DIR", "/models/checkpoints")
DEFAULT_CHECKPOINT = os.getenv("DEFAULT_CHECKPOINT", "")
PIPELINE_KIND = os.getenv("PIPELINE_KIND", "auto")
TORCH_DEVICE = os.getenv("TORCH_DEVICE", "cpu")
TORCH_DTYPE = os.getenv("TORCH_DTYPE", "float32")
LOG_DIR = os.getenv("LOG_DIR", "/logs")
MAX_CHECKPOINT_BYTES = int(os.getenv("MAX_CHECKPOINT_BYTES", "0"))

logger = logging.getLogger("stage-sampler")


def setup_logging() -> None:
    os.makedirs(LOG_DIR, exist_ok=True)
    log_path = os.path.join(LOG_DIR, "stage-sampler.log")

    logger.setLevel(logging.INFO)
    formatter = logging.Formatter(
        "%(asctime)s %(name)s %(levelname)s %(message)s",
        "%Y-%m-%dT%H:%M:%SZ",
    )
    formatter.converter = time.gmtime

    logger.handlers.clear()
    logger.propagate = False

    file_handler = logging.FileHandler(log_path)
    file_handler.setFormatter(formatter)

    stream_handler = logging.StreamHandler(sys.stdout)
    stream_handler.setFormatter(formatter)

    logger.addHandler(file_handler)
    logger.addHandler(stream_handler)
    logger.info("logging initialized log_path=%s", log_path)


def get_torch():
    try:
        import torch

        return torch
    except Exception as exc:
        logger.exception("failed to import torch: %s", exc)
        raise


def get_diffusers():
    try:
        from diffusers import (
            DDIMScheduler,
            EulerAncestralDiscreteScheduler,
            EulerDiscreteScheduler,
            StableDiffusionPipeline,
            StableDiffusionXLPipeline,
        )
    except Exception as exc:
        logger.exception("failed to import diffusers: %s", exc)
        raise

    return types.SimpleNamespace(
        DDIMScheduler=DDIMScheduler,
        EulerAncestralDiscreteScheduler=EulerAncestralDiscreteScheduler,
        EulerDiscreteScheduler=EulerDiscreteScheduler,
        StableDiffusionPipeline=StableDiffusionPipeline,
        StableDiffusionXLPipeline=StableDiffusionXLPipeline,
    )


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

        logger.info("initializing pipeline checkpoint=%s kind=%s", checkpoint, kind)
        pipe = load_pipeline(checkpoint, kind)
        self._pipe = pipe
        self._checkpoint = checkpoint
        self._kind = kind
        return pipe


PIPELINE_CACHE = PipelineCache()


def resolve_device() -> str:
    torch = get_torch()
    if TORCH_DEVICE == "cuda" and not torch.cuda.is_available():
        logger.warning("cuda requested but not available, falling back to cpu")
        return "cpu"
    return TORCH_DEVICE


def parse_dtype(value: str, device: str):
    torch = get_torch()
    if device == "cpu":
        return torch.float32
    if value == "float16":
        return torch.float16
    if value == "bfloat16":
        return torch.bfloat16
    return torch.float32


def format_error(exc: Exception) -> str:
    message = str(exc)
    name = type(exc).__name__
    if message:
        return f"{name}: {message}"
    return name


def resolve_fallback_checkpoint(current: str) -> str | None:
    if not DEFAULT_CHECKPOINT:
        return None
    try:
        fallback = resolve_checkpoint(DEFAULT_CHECKPOINT, CHECKPOINTS_DIR, DEFAULT_CHECKPOINT)
    except FileNotFoundError:
        return None
    if os.path.abspath(fallback) == os.path.abspath(current):
        return None
    return fallback


def get_pipeline_with_fallback(checkpoint: str, kind: str):
    try:
        return PIPELINE_CACHE.get(checkpoint, kind), checkpoint, ""
    except Exception as exc:
        error_message = format_error(exc)
        fallback = resolve_fallback_checkpoint(checkpoint)
        if fallback:
            logger.warning(
                "pipeline load failed checkpoint=%s err=%s; falling back to %s",
                checkpoint,
                error_message,
                fallback,
            )
            try:
                return PIPELINE_CACHE.get(fallback, kind), fallback, error_message
            except Exception as fallback_exc:
                fallback_message = format_error(fallback_exc)
                raise RuntimeError(
                    f"fallback checkpoint failed: {fallback_message} (original: {error_message})"
                ) from fallback_exc
        raise


def load_pipeline(checkpoint: str, kind: str):
    if MAX_CHECKPOINT_BYTES > 0:
        size_bytes = os.path.getsize(checkpoint)
        if size_bytes > MAX_CHECKPOINT_BYTES:
            size_mb = size_bytes / (1024 * 1024)
            limit_mb = MAX_CHECKPOINT_BYTES / (1024 * 1024)
            raise RuntimeError(
                f"checkpoint too large ({size_mb:.0f}MB) for limit {limit_mb:.0f}MB"
            )

    device = resolve_device()
    dtype = parse_dtype(TORCH_DTYPE, device)
    if kind == "auto":
        kind = detect_kind(checkpoint)
    logger.info("loading pipeline checkpoint=%s kind=%s device=%s dtype=%s", checkpoint, kind, device, dtype)

    if kind == "sdxl":
        pipe = get_diffusers().StableDiffusionXLPipeline.from_single_file(
            checkpoint,
            torch_dtype=dtype,
            use_safetensors=True,
        )
    else:
        pipe = get_diffusers().StableDiffusionPipeline.from_single_file(
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
        pipe.scheduler = get_diffusers().EulerDiscreteScheduler.from_config(pipe.scheduler.config)
    elif sampler in ("euler_a", "euler_ancestral"):
        pipe.scheduler = get_diffusers().EulerAncestralDiscreteScheduler.from_config(pipe.scheduler.config)
    elif sampler in ("ddim",):
        pipe.scheduler = get_diffusers().DDIMScheduler.from_config(pipe.scheduler.config)
    elif sampler:
        logger.warning("unknown sampler %s, using pipeline default", sampler)


class StageRunner(orchestrator_pb2_grpc.StageRunnerServicer):
    def RunStage(self, request, context):
        requested_checkpoint = request.params.get("checkpoint", "")
        try:
            checkpoint = resolve_checkpoint(
                requested_checkpoint,
                CHECKPOINTS_DIR,
                DEFAULT_CHECKPOINT,
            )
        except FileNotFoundError as exc:
            logger.error("checkpoint not found: %s", exc)
            return orchestrator_pb2.StageResult(
                stage_id=request.stage_id,
                status="failed",
                error_message=format_error(exc),
            )

        width = clamp_dim(parse_int(request.params.get("width", "512"), 512))
        height = clamp_dim(parse_int(request.params.get("height", "512"), 512))
        steps = parse_int(request.params.get("steps", "20"), 20)
        cfg = parse_float(request.params.get("cfg", "8"), 8.0)
        seed = parse_int(request.params.get("seed", "0"), 0)

        params = dict(request.params)
        try:
            pipe, resolved_checkpoint, fallback_error = get_pipeline_with_fallback(
                checkpoint, PIPELINE_KIND
            )
        except Exception as exc:
            error_message = format_error(exc)
            logger.exception("failed to load pipeline: %s", error_message)
            return orchestrator_pb2.StageResult(
                stage_id=request.stage_id,
                status="failed",
                error_message=error_message,
            )
        params["checkpoint"] = os.path.basename(resolved_checkpoint)
        if fallback_error and requested_checkpoint:
            params["checkpoint_requested"] = requested_checkpoint
        apply_scheduler(pipe, request.params.get("sampler", ""))

        generator = None
        device = resolve_device()
        if seed > 0:
            generator = get_torch().Generator(device=device).manual_seed(seed)

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
            error_message = format_error(exc)
            logger.exception("pipeline execution failed: %s", error_message)
            return orchestrator_pb2.StageResult(
                stage_id=request.stage_id,
                status="failed",
                error_message=error_message,
            )

        output_dir = os.path.join(ARTIFACTS_ROOT, request.stage_id)
        os.makedirs(output_dir, exist_ok=True)

        output_path = os.path.join(output_dir, "output.png")
        try:
            result.images[0].save(output_path)
        except Exception as exc:
            error_message = format_error(exc)
            logger.exception("failed to save output: %s", error_message)
            return orchestrator_pb2.StageResult(
                stage_id=request.stage_id,
                status="failed",
                error_message=error_message,
            )

        metadata_path = os.path.join(output_dir, "metadata.json")
        try:
            write_metadata(metadata_path, build_metadata(params))
        except Exception as exc:
            error_message = format_error(exc)
            logger.exception("failed to write metadata: %s", error_message)
            return orchestrator_pb2.StageResult(
                stage_id=request.stage_id,
                status="failed",
                error_message=error_message,
            )

        logger.info("job completed id=%s output=%s", request.stage_id, output_path)

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
    setup_logging()
    if not os.path.isdir(CHECKPOINTS_DIR):
        logger.warning("checkpoints dir missing: %s", CHECKPOINTS_DIR)
    logger.info(
        "starting stage-sampler artifacts=%s checkpoints=%s",
        ARTIFACTS_ROOT,
        CHECKPOINTS_DIR,
    )
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=1))
    orchestrator_pb2_grpc.add_StageRunnerServicer_to_server(StageRunner(), server)
    server.add_insecure_port("[::]:9091")
    server.start()
    server.wait_for_termination()


if __name__ == "__main__":
    serve()
