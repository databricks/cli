from typing import TYPE_CHECKING, Optional

from databricks.bundles.core._diagnostics import Diagnostics
from databricks.bundles.core._location import Location
from databricks.bundles.core._resource import Resource
from databricks.bundles.core._resource_type import _ResourceType
from databricks.bundles.core._transform import _transform

if TYPE_CHECKING:
    from databricks.bundles.alerts._models.alert import Alert, AlertParam
    from databricks.bundles.catalogs._models.catalog import Catalog, CatalogParam
    from databricks.bundles.jobs._models.job import Job, JobParam
    from databricks.bundles.pipelines._models.pipeline import Pipeline, PipelineParam
    from databricks.bundles.schemas._models.schema import Schema, SchemaParam
    from databricks.bundles.volumes._models.volume import Volume, VolumeParam

__all__ = ["Resources"]


class Resources:
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
    def jobs(self) -> dict[str, "Job"]:
        return self._resources["jobs"]

    @property
    def pipelines(self) -> dict[str, "Pipeline"]:
        return self._resources["pipelines"]

    @property
    def schemas(self) -> dict[str, "Schema"]:
        return self._resources["schemas"]

    @property
    def volumes(self) -> dict[str, "Volume"]:
        return self._resources["volumes"]

    @property
    def diagnostics(self) -> Diagnostics:
        """
        Returns diagnostics. If there are any diagnostic errors, bundle validation fails.
        """
        return self._diagnostics

    @property
    def alerts(self) -> dict[str, "Alert"]:
        return self._resources["alerts"]

    @property
    def catalogs(self) -> dict[str, "Catalog"]:
        return self._resources["catalogs"]

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

    def add_job(
        self,
        resource_name: str,
        job: "JobParam",
        *,
        location: Optional[Location] = None,
    ) -> None:
        """
        Adds a job to the collection of resources. Resource name must be unique across all jobs.

        :param resource_name: unique identifier for the job
        :param job: the job to add, can be Job or dict
        :param location: optional location of the job in the source code
        """
        from databricks.bundles.jobs import Job

        job = _transform(Job, job)
        path = ("resources", "jobs", resource_name)
        location = location or Location.from_stack_frame(depth=1)

        if self._resources["jobs"].get(resource_name):
            self.add_diagnostic_error(
                msg=f"Duplicate resource name '{resource_name}' for resource 'job'. Resource names must be unique.",
                location=location,
                path=path,
            )
        else:
            if location:
                self.add_location(path, location)

            self._resources["jobs"][resource_name] = job

    def add_pipeline(
        self,
        resource_name: str,
        pipeline: "PipelineParam",
        *,
        location: Optional[Location] = None,
    ) -> None:
        """
        Adds a pipeline to the collection of resources. Resource name must be unique across all pipelines.

        :param resource_name: unique identifier for the pipeline
        :param pipeline: the pipeline to add, can be Pipeline or dict
        :param location: optional location of the pipeline in the source code
        """
        from databricks.bundles.pipelines import Pipeline

        pipeline = _transform(Pipeline, pipeline)
        path = ("resources", "pipelines", resource_name)
        location = location or Location.from_stack_frame(depth=1)

        if self._resources["pipelines"].get(resource_name):
            self.add_diagnostic_error(
                msg=f"Duplicate resource name '{resource_name}' for resource 'pipeline'. Resource names must be unique.",
                location=location,
                path=path,
            )
        else:
            if location:
                self.add_location(path, location)

            self._resources["pipelines"][resource_name] = pipeline

    def add_schema(
        self,
        resource_name: str,
        schema: "SchemaParam",
        *,
        location: Optional[Location] = None,
    ) -> None:
        """
        Adds a schema to the collection of resources. Resource name must be unique across all schemas.

        :param resource_name: unique identifier for the schema
        :param schema: the schema to add, can be Schema or dict
        :param location: optional location of the schema in the source code
        """
        from databricks.bundles.schemas import Schema

        schema = _transform(Schema, schema)
        path = ("resources", "schemas", resource_name)
        location = location or Location.from_stack_frame(depth=1)

        if self._resources["schemas"].get(resource_name):
            self.add_diagnostic_error(
                msg=f"Duplicate resource name '{resource_name}' for resource 'schema'. Resource names must be unique.",
                location=location,
                path=path,
            )
        else:
            if location:
                self.add_location(path, location)

            self._resources["schemas"][resource_name] = schema

    def add_volume(
        self,
        resource_name: str,
        volume: "VolumeParam",
        *,
        location: Optional[Location] = None,
    ) -> None:
        """
        Adds a volume to the collection of resources. Resource name must be unique across all volumes.

        :param resource_name: unique identifier for the volume
        :param volume: the volume to add, can be Volume or dict
        :param location: optional location of the volume in the source code
        """
        from databricks.bundles.volumes import Volume

        volume = _transform(Volume, volume)
        path = ("resources", "volumes", resource_name)
        location = location or Location.from_stack_frame(depth=1)

        if self._resources["volumes"].get(resource_name):
            self.add_diagnostic_error(
                msg=f"Duplicate resource name '{resource_name}' for resource 'volume'. Resource names must be unique.",
                location=location,
                path=path,
            )
        else:
            if location:
                self.add_location(path, location)

            self._resources["volumes"][resource_name] = volume

    def add_alert(
        self,
        resource_name: str,
        alert: "AlertParam",
        *,
        location: Optional[Location] = None,
    ) -> None:
        """
        Adds an alert to the collection of resources. Resource name must be unique across all alerts.
        """
        from databricks.bundles.alerts import Alert

        alert = _transform(Alert, alert)
        path = ("resources", "alerts", resource_name)
        location = location or Location.from_stack_frame(depth=1)

        if self._resources["alerts"].get(resource_name):
            self.add_diagnostic_error(
                msg=f"Duplicate resource name '{resource_name}' for resource 'alert'. Resource names must be unique.",
                location=location,
                path=path,
            )
        else:
            if location:
                self.add_location(path, location)

            self._resources["alerts"][resource_name] = alert

    def add_catalog(
        self,
        resource_name: str,
        catalog: "CatalogParam",
        *,
        location: Optional[Location] = None,
    ) -> None:
        """
        Adds a catalog to the collection of resources. Resource name must be unique across all catalogs.

        :param resource_name: unique identifier for the catalog
        :param catalog: the catalog to add, can be Catalog or dict
        :param location: optional location of the catalog in the source code
        """
        from databricks.bundles.catalogs import Catalog

        catalog = _transform(Catalog, catalog)
        path = ("resources", "catalogs", resource_name)
        location = location or Location.from_stack_frame(depth=1)

        if self._resources["catalogs"].get(resource_name):
            self.add_diagnostic_error(
                msg=f"Duplicate resource name '{resource_name}' for resource 'catalog'. Resource names must be unique.",
                location=location,
                path=path,
            )
        else:
            if location:
                self.add_location(path, location)

            self._resources["catalogs"][resource_name] = catalog

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
