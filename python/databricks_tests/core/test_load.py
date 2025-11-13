import os
from dataclasses import replace

import databricks_tests.fixtures as fixtures
import databricks_tests.fixtures.dummy_module as dummy_module
from databricks.bundles.core import Location, load_resources_from_current_package_module
from databricks.bundles.core._load import (
    _parse_locations,
    load_resources_from_module,
    load_resources_from_package_module,
)
from databricks.bundles.jobs import Job


def test_load_resources_from_current_package_module():
    resources = load_resources_from_current_package_module()

    assert not resources.diagnostics.has_error()


def test_load_resources_from_module():
    class MyModule:
        my_job_1 = Job.from_dict({})
        __file__ = "my_module.py"

    resources = load_resources_from_module(MyModule)  # type: ignore

    assert not resources.diagnostics.has_error()
    assert resources.jobs == {"my_job_1": Job.from_dict({})}


def test_load_resources_from_module_error():
    class MyModuleMeta(type):
        def __dir__(cls):
            raise Exception("error")

    class MyModule(metaclass=MyModuleMeta): ...

    resources = load_resources_from_module(MyModule)  # type: ignore

    assert resources.jobs == {}
    assert resources.diagnostics.has_error()
    assert resources.diagnostics.items[0].summary == "Error while loading 'MyModule'"


def test_load_resources_from_package_module():
    # function doesn't throw exception, because we want to load as much as we can
    resources = load_resources_from_package_module(fixtures)

    # dummy_module is successfully loaded
    assert resources.jobs == {"my_job": dummy_module.my_job}

    # error_module has error, and it is not loaded
    assert resources.diagnostics.has_error()
    assert len(resources.diagnostics.items) == 1
    assert (
        resources.diagnostics.items[0].summary
        == "Error while loading 'databricks_tests.fixtures.error_module'"
    )


def test_parse_locations():
    locations = _parse_locations(dummy_module)
    locations = {k: normalize_location(v) for k, v in locations.items()}

    assert locations == {
        "my_job": Location(
            line=3,
            column=1,
            file="databricks_tests/fixtures/dummy_module.py",
        ),
    }


def normalize_location(location: Location) -> Location:
    return replace(location, file=os.path.relpath(location.file, os.getcwd()))
