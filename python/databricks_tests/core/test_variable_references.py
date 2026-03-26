"""Cross-language test for variable reference detection.

Loads shared test cases from libs/interpolation/testdata/variable_references.json
and verifies the Python regex agrees with the Go parser on which strings are pure
variable references.

The same JSON file is consumed by the Go test suite
(libs/interpolation/parse_test.go:TestParsePureVariableReferences).

When modifying the Go parser (e.g. adding new key patterns, escape sequences,
or reference syntax), add test cases to the JSON file so both Go and Python
are validated.
"""

import json
from pathlib import Path

import pytest

from databricks.bundles.core._transform import _unwrap_variable_path

_testdata = (
    Path(__file__).resolve().parents[3]
    / "libs"
    / "interpolation"
    / "testdata"
    / "variable_references.json"
)
_cases = json.loads(_testdata.read_text())


@pytest.mark.parametrize(
    "tc",
    _cases,
    ids=[tc["comment"] for tc in _cases],
)
def test_pure_variable_reference(tc):
    result = _unwrap_variable_path(tc["input"])

    if tc["is_pure_ref"]:
        assert result == tc.get("path"), (
            f"expected pure ref with path={tc.get('path')!r}, got {result!r}"
        )
    else:
        assert result is None, f"expected None for non-pure ref, got {result!r}"
