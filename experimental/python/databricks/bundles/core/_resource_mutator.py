from collections.abc import Callable
from dataclasses import dataclass
from typing import TYPE_CHECKING, Generic, Type, TypeVar, overload

from databricks.bundles.core._bundle import Bundle
from databricks.bundles.core._resource import Resource

if TYPE_CHECKING:
    from databricks.bundles.jobs._models.job import Job
    from databricks.bundles.pipelines._models.pipeline import Pipeline

_T = TypeVar("_T", bound=Resource)


@dataclass(frozen=True)
class ResourceMutator(Generic[_T]):
    """
    Resource mutators are used to modify resources before they are deployed.

    Mutators are applied both to resources defined in YAML and Python.
    Mutators are applied in the order they are defined in databricks.yml.

    Example:

        .. code-block:: yaml

            experimental:
                python:
                    mutators:
                    - "resources:my_job_mutator"

        .. code-block:: python

            from databricks.bundles.core import Bundle, job_mutator
            from databricks.bundles.jobs import Job


            @job_mutator
            def my_job_mutator(bundle: Bundle, job: Job) -> Job:
                return replace(job, name="my_job")

    See :meth:`databricks.bundles.core.job_mutator`.
    """

    resource_type: Type[_T]
    """
    Resource type that this mutator applies to.
    """

    function: Callable
    """
    Underling function that was decorated. Can be accessed for unit-testing.
    """


# Below, we define decorators for each resource type. This approach allows us
# to implement mutators that are only applied for specific resource types.
#
# Alternative approaches considered and rejected during design:
#
# - Inspecting type annotations without decorators.
#   Rationale: Avoid implicit runtime behavior changes based solely on type annotations,
#   especially if a function lacks an explicit decorator.
#
# - Using a universal @mutator decorator.
#   Rationale: Determining whether a mutator is invoked based solely on type annotations
#   was deemed overly implicit and potentially confusing.


@overload
def job_mutator(
    function: Callable[[Bundle, "Job"], "Job"],
) -> ResourceMutator["Job"]: ...


@overload
def job_mutator(function: Callable[["Job"], "Job"]) -> ResourceMutator["Job"]: ...


def job_mutator(function: Callable) -> ResourceMutator["Job"]:
    """
    Decorator for defining a job mutator. Function should return a new instance of the job with the desired changes,
    instead of mutating the input job.

    Example:

    .. code-block:: python

        @job_mutator
        def my_job_mutator(bundle: Bundle, job: Job) -> Job:
            return replace(job, name="my_job")

    :param function: Function that mutates a job.
    """
    from databricks.bundles.jobs._models.job import Job

    return ResourceMutator(resource_type=Job, function=function)


@overload
def pipeline_mutator(
    function: Callable[[Bundle, "Pipeline"], "Pipeline"],
) -> ResourceMutator["Pipeline"]: ...


@overload
def pipeline_mutator(
    function: Callable[["Pipeline"], "Pipeline"],
) -> ResourceMutator["Pipeline"]: ...


def pipeline_mutator(function: Callable) -> ResourceMutator["Pipeline"]:
    """
    Decorator for defining a pipeline mutator. Function should return a new instance of the pipeline with the desired changes,
    instead of mutating the input pipeline.

    Example:

    .. code-block:: python

        @pipeline_mutator
        def my_pipeline_mutator(bundle: Bundle, pipeline: Pipeline) -> Pipeline:
            return replace(pipeline, name="my_job")

    :param function: Function that mutates a pipeline.
    """
    from databricks.bundles.pipelines._models.pipeline import Pipeline

    return ResourceMutator(resource_type=Pipeline, function=function)
