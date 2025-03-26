from dataclasses import replace

from databricks.bundles.core import job_mutator, variables, Variable
from databricks.bundles.jobs import Job

@variables
class Variables:
    string_variable: Variable[str]

@job_mutator
def test_variables_1(job: Job) -> Job:
    # even though we have output name=Variables.string_variable in load_resources,
    # it should be resolved to "abc" before it's passed to the mutator

    assert job.name == "abc"

    return replace(job, name=Variables.string_variable)


@job_mutator
def test_variables_2(job: Job) -> Job:
    # now, this makes no sense, because variable isn't going to be resolved
    # while it was resolved for the first mutator

    assert job.name == Variables.string_variable

    return replace(job, name=Variables.string_variable)
