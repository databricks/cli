from collections.abc import Callable
from dataclasses import dataclass
from typing import Generic, Type, TypeVar

from databricks.bundles.core._resource import Resource

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


# A decorator is generated for each resource type (see
# _generated/_resource_mutators.py). This approach allows us to implement
# mutators that are only applied for specific resource types.
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
