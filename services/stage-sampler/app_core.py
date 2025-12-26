import json
import os
from typing import Dict


def resolve_checkpoint(name: str, checkpoints_dir: str, default_checkpoint: str) -> str:
    if name:
        if os.path.isabs(name) and os.path.exists(name):
            return name
        candidate = os.path.join(checkpoints_dir, name)
        if os.path.exists(candidate):
            return candidate
    if default_checkpoint:
        candidate = os.path.join(checkpoints_dir, default_checkpoint)
        if os.path.exists(candidate):
            return candidate
    raise FileNotFoundError("checkpoint not found")


def detect_kind(checkpoint: str) -> str:
    name = os.path.basename(checkpoint).lower()
    if "xl" in name or "sdxl" in name:
        return "sdxl"
    return "sd15"


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
