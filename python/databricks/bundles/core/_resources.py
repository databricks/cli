from typing import Optional

from databricks.bundles.core._diagnostics import Diagnostics
from databricks.bundles.core._generated import _GeneratedResources
from databricks.bundles.core._location import Location
from databricks.bundles.core._resource import Resource
from databricks.bundles.core._resource_type import _ResourceType

__all__ = ["Resources"]


class Resources(_GeneratedResources):
    """
    Resources is a collection of resources in a bundle.

    Resources class is returned by 'load_resources' function specified in databricks.yml. Each element in
    'python/resources' list is a fully qualified function name that returns an instance of Resources class.
    If there are multiple functions specified in 'python/resources' list, the resources from all functions
    are combined into a single Resources object.

    Example:

    .. code-block:: yaml

        python:
          resources:
            - "resources:load_resources"

    `load_resources` function can be implemented using built-in functions:

    - :meth:`load_resources_from_current_package_module`
    - :meth:`load_resources_from_package_module`
    - :meth:`load_resources_from_modules`
    - :meth:`load_resources_from_module`

    Programmatic construction of resources is supported using :meth:`add_resource` and :meth:`add_job` methods.

    Example:

    .. code-block:: python

        def load_resources(bundle: Bundle) -> Resources:
            resources = Resources()

            for resource_name, config in get_configs():
                job = create_job(config)

                resources.add_job(resource_name, job)

            return resources
    """

    def __init__(self):
        self._resources: dict[str, dict] = {
            resource_type.plural_name: {} for resource_type in _ResourceType.all()
        }
        self._locations = dict[tuple[str, ...], Location]()
        self._diagnostics = Diagnostics()

    @property
    def diagnostics(self) -> Diagnostics:
        """
        Returns diagnostics. If there are any diagnostic errors, bundle validation fails.
        """
        return self._diagnostics

    def add_resource(
        self,
        resource_name: str,
        resource: Resource,
        *,
        location: Optional[Location] = None,
    ) -> None:
        """
        Adds a resource to the collection of resources. Resource name must be unique across all
        resources of the same type.

        :param resource_name: unique identifier for the resource
        :param resource: the resource to add
        :param location: optional location of the resource in the source code
        """

        location = location or Location.from_stack_frame(depth=1)

        for resource_type in _ResourceType.all():
            if isinstance(resource, resource_type.resource_type):
                add_method = getattr(self, f"add_{resource_type.singular_name}")
                add_method(resource_name, resource, location=location)
                return

        raise ValueError(f"Unsupported resource type: {type(resource)}")

    def add_location(self, path: tuple[str, ...], location: Location) -> None:
        """
        Associate source code location with a path in the bundle configuration.
        """
        self._locations[path] = location

    def add_diagnostics(self, other: Diagnostics) -> None:
        """
        Add diagnostics from another Diagnostics object.
        :param other:
        :return:
        """
        self._diagnostics = self._diagnostics.extend(other)

    def add_diagnostic_error(
        self,
        msg: str,
        *,
        detail: Optional[str] = None,
        path: Optional[tuple[str, ...]] = None,
        location: Optional[Location] = None,
    ) -> None:
        """
        Report a diagnostic error. If there are any diagnostic errors, bundle validation fails.

        :param msg: short summary of the error
        :param detail: optional detailed description of the error
        :param path: optional path in bundle configuration where the error occurred
        :param location: optional location in the source code where the error occurred
        """
        self.add_diagnostics(
            Diagnostics.create_error(
                msg=msg,
                location=location,
                detail=detail,
                path=path,
            )
        )

    def add_diagnostic_warning(
        self,
        msg: str,
        *,
        detail: Optional[str] = None,
        path: Optional[tuple[str, ...]] = None,
        location: Optional[Location] = None,
    ) -> None:
        """
        Report a diagnostic warning. Warnings are informational and do not cause bundle validation to fail.

        :param msg: short summary of the warning
        :param detail: optional detailed description of the warning
        :param path: optional path in bundle configuration where the warning occurred
        :param location: optional location in the source code where the warning occurred
        """
        self.add_diagnostics(
            Diagnostics.create_warning(
                msg=msg,
                location=location,
                detail=detail,
                path=path,
            )
        )

    def add_resources(self, other: "Resources") -> None:
        """
        Add resources from another Resources object.

        Adds error to diagnostics if there are duplicate resource names.
        """
        for resource_type in _ResourceType.all():
            add_method = getattr(self, f"add_{resource_type.singular_name}")
            for name, resource in getattr(other, resource_type.plural_name).items():
                add_method(name, resource)

        for path, location in other._locations.items():
            self.add_location(path, location)

        self._diagnostics = self._diagnostics.extend(other._diagnostics)
