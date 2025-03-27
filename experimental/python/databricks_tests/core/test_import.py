import sys
from types import ModuleType

import pytest


@pytest.fixture(scope="function")
def reset_sys_modules():
    """
    Reset sys.module before test and restore it after test.
    """

    original_modules = sys.modules.copy()
    try:
        import databricks.bundles

        for module in original_modules.keys():
            if is_subpackage(module, databricks.bundles):
                del sys.modules[module]

        assert databricks.bundles.__package__
        del sys.modules[databricks.bundles.__package__]

        yield
    finally:
        # Clear any modifications and restore the original modules
        sys.modules.clear()
        sys.modules.update(original_modules)


def test_import_regression(reset_sys_modules):
    """
    We don't want importing databricks.bundles.core to import any of resources.

    We refer to resources in type signatures (e.g. in Resources or @job_mutator),
    unless we use forward references (e.g. "Job" instead of Job) we can create a
    cycling dependency resulting in a runtime error.

    We want every package with resources (e.g., databricks.bundles.jobs) to be loaded
    only when needed.
    """

    import databricks.bundles.core

    assert databricks.bundles.core

    import databricks.bundles

    for module in sys.modules.keys():
        if is_subpackage(module, databricks.bundles):
            is_core = module == databricks.bundles.core.__package__
            is_core_subpackage = is_subpackage(module, databricks.bundles.core)

            assert is_core or is_core_subpackage, (
                f"Unexpected loaded module AFTER loading core: {module}"
            )


def is_subpackage(module: str, parent: ModuleType) -> bool:
    assert parent.__package__

    return module.startswith(parent.__package__ + ".")
