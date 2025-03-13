import sys
from io import StringIO
from pathlib import Path

from databricks.bundles.build import (
    _append_resources,
    _Args,
    _load_object,
    _parse_args,
    _parse_bundle_info,
    _relativize_location,
    _write_diagnostics,
    _write_locations,
)
from databricks.bundles.core import (
    Bundle,
    Diagnostic,
    Diagnostics,
    Location,
    Resources,
    Severity,
)
from databricks.bundles.jobs import Job


def test_write_diagnostics():
    diagnostics = Diagnostics.create_warning(
        "foo is deprecated", path=("resources", "jobs", "job_0")
    )
    diagnostics = diagnostics.extend(Diagnostics.create_error("foo is not available"))

    out = StringIO()

    _write_diagnostics(out, diagnostics)

    assert out.getvalue() == (
        '{"severity": "warning", "summary": "foo is deprecated", "path": "resources.jobs.job_0"}\n'
        '{"severity": "error", "summary": "foo is not available"}\n'
    )


def test_write_diagnostics_location():
    diagnostics = Diagnostics.create_warning(
        "foo is deprecated", location=Location(file="foo.py", line=42, column=1)
    )

    out = StringIO()

    _write_diagnostics(out, diagnostics)

    assert out.getvalue() == (
        '{"severity": "warning", "summary": "foo is deprecated", "location": {"file": "foo.py", "line": 42, "column": 1}}\n'
    )


def test_write_diagnostics_detail():
    diagnostics = Diagnostics(
        items=(
            Diagnostic(
                severity=Severity.WARNING,
                summary="foo",
                detail="foo detail",
                location=Location(
                    file="foo.py",
                    line=42,
                    column=1,
                ),
            ),
        ),
    )

    out = StringIO()
    _write_diagnostics(out, diagnostics)

    assert out.getvalue() == (
        '{"severity": "warning", "summary": "foo", "detail": "foo detail", '
        '"location": {"file": "foo.py", "line": 42, "column": 1}}\n'
    )


def test_write_location():
    locations = {
        ("resources", "jobs", "job_0"): Location(file="foo.py", line=42, column=1),
    }

    out = StringIO()
    _write_locations(out, locations)

    assert (
        out.getvalue()
        == '{"path": "resources.jobs.job_0", "file": "foo.py", "line": 42, "column": 1}\n'
    )


def test_relativize_location():
    file = Path("bar.py").absolute().as_posix()
    location = Location(file=file, line=42, column=1)

    assert _relativize_location(location) == Location(file="bar.py", line=42, column=1)


def test_load_object_common_error():
    obj, diagnostics = _load_object("resources:load_resources")
    [item] = diagnostics.items

    assert obj is None

    assert item.severity == Severity.ERROR
    assert item.summary == "Can't find function 'load_resources' in module 'resources'"

    assert item.detail
    assert "ModuleNotFoundError: No module named 'resources'" in item.detail
    assert "Explanation:" in item.detail


def test_load_object_failed_to_import():
    obj, diagnostics = _load_object("uncommon_module:load_resources")
    [item] = diagnostics.items

    assert obj is None

    assert item.severity == Severity.ERROR
    assert item.summary == "Failed to import module 'uncommon_module'"

    assert item.detail
    assert "ModuleNotFoundError: No module named 'uncommon_module'" in item.detail
    assert "Explanation:" not in item.detail


def test_load_object_no_attr_common():
    class FakeModule:
        __file__ = "resources.py"

    try:
        sys.modules["resources"] = FakeModule  # type:ignore
        obj, diagnostics = _load_object("resources:load_resources")
        [item] = diagnostics.items
    finally:
        del sys.modules["resources"]

    assert obj is None

    assert item.severity == Severity.ERROR
    assert item.summary == "Can't find function 'load_resources' in module 'resources'"

    assert item.detail
    assert "AttributeError:" in item.detail
    assert "Explanation:" in item.detail


def test_load_object_no_attr_uncommon():
    class FakeModule:
        __file__ = "uncommon_module.py"

    try:
        sys.modules["uncommon_module"] = FakeModule  # type:ignore
        obj, diagnostics = _load_object("uncommon_module:load_resources")
        [item] = diagnostics.items
    finally:
        del sys.modules["uncommon_module"]

    assert obj is None

    assert item.severity == Severity.ERROR
    assert item.summary == "Name 'load_resources' not found in module 'uncommon_module'"

    assert item.detail
    assert "AttributeError:" in item.detail
    assert "Explanation:" not in item.detail


def test_parse_bundle_info():
    input = {
        "bundle": {
            "target": "development",
        },
        "variables": {
            "foo": {
                "value": "bar",
            }
        },
    }

    assert _parse_bundle_info(input) == Bundle(
        target="development",
        variables={
            "foo": "bar",
        },
    )


def test_append_resources():
    input = {
        "resources": {
            "jobs": {
                "job_0": {"name": "job_0"},
                "job_1": {"name": "job_1"},
            }
        },
    }

    resources = Resources()
    resources.add_job("job_1", Job(name="new name", description="new description"))
    resources.add_job("job_2", Job(name="job_2"))

    out = _append_resources(input, resources)

    assert out is not input
    assert out == {
        "resources": {
            "jobs": {
                "job_0": {"name": "job_0"},
                "job_1": {"name": "new name", "description": "new description"},
                "job_2": {"name": "job_2"},
            }
        },
    }


def test_parse_args():
    args = _parse_args(
        [
            "--input",
            "input.json",
            "--output",
            "output.json",
            "--phase",
            "load_resources",
            "--diagnostics",
            "diagnostics.json",
            "--locations",
            "locations.json",
        ]
    )

    assert args == _Args(
        diagnostics="diagnostics.json",
        input="input.json",
        output="output.json",
        phase="load_resources",
        locations="locations.json",
        unknown_args=[],
    )


def test_parse_args_unknown():
    args = _parse_args(
        [
            "--input",
            "input.json",
            "--output",
            "output.json",
            "--phase",
            "load_resources",
            "--unknown",
            "--diagnostics",
            "diagnostics.json",
        ]
    )

    assert args == _Args(
        diagnostics="diagnostics.json",
        input="input.json",
        output="output.json",
        phase="load_resources",
        locations=None,
        unknown_args=["--unknown"],
    )
