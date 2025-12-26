import json
import sys
from pathlib import Path

import pytest

sys.path.append(str(Path(__file__).resolve().parents[1]))

import app_core


def test_parse_int_and_float():
    assert app_core.parse_int("12", 0) == 12
    assert app_core.parse_int("bad", 7) == 7
    assert app_core.parse_float("3.5", 0.0) == 3.5
    assert app_core.parse_float(None, 1.5) == 1.5


def test_clamp_dim():
    assert app_core.clamp_dim(32) == 64
    assert app_core.clamp_dim(65) == 64
    assert app_core.clamp_dim(128) == 128


def test_detect_kind():
    assert app_core.detect_kind("modelXL.safetensors") == "sdxl"
    assert app_core.detect_kind("sd15.safetensors") == "sd15"


def test_resolve_checkpoint(tmp_path: Path):
    checkpoints = tmp_path / "checkpoints"
    checkpoints.mkdir()
    target = checkpoints / "model.safetensors"
    target.write_text("stub")

    resolved = app_core.resolve_checkpoint("model.safetensors", str(checkpoints), "")
    assert resolved == str(target)

    default_target = checkpoints / "default.safetensors"
    default_target.write_text("stub")
    resolved_default = app_core.resolve_checkpoint("", str(checkpoints), "default.safetensors")
    assert resolved_default == str(default_target)

    with pytest.raises(FileNotFoundError):
        app_core.resolve_checkpoint("missing.safetensors", str(checkpoints), "")


def test_metadata_write(tmp_path: Path):
    meta = app_core.build_metadata({"checkpoint": "foo", "steps": "20"})
    path = tmp_path / "meta.json"
    app_core.write_metadata(path, meta)

    loaded = json.loads(path.read_text())
    assert loaded["checkpoint"] == "foo"
    assert loaded["steps"] == "20"
