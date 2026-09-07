"""Coverage guard: every PyDABs resource must have an acceptance fixture.

Asserts each resource in the _ResourceType registry has an
acceptance/bundle/python/<plural>-support/ fixture (see that directory's README.md for
how to author one); this fails CI until it exists.
"""

from pathlib import Path

import pytest

from databricks.bundles.core._resource_type import _ResourceType

_ACCEPTANCE_DIR = Path(__file__).parents[3] / "acceptance" / "bundle" / "python"

_PLURALS = sorted(t.plural_name for t in _ResourceType.all())


@pytest.mark.parametrize("plural", _PLURALS)
def test_python_support_coverage(plural: str):
    covered = (_ACCEPTANCE_DIR / f"{plural}-support" / "databricks.yml").exists()

    assert covered, (
        f"no acceptance/bundle/python/{plural}-support/ fixture for {plural!r}; "
        "add one (see acceptance/bundle/python/README.md)"
    )
