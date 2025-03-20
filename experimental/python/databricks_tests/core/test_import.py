import sys

import pytest


@pytest.fixture(scope="function")
def reset_sys_modules():
    """
    Reset sys.module before test and restore it after test.
    """

    original_modules = sys.modules.copy()

    for module in original_modules.keys():
        if module.startswith("databricks.bundles"):
            del sys.modules[module]

    try:
        yield
    finally:
        # Clear any modifications and restore the original modules
        sys.modules.clear()
        sys.modules.update(original_modules)


def test_import_regression(reset_sys_modules):
    """
    We don't want importing databricks.bundles.core to import any of resources.

    Naive implementation can have depenency due to typing. To break it we should be
    using TYPE_CHECKING imports. Otherwise, there is a cycle between
    databricks.bundles.core.Resource and databricks.bundles.jobs.Job
    (or other resources).
    """

    import databricks.bundles.core

    assert databricks.bundles.core

    for module in sys.modules.keys():
        if module.startswith("databricks.bundles."):
            is_core = module == "databricks.bundles.core" or module.startswith(
                "databricks.bundles.core."
            )

            assert is_core, f"Unexpected loaded module AFTER loading core: {module}"
