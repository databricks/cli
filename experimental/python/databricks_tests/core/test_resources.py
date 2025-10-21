from dataclasses import dataclass, replace
from typing import Callable

import pytest

from databricks.bundles.apps._models.app import App
from databricks.bundles.core import Location, Resources, Severity
from databricks.bundles.core._bundle import Bundle
from databricks.bundles.core._resource import Resource
from databricks.bundles.core._resource_mutator import (
    ResourceMutator,
    app_mutator,
    job_mutator,
    pipeline_mutator,
    schema_mutator,
    volume_mutator,
)
from databricks.bundles.core._resource_type import _ResourceType
from databricks.bundles.jobs._models.job import Job
from databricks.bundles.pipelines._models.pipeline import Pipeline
from databricks.bundles.schemas._models.schema import Schema
from databricks.bundles.volumes._models.volume import Volume


@dataclass(kw_only=True)
class TestCase:
    add_resource: Callable
    dict_example: dict
    dataclass_example: Resource
    mutator: Callable


resource_types = {tpe.resource_type: tpe for tpe in _ResourceType.all()}
test_cases = [
    (
        TestCase(
            add_resource=Resources.add_job,
            dict_example={"name": "My job"},
            dataclass_example=Job(name="My job"),
            mutator=job_mutator,
        ),
        resource_types[Job],
    ),
    (
        TestCase(
            add_resource=Resources.add_pipeline,
            dict_example={"name": "My pipeline"},
            dataclass_example=Pipeline(name="My pipeline"),
            mutator=pipeline_mutator,
        ),
        resource_types[Pipeline],
    ),
    (
        TestCase(
            add_resource=Resources.add_volume,
            dict_example={
                "name": "My Volume",
                "catalog_name": "my_catalog",
                "schema_name": "my_schema",
            },
            dataclass_example=Volume(
                catalog_name="my_catalog",
                name="My Volume",
                schema_name="my_schema",
            ),
            mutator=volume_mutator,
        ),
        resource_types[Volume],
    ),
    (
        TestCase(
            add_resource=Resources.add_schema,
            dict_example={"catalog_name": "my_catalog", "name": "my_schema"},
            dataclass_example=Schema(catalog_name="my_catalog", name="my_schema"),
            mutator=schema_mutator,
        ),
        resource_types[Schema],
    ),
    (
        TestCase(
            add_resource=Resources.add_app,
            dict_example={"name": "my_app"},
            dataclass_example=App(name="my_app"),
            mutator=app_mutator,
        ),
        resource_types[App],
    ),
]
test_case_ids = [tpe.plural_name for _, tpe in test_cases]


def test_has_all_test_cases():
    for tpe in _ResourceType.all():
        found = False

        for _, test_case_tpe in test_cases:
            if test_case_tpe == tpe:
                found = True
                break

        assert found, f"Missing test case for '{tpe.plural_name}'"


# Job-specific tests are left to self-test and give more readable examples


def test_add_job():
    resources = Resources()

    resources.add_job("my_job", Job(name="My job"))

    assert resources.jobs == {"my_job": Job(name="My job")}


@pytest.mark.parametrize("tc,tpe", test_cases, ids=test_case_ids)
def test_add_resource_type(tc: TestCase, tpe: _ResourceType):
    resources = Resources()

    tc.add_resource(
        resources,
        **{
            "resource_name": "my_resource",
            tpe.singular_name: tc.dict_example,
        },
    )

    resource_dict = getattr(resources, tpe.plural_name)
    assert resource_dict == {"my_resource": tc.dataclass_example}


def test_add_job_dict():
    resources = Resources()

    resources.add_job("my_job", {"name": "My job"})

    assert resources.jobs == {"my_job": Job(name="My job")}


@pytest.mark.parametrize("tc,tpe", test_cases, ids=test_case_ids)
def test_add_resource_type_dict(tc: TestCase, tpe: _ResourceType):
    resources = Resources()

    tc.add_resource(
        resources,
        **{
            "resource_name": "my_resource",
            tpe.singular_name: tc.dict_example,
        },
    )

    resource_dict = getattr(resources, tpe.plural_name)
    assert resource_dict == {"my_resource": tc.dataclass_example}


def test_add_job_location():
    resources = Resources()
    location = Location(file="my_file", line=1, column=2)

    resources.add_job("my_job", Job(name="My job"), location=location)

    assert resources._locations == {("resources", "jobs", "my_job"): location}


