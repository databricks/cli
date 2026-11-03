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
        from databricks.bundles.core._generated import _all_resource_types

        return _all_resource_types()
