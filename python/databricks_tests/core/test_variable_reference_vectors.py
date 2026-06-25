import json
from pathlib import Path
from typing import Any, Optional

import pytest
from databricks.bundles.core._transform import (
    _unwrap_variable_path,
    _variable_regex,
)

_REFERENCE_VECTORS = Path(__file__).resolve().parents[3] / "libs/dyn/dynvar/testdata/reference_vectors.json"


def _load_reference_vectors() -> list[dict[str, Any]]:
    with _REFERENCE_VECTORS.open(encoding="utf-8") as f:
        data = json.load(f)
    vectors = data["vectors"]
    assert vectors
    return vectors


def _find_references(value: str) -> list[str]:
    return [match.group(1) for match in _variable_regex.finditer(value)]


def _matches(value: str) -> bool:
    return bool(_find_references(value))


@pytest.mark.parametrize("vector", _load_reference_vectors(), ids=lambda v: v["id"])
def test_reference_vectors(vector: dict[str, Any]) -> None:
    value: str = vector["input"]
    assert _matches(value) == vector["match"], "regex match"

    if vector["match"] and "references" in vector:
        assert _find_references(value) == vector["references"], "references"

    if "pure" in vector:
        unwrap_path: Optional[str] = _unwrap_variable_path(value)
        if vector["pure"]:
            assert unwrap_path is not None, "unwrap path for pure reference"
        else:
            assert unwrap_path is None, "unwrap path for impure reference"

    if "path" in vector:
        unwrap_path = _unwrap_variable_path(value)
        assert unwrap_path == vector["path"], "PureReferenceToPath equivalent"
    elif vector.get("path_ok") is False:
        assert _unwrap_variable_path(value) is None, "PureReferenceToPath equivalent"