@pytest.mark.parametrize("tc,tpe", test_cases, ids=test_case_ids)
def test_add_resource_type_location(tc: TestCase, tpe: _ResourceType):
    resources = Resources()
    location = Location(file="my_file", line=1, column=2)

    tc.add_resource(
        resources,
        **{
            "resource_name": "my_resource",
            tpe.singular_name: tc.dict_example,
            "location": location,
        },
    )

    assert resources._locations == {
        ("resources", tpe.plural_name, "my_resource"): location
    }


def test_add_job_location_automatic():
    resources = Resources()

    resources.add_job("my_job", Job(name="My job"))

    assert resources._locations.keys() == {("resources", "jobs", "my_job")}
    [location] = resources._locations.values()

    assert location.file == __file__
    assert location.line and location.line > 0
    assert location.column and location.column > 0


@pytest.mark.parametrize("tc,tpe", test_cases, ids=test_case_ids)
def test_add_resource_type_location_automatic(tc: TestCase, tpe: _ResourceType):
    resources = Resources()

    tc.add_resource(
        resources,
        **{
            "resource_name": "my_resource",
            tpe.singular_name: tc.dict_example,
        },
    )

    assert resources._locations.keys() == {
        ("resources", tpe.plural_name, "my_resource")
    }
    [location] = resources._locations.values()

    assert location.file == __file__
    assert location.line and location.line > 0
    assert location.column and location.column > 0


@pytest.mark.parametrize("tc,tpe", test_cases, ids=test_case_ids)
def test_add_resource(tc: TestCase, tpe: _ResourceType):
    resources = Resources()

    resources.add_resource("my_resource", tc.dataclass_example)

    resources_dict = getattr(resources, tpe.plural_name)
    assert resources_dict == {"my_resource": tc.dataclass_example}


@pytest.mark.parametrize("tc,tpe", test_cases, ids=test_case_ids)
def test_add_resources(tc: TestCase, tpe: _ResourceType):
    resources_1 = Resources()
    resources_2 = Resources()

    resources_2.add_resource("my_resource", tc.dataclass_example)
    resources_1.add_resources(resources_2)

    resources_dict = getattr(resources_1, tpe.plural_name)
    assert resources_dict == {"my_resource": tc.dataclass_example}


@pytest.mark.parametrize("tc,tpe", test_cases, ids=test_case_ids)
def test_mutator(tc: TestCase, tpe: _ResourceType):
    @tc.mutator
    def my_func(bundle, resource):
        return resource

    bundle = Bundle(target="default")

    assert isinstance(my_func, ResourceMutator)
    assert tc.mutator.__name__ == tpe.singular_name + "_mutator"
    assert my_func.resource_type == tpe.resource_type
    assert my_func.function(bundle, tc.dataclass_example) is tc.dataclass_example


@pytest.mark.parametrize("tc,tpe", test_cases, ids=test_case_ids)
def test_mutator_export(tc: TestCase, tpe: _ResourceType):
    import databricks.bundles.core

    assert tc.mutator.__name__ in databricks.bundles.core.__all__, (
        "mutator is not in databricks.bundles.core.__all__"
    )


@pytest.mark.parametrize("tc,tpe", test_cases, ids=test_case_ids)
def test_add_duplicate_resource(tc: TestCase, tpe: _ResourceType):
    resources = Resources()

    copy_1 = replace(tc.dataclass_example)
    copy_2 = replace(tc.dataclass_example)

    resources.add_resource("my_resource", copy_1)
    resources.add_resource("my_resource", copy_2)

    # it's important not to override resources, because, for instance, they can come from YAML
    resources_dict = getattr(resources, tpe.plural_name)
    assert resources_dict["my_resource"] is copy_1
    assert resources_dict["my_resource"] is not copy_2

    assert len(resources.diagnostics.items) == 1
    [item] = resources.diagnostics.items

    assert item.severity == Severity.ERROR
    assert (
        item.summary
        == f"Duplicate resource name 'my_resource' for a {tpe.singular_name}. Resource names must be unique."
    )


def test_add_diagnostics_error():
    resources = Resources()

    resources.add_diagnostic_error("Error message")

    assert len(resources.diagnostics.items) == 1
    [item] = resources.diagnostics.items

    assert item.severity == Severity.ERROR
    assert item.summary == "Error message"


def test_add_diagnostics_warning():
    resources = Resources()

    resources.add_diagnostic_warning("Error message")

    assert len(resources.diagnostics.items) == 1
    [item] = resources.diagnostics.items

    assert item.severity == Severity.WARNING
    assert item.summary == "Error message"
