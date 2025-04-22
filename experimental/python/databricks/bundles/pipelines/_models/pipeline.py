from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._resource import Resource
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOrDict,
    VariableOrList,
    VariableOrOptional,
)
from databricks.bundles.pipelines._models.event_log_spec import (
    EventLogSpec,
    EventLogSpecParam,
)
from databricks.bundles.pipelines._models.filters import (
    Filters,
    FiltersParam,
)
from databricks.bundles.pipelines._models.ingestion_gateway_pipeline_definition import (
    IngestionGatewayPipelineDefinition,
    IngestionGatewayPipelineDefinitionParam,
)
from databricks.bundles.pipelines._models.ingestion_pipeline_definition import (
    IngestionPipelineDefinition,
    IngestionPipelineDefinitionParam,
)
from databricks.bundles.pipelines._models.notifications import (
    Notifications,
    NotificationsParam,
)
from databricks.bundles.pipelines._models.pipeline_cluster import (
    PipelineCluster,
    PipelineClusterParam,
)
from databricks.bundles.pipelines._models.pipeline_library import (
    PipelineLibrary,
    PipelineLibraryParam,
)
from databricks.bundles.pipelines._models.pipeline_permission import (
    PipelinePermission,
    PipelinePermissionParam,
)
from databricks.bundles.pipelines._models.restart_window import (
    RestartWindow,
    RestartWindowParam,
)
from databricks.bundles.pipelines._models.run_as import RunAs, RunAsParam

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class Pipeline(Resource):
    """"""

    budget_policy_id: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    Budget policy of this pipeline.
    """

    catalog: VariableOrOptional[str] = None
    """
    A catalog in Unity Catalog to publish data from this pipeline to. If `target` is specified, tables in this pipeline are published to a `target` schema inside `catalog` (for example, `catalog`.`target`.`table`). If `target` is not specified, no data is published to Unity Catalog.
    """

    channel: VariableOrOptional[str] = None
    """
    DLT Release Channel that specifies which version to use.
    """

    clusters: VariableOrList[PipelineCluster] = field(default_factory=list)
    """
    Cluster settings for this pipeline deployment.
    """

    configuration: VariableOrDict[str] = field(default_factory=dict)
    """
    String-String configuration for this pipeline execution.
    """

    continuous: VariableOrOptional[bool] = None
    """
    Whether the pipeline is continuous or triggered. This replaces `trigger`.
    """

    development: VariableOrOptional[bool] = None
    """
    Whether the pipeline is in Development mode. Defaults to false.
    """

    edition: VariableOrOptional[str] = None
    """
    Pipeline product edition.
    """

    event_log: VariableOrOptional[EventLogSpec] = None
    """
    Event log configuration for this pipeline
    """

    filters: VariableOrOptional[Filters] = None
    """
    Filters on which Pipeline packages to include in the deployed graph.
    """

    gateway_definition: VariableOrOptional[IngestionGatewayPipelineDefinition] = None
    """
    :meta private: [EXPERIMENTAL]
    
    The definition of a gateway pipeline to support change data capture.
    """

    id: VariableOrOptional[str] = None
    """
    Unique identifier for this pipeline.
    """

    ingestion_definition: VariableOrOptional[IngestionPipelineDefinition] = None
    """
    The configuration for a managed ingestion pipeline. These settings cannot be used with the 'libraries', 'schema', 'target', or 'catalog' settings.
    """

    libraries: VariableOrList[PipelineLibrary] = field(default_factory=list)
    """
    Libraries or code needed by this deployment.
    """

    name: VariableOrOptional[str] = None
    """
    Friendly identifier for this pipeline.
    """

    notifications: VariableOrList[Notifications] = field(default_factory=list)
    """
    List of notification settings for this pipeline.
    """

    permissions: VariableOrList[PipelinePermission] = field(default_factory=list)

    photon: VariableOrOptional[bool] = None
    """
    Whether Photon is enabled for this pipeline.
    """

    restart_window: VariableOrOptional[RestartWindow] = None
    """
    :meta private: [EXPERIMENTAL]
    
    Restart window of this pipeline.
    """

    run_as: VariableOrOptional[RunAs] = None
    """
    :meta private: [EXPERIMENTAL]
    """

    schema: VariableOrOptional[str] = None
    """
    The default schema (database) where tables are read from or published to.
    """

    serverless: VariableOrOptional[bool] = None
    """
    Whether serverless compute is enabled for this pipeline.
    """

    storage: VariableOrOptional[str] = None
    """
    DBFS root directory for storing checkpoints and tables.
    """

    target: VariableOrOptional[str] = None
    """
    Target schema (database) to add tables in this pipeline to. Exactly one of `schema` or `target` must be specified. To publish to Unity Catalog, also specify `catalog`. This legacy field is deprecated for pipeline creation in favor of the `schema` field.
    """

    @classmethod
    def from_dict(cls, value: "PipelineDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "PipelineDict":
        return _transform_to_json_value(self)  # type:ignore


class PipelineDict(TypedDict, total=False):
    """"""

    budget_policy_id: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    Budget policy of this pipeline.
    """

    catalog: VariableOrOptional[str]
    """
    A catalog in Unity Catalog to publish data from this pipeline to. If `target` is specified, tables in this pipeline are published to a `target` schema inside `catalog` (for example, `catalog`.`target`.`table`). If `target` is not specified, no data is published to Unity Catalog.
    """

    channel: VariableOrOptional[str]
    """
    DLT Release Channel that specifies which version to use.
    """

    clusters: VariableOrList[PipelineClusterParam]
    """
    Cluster settings for this pipeline deployment.
    """

    configuration: VariableOrDict[str]
    """
    String-String configuration for this pipeline execution.
    """

    continuous: VariableOrOptional[bool]
    """
    Whether the pipeline is continuous or triggered. This replaces `trigger`.
    """

    development: VariableOrOptional[bool]
    """
    Whether the pipeline is in Development mode. Defaults to false.
    """

    edition: VariableOrOptional[str]
    """
    Pipeline product edition.
    """

    event_log: VariableOrOptional[EventLogSpecParam]
    """
    Event log configuration for this pipeline
    """

    filters: VariableOrOptional[FiltersParam]
    """
    Filters on which Pipeline packages to include in the deployed graph.
    """

    gateway_definition: VariableOrOptional[IngestionGatewayPipelineDefinitionParam]
    """
    :meta private: [EXPERIMENTAL]
    
    The definition of a gateway pipeline to support change data capture.
    """

    id: VariableOrOptional[str]
    """
    Unique identifier for this pipeline.
    """

    ingestion_definition: VariableOrOptional[IngestionPipelineDefinitionParam]
    """
    The configuration for a managed ingestion pipeline. These settings cannot be used with the 'libraries', 'schema', 'target', or 'catalog' settings.
    """

    libraries: VariableOrList[PipelineLibraryParam]
    """
    Libraries or code needed by this deployment.
    """

    name: VariableOrOptional[str]
    """
    Friendly identifier for this pipeline.
    """

    notifications: VariableOrList[NotificationsParam]
    """
    List of notification settings for this pipeline.
    """

    permissions: VariableOrList[PipelinePermissionParam]

    photon: VariableOrOptional[bool]
    """
    Whether Photon is enabled for this pipeline.
    """

    restart_window: VariableOrOptional[RestartWindowParam]
    """
    :meta private: [EXPERIMENTAL]
    
    Restart window of this pipeline.
    """

    run_as: VariableOrOptional[RunAsParam]
    """
    :meta private: [EXPERIMENTAL]
    """

    schema: VariableOrOptional[str]
    """
    The default schema (database) where tables are read from or published to.
    """

    serverless: VariableOrOptional[bool]
    """
    Whether serverless compute is enabled for this pipeline.
    """

    storage: VariableOrOptional[str]
    """
    DBFS root directory for storing checkpoints and tables.
    """

    target: VariableOrOptional[str]
    """
    Target schema (database) to add tables in this pipeline to. Exactly one of `schema` or `target` must be specified. To publish to Unity Catalog, also specify `catalog`. This legacy field is deprecated for pipeline creation in favor of the `schema` field.
    """


PipelineParam = PipelineDict | Pipeline
