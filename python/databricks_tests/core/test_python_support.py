"""Coverage guard: every PyDABs resource must have an acceptance fixture.

Asserts each resource in the _ResourceType registry has an
acceptance/bundle/python/<plural>-support/ fixture (see that directory's README.md for
how to author one); this fails CI until it exists.
"""

from pathlib import Path

import pytest

from databricks.bundles.core._resource_type import _ResourceType

_ACCEPTANCE_DIR = Path(__file__).parents[3] / "acceptance" / "bundle" / "python"

# Resources knowingly lacking a <plural>-support fixture. Shrink-only: the test fails
# if an entry here is actually covered, so gaps can only close.
_LACKING = {
    # jobs predates the <plural>-support convention; covered across the suite instead.
    "jobs",
}

_PLURALS = sorted(t.plural_name for t in _ResourceType.all())


@pytest.mark.parametrize("plural", _PLURALS)
def test_python_support_coverage(plural: str):
    covered = (_ACCEPTANCE_DIR / f"{plural}-support" / "databricks.yml").exists()

    if plural in _LACKING:
        assert not covered, f"{plural!r} now has a fixture; remove it from _LACKING"
    else:
        assert covered, (
            f"no acceptance/bundle/python/{plural}-support/ fixture for {plural!r}; "
            "add one (see acceptance/bundle/python/README.md) or add it to _LACKING"
        )
