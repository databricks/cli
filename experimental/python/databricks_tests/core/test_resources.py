from databricks.bundles.core import Location, Resources, Severity
from databricks.bundles.jobs._models.job import Job


def test_add_job():
    resources = Resources()

    resources.add_job("my_job", Job(name="My job"))

    assert resources.jobs == {"my_job": Job(name="My job")}


def test_add_job_dict():
    resources = Resources()

    resources.add_job("my_job", {"name": "My job"})

    assert resources.jobs == {"my_job": Job(name="My job")}


def test_add_job_location():
    resources = Resources()
    location = Location(file="my_file", line=1, column=2)

    resources.add_job("my_job", Job(name="My job"), location=location)

    assert resources._locations == {("resources", "jobs", "my_job"): location}


def test_add_job_location_automatic():
    resources = Resources()

    resources.add_job("my_job", Job(name="My job"))

    assert resources._locations.keys() == {("resources", "jobs", "my_job")}
    [location] = resources._locations.values()

    assert location.file == __file__
    assert location.line and location.line > 0
    assert location.column and location.column > 0


def test_add_resource_job():
    resources = Resources()

    resources.add_resource("my_job", Job(name="My job"))

    assert resources.jobs == {"my_job": Job(name="My job")}


def test_add_duplicate_job():
    resources = Resources()

    resources.add_job("my_job", {"name": "My job"})
    resources.add_job("my_job", {"name": "My job (2)"})

    # it's important not to override jobs, because, for instance, they can come from YAML
    assert resources.jobs == {"my_job": Job(name="My job")}

    assert len(resources.diagnostics.items) == 1
    [item] = resources.diagnostics.items

    assert item.severity == Severity.ERROR
    assert (
        item.summary
        == "Duplicate resource name 'my_job' for a job. Resource names must be unique."
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
