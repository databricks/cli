from typing import TYPE_CHECKING, Optional

from databricks.bundles.core._diagnostics import Diagnostics
from databricks.bundles.core._location import Location
from databricks.bundles.core._resource import Resource
from databricks.bundles.core._transform import _transform

if TYPE_CHECKING:
    from databricks.bundles.jobs._models.job import Job, JobParam
    from databricks.bundles.pipelines._models.pipeline import Pipeline, PipelineParam

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

        experimental:
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
        self._jobs = dict[str, "Job"]()
        self._pipelines = dict[str, "Pipeline"]()
        self._locations = dict[tuple[str, ...], Location]()
        self._diagnostics = Diagnostics()

    @property
    def jobs(self) -> dict[str, "Job"]:
        return self._jobs

    @property
    def pipelines(self) -> dict[str, "Pipeline"]:
        return self._pipelines

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

        from databricks.bundles.jobs import Job
        from databricks.bundles.pipelines import Pipeline

        location = location or Location.from_stack_frame(depth=1)

        match resource:
            case Job():
                self.add_job(resource_name, resource, location=location)
            case Pipeline():
                self.add_pipeline(resource_name, resource, location=location)
            case _:
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

        if self._jobs.get(resource_name):
            self.add_diagnostic_error(
                msg=f"Duplicate resource name '{resource_name}' for a job. Resource names must be unique.",
                location=location,
                path=path,
            )
        else:
            if location:
                self.add_location(path, location)

            self._jobs[resource_name] = job

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

        if self._pipelines.get(resource_name):
            self.add_diagnostic_error(
                msg=f"Duplicate resource name '{resource_name}' for a pipeline. Resource names must be unique.",
                location=location,
                path=path,
            )
        else:
            if location:
                self.add_location(path, location)

            self._pipelines[resource_name] = pipeline

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
        for name, job in other.jobs.items():
            self.add_job(name, job)

        for name, pipeline in other.pipelines.items():
            self.add_pipeline(name, pipeline)

        for path, location in other._locations.items():
            self.add_location(path, location)

        self._diagnostics = self._diagnostics.extend(other._diagnostics)
