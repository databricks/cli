from dataclasses import dataclass
from typing import Type

from databricks.bundles.core._resource import Resource


@dataclass(kw_only=True, frozen=True)
class _ResourceType:
    """
    NB: this class should stay internal-only and NOT be exported from databricks.bundles.core
    """

    resource_type: Type[Resource]

    singular_name: str
    """
    Singular name, should be used in methods (e.g. "add_job"), error messages and as parameter names.
    """

    plural_name: str
    """
    Plural name, the same as in "resources" bundle section.
    """

    @classmethod
    def all(cls) -> tuple["_ResourceType", ...]:
        """
        Returns all supported resource types.
        """

        # intentionally lazily load all resource types to avoid imports from databricks.bundles.core to
        # be imported in databricks.bundles.<resource_type>

        from databricks.bundles.jobs._models.job import Job
        from databricks.bundles.pipelines._models.pipeline import Pipeline

        return (
            _ResourceType(
                resource_type=Job,
                singular_name="job",
                plural_name="jobs",
            ),
            _ResourceType(
                resource_type=Pipeline,
                plural_name="pipelines",
                singular_name="pipeline",
            ),
        )
