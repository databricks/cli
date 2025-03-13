from collections.abc import Callable
from dataclasses import dataclass
from typing import TYPE_CHECKING, Generic, Type, TypeVar, overload

from databricks.bundles.core._bundle import Bundle
from databricks.bundles.core._resource import Resource

if TYPE_CHECKING:
    from databricks.bundles.jobs._models.job import Job

_T = TypeVar("_T", bound=Resource)


@dataclass(frozen=True)
class ResourceMutator(Generic[_T]):
    """
    Mutators defined within a single Python module are applied in the order they are defined.
    The relative order of mutators defined in different modules is not guaranteed.

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
