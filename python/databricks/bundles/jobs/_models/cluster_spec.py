from typing import Literal, Optional, TypedDict, ClassVar, TYPE_CHECKING
from enum import Enum
from dataclasses import dataclass, replace, field

from databricks.bundles.core._resource import Resource
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional, VariableOrList, VariableOrDict

from databricks.bundles.jobs._models.adlsgen2_info import Adlsgen2Info, Adlsgen2InfoDict, Adlsgen2InfoParam
from databricks.bundles.jobs._models.auto_scale import AutoScale, AutoScaleDict, AutoScaleParam
from databricks.bundles.jobs._models.aws_attributes import AwsAttributes, AwsAttributesDict, AwsAttributesParam
from databricks.bundles.jobs._models.azure_attributes import AzureAttributes, AzureAttributesDict, AzureAttributesParam
from databricks.bundles.jobs._models.clients_types import ClientsTypes, ClientsTypesDict, ClientsTypesParam
from databricks.bundles.jobs._models.cluster_log_conf import ClusterLogConf, ClusterLogConfDict, ClusterLogConfParam
from databricks.bundles.jobs._models.dbfs_storage_info import DbfsStorageInfo, DbfsStorageInfoDict, DbfsStorageInfoParam
from databricks.bundles.jobs._models.docker_basic_auth import DockerBasicAuth, DockerBasicAuthDict, DockerBasicAuthParam
from databricks.bundles.jobs._models.docker_image import DockerImage, DockerImageDict, DockerImageParam
from databricks.bundles.jobs._models.environment import Environment, EnvironmentDict, EnvironmentParam
from databricks.bundles.jobs._models.gcp_attributes import GcpAttributes, GcpAttributesDict, GcpAttributesParam
from databricks.bundles.jobs._models.gcs_storage_info import GcsStorageInfo, GcsStorageInfoDict, GcsStorageInfoParam
from databricks.bundles.jobs._models.init_script_info import InitScriptInfo, InitScriptInfoDict, InitScriptInfoParam
from databricks.bundles.jobs._models.library import Library, LibraryDict, LibraryParam
from databricks.bundles.jobs._models.local_file_info import LocalFileInfo, LocalFileInfoDict, LocalFileInfoParam
from databricks.bundles.jobs._models.log_analytics_info import LogAnalyticsInfo, LogAnalyticsInfoDict, LogAnalyticsInfoParam
from databricks.bundles.jobs._models.maven_library import MavenLibrary, MavenLibraryDict, MavenLibraryParam
from databricks.bundles.jobs._models.node_type_flexibility import NodeTypeFlexibility, NodeTypeFlexibilityDict, NodeTypeFlexibilityParam
from databricks.bundles.jobs._models.python_py_pi_library import PythonPyPiLibrary, PythonPyPiLibraryDict, PythonPyPiLibraryParam
from databricks.bundles.jobs._models.r_cran_library import RCranLibrary, RCranLibraryDict, RCranLibraryParam
from databricks.bundles.jobs._models.s3_storage_info import S3StorageInfo, S3StorageInfoDict, S3StorageInfoParam
from databricks.bundles.jobs._models.volumes_storage_info import VolumesStorageInfo, VolumesStorageInfoDict, VolumesStorageInfoParam
from databricks.bundles.jobs._models.workload_type import WorkloadType, WorkloadTypeDict, WorkloadTypeParam
from databricks.bundles.jobs._models.workspace_storage_info import WorkspaceStorageInfo, WorkspaceStorageInfoDict, WorkspaceStorageInfoParam
from databricks.bundles.jobs._models.ai_runtime_task import AiRuntimeTask, AiRuntimeTaskDict, AiRuntimeTaskParam
from databricks.bundles.jobs._models.alert_task import AlertTask, AlertTaskDict, AlertTaskParam
from databricks.bundles.jobs._models.alert_task_subscriber import AlertTaskSubscriber, AlertTaskSubscriberDict, AlertTaskSubscriberParam
from databricks.bundles.jobs._models.clean_rooms_notebook_task import CleanRoomsNotebookTask, CleanRoomsNotebookTaskDict, CleanRoomsNotebookTaskParam
from databricks.bundles.jobs._models.compute import Compute, ComputeDict, ComputeParam
from databricks.bundles.jobs._models.compute_config import ComputeConfig, ComputeConfigDict, ComputeConfigParam
from databricks.bundles.jobs._models.compute_spec import ComputeSpec, ComputeSpecDict, ComputeSpecParam
from databricks.bundles.jobs._models.condition_task import ConditionTask, ConditionTaskDict, ConditionTaskParam
from databricks.bundles.jobs._models.continuous import Continuous, ContinuousDict, ContinuousParam
from databricks.bundles.jobs._models.cron_schedule import CronSchedule, CronScheduleDict, CronScheduleParam
from databricks.bundles.jobs._models.dashboard_task import DashboardTask, DashboardTaskDict, DashboardTaskParam
from databricks.bundles.jobs._models.dbt_platform_task import DbtPlatformTask, DbtPlatformTaskDict, DbtPlatformTaskParam
from databricks.bundles.jobs._models.dbt_task import DbtTask, DbtTaskDict, DbtTaskParam
from databricks.bundles.jobs._models.deployment_spec import DeploymentSpec, DeploymentSpecDict, DeploymentSpecParam
from databricks.bundles.jobs._models.file_arrival_trigger_configuration import FileArrivalTriggerConfiguration, FileArrivalTriggerConfigurationDict, FileArrivalTriggerConfigurationParam
from databricks.bundles.jobs._models.for_each_task import ForEachTask, ForEachTaskDict, ForEachTaskParam
from databricks.bundles.jobs._models.gen_ai_compute_task import GenAiComputeTask, GenAiComputeTaskDict, GenAiComputeTaskParam
from databricks.bundles.jobs._models.git_source import GitSource, GitSourceDict, GitSourceParam
from databricks.bundles.jobs._models.job_cluster import JobCluster, JobClusterDict, JobClusterParam
from databricks.bundles.jobs._models.job_email_notifications import JobEmailNotifications, JobEmailNotificationsDict, JobEmailNotificationsParam
from databricks.bundles.jobs._models.job_environment import JobEnvironment, JobEnvironmentDict, JobEnvironmentParam
from databricks.bundles.jobs._models.job_notification_settings import JobNotificationSettings, JobNotificationSettingsDict, JobNotificationSettingsParam
from databricks.bundles.jobs._models.job_parameter_definition import JobParameterDefinition, JobParameterDefinitionDict, JobParameterDefinitionParam
from databricks.bundles.jobs._models.job_run_as import JobRunAs, JobRunAsDict, JobRunAsParam
from databricks.bundles.jobs._models.jobs_health_rule import JobsHealthRule, JobsHealthRuleDict, JobsHealthRuleParam
from databricks.bundles.jobs._models.jobs_health_rules import JobsHealthRules, JobsHealthRulesDict, JobsHealthRulesParam
from databricks.bundles.jobs._models.model_trigger_configuration import ModelTriggerConfiguration, ModelTriggerConfigurationDict, ModelTriggerConfigurationParam
from databricks.bundles.jobs._models.notebook_task import NotebookTask, NotebookTaskDict, NotebookTaskParam
from databricks.bundles.jobs._models.periodic_trigger_configuration import PeriodicTriggerConfiguration, PeriodicTriggerConfigurationDict, PeriodicTriggerConfigurationParam
from databricks.bundles.jobs._models.pipeline_params import PipelineParams, PipelineParamsDict, PipelineParamsParam
from databricks.bundles.jobs._models.pipeline_task import PipelineTask, PipelineTaskDict, PipelineTaskParam
from databricks.bundles.jobs._models.power_bi_model import PowerBiModel, PowerBiModelDict, PowerBiModelParam
from databricks.bundles.jobs._models.power_bi_table import PowerBiTable, PowerBiTableDict, PowerBiTableParam
from databricks.bundles.jobs._models.power_bi_task import PowerBiTask, PowerBiTaskDict, PowerBiTaskParam
from databricks.bundles.jobs._models.python_operator_task import PythonOperatorTask, PythonOperatorTaskDict, PythonOperatorTaskParam
from databricks.bundles.jobs._models.python_operator_task_parameter import PythonOperatorTaskParameter, PythonOperatorTaskParameterDict, PythonOperatorTaskParameterParam
from databricks.bundles.jobs._models.python_wheel_task import PythonWheelTask, PythonWheelTaskDict, PythonWheelTaskParam
from databricks.bundles.jobs._models.queue_settings import QueueSettings, QueueSettingsDict, QueueSettingsParam
from databricks.bundles.jobs._models.run_job_task import RunJobTask, RunJobTaskDict, RunJobTaskParam
from databricks.bundles.jobs._models.spark_jar_task import SparkJarTask, SparkJarTaskDict, SparkJarTaskParam
from databricks.bundles.jobs._models.spark_python_task import SparkPythonTask, SparkPythonTaskDict, SparkPythonTaskParam
from databricks.bundles.jobs._models.spark_submit_task import SparkSubmitTask, SparkSubmitTaskDict, SparkSubmitTaskParam
from databricks.bundles.jobs._models.sparse_checkout import SparseCheckout, SparseCheckoutDict, SparseCheckoutParam
from databricks.bundles.jobs._models.sql_condition_configuration import SqlConditionConfiguration, SqlConditionConfigurationDict, SqlConditionConfigurationParam
from databricks.bundles.jobs._models.sql_task import SqlTask, SqlTaskDict, SqlTaskParam
from databricks.bundles.jobs._models.sql_task_alert import SqlTaskAlert, SqlTaskAlertDict, SqlTaskAlertParam
from databricks.bundles.jobs._models.sql_task_dashboard import SqlTaskDashboard, SqlTaskDashboardDict, SqlTaskDashboardParam
from databricks.bundles.jobs._models.sql_task_file import SqlTaskFile, SqlTaskFileDict, SqlTaskFileParam
from databricks.bundles.jobs._models.sql_task_query import SqlTaskQuery, SqlTaskQueryDict, SqlTaskQueryParam
from databricks.bundles.jobs._models.sql_task_subscription import SqlTaskSubscription, SqlTaskSubscriptionDict, SqlTaskSubscriptionParam
from databricks.bundles.jobs._models.subscription import Subscription, SubscriptionDict, SubscriptionParam
from databricks.bundles.jobs._models.subscription_subscriber import SubscriptionSubscriber, SubscriptionSubscriberDict, SubscriptionSubscriberParam
from databricks.bundles.jobs._models.table_update_trigger_configuration import TableUpdateTriggerConfiguration, TableUpdateTriggerConfigurationDict, TableUpdateTriggerConfigurationParam
from databricks.bundles.jobs._models.task import Task, TaskDict, TaskParam
from databricks.bundles.jobs._models.task_dependency import TaskDependency, TaskDependencyDict, TaskDependencyParam
from databricks.bundles.jobs._models.task_email_notifications import TaskEmailNotifications, TaskEmailNotificationsDict, TaskEmailNotificationsParam
from databricks.bundles.jobs._models.task_notification_settings import TaskNotificationSettings, TaskNotificationSettingsDict, TaskNotificationSettingsParam
from databricks.bundles.jobs._models.trigger_settings import TriggerSettings, TriggerSettingsDict, TriggerSettingsParam
from databricks.bundles.jobs._models.webhook import Webhook, WebhookDict, WebhookParam
from databricks.bundles.jobs._models.webhook_notifications import WebhookNotifications, WebhookNotificationsDict, WebhookNotificationsParam
from databricks.bundles.jobs._models.job import Job, JobDict, JobParam
from databricks.bundles.jobs._models.job_permission import JobPermission, JobPermissionDict, JobPermissionParam
from databricks.bundles.jobs._models.lifecycle import Lifecycle, LifecycleDict, LifecycleParam
from databricks.bundles.jobs._models.aws_availability import AwsAvailability, AwsAvailabilityParam

from databricks.bundles.jobs._models.azure_availability import AzureAvailability, AzureAvailabilityParam

from databricks.bundles.jobs._models.confidential_compute_type import ConfidentialComputeType, ConfidentialComputeTypeParam

from databricks.bundles.jobs._models.data_security_mode import DataSecurityMode, DataSecurityModeParam

from databricks.bundles.jobs._models.dependency_mode import DependencyMode, DependencyModeParam

from databricks.bundles.jobs._models.ebs_volume_type import EbsVolumeType, EbsVolumeTypeParam

from databricks.bundles.jobs._models.gcp_availability import GcpAvailability, GcpAvailabilityParam

from databricks.bundles.jobs._models.hardware_accelerator_type import HardwareAcceleratorType, HardwareAcceleratorTypeParam

from databricks.bundles.jobs._models.kind import Kind, KindParam

from databricks.bundles.jobs._models.runtime_engine import RuntimeEngine, RuntimeEngineParam

from databricks.bundles.jobs._models.authentication_method import AuthenticationMethod, AuthenticationMethodParam

from databricks.bundles.jobs._models.compute_spec_accelerator_type import ComputeSpecAcceleratorType, ComputeSpecAcceleratorTypeParam

from databricks.bundles.jobs._models.condition import Condition, ConditionParam

from databricks.bundles.jobs._models.condition_task_op import ConditionTaskOp, ConditionTaskOpParam

from databricks.bundles.jobs._models.git_provider import GitProvider, GitProviderParam

from databricks.bundles.jobs._models.job_permission_level import JobPermissionLevel, JobPermissionLevelParam

from databricks.bundles.jobs._models.jobs_health_metric import JobsHealthMetric, JobsHealthMetricParam

from databricks.bundles.jobs._models.jobs_health_operator import JobsHealthOperator, JobsHealthOperatorParam

from databricks.bundles.jobs._models.model_trigger_configuration_condition import ModelTriggerConfigurationCondition, ModelTriggerConfigurationConditionParam

from databricks.bundles.jobs._models.pause_status import PauseStatus, PauseStatusParam

from databricks.bundles.jobs._models.performance_target import PerformanceTarget, PerformanceTargetParam

from databricks.bundles.jobs._models.periodic_trigger_configuration_time_unit import PeriodicTriggerConfigurationTimeUnit, PeriodicTriggerConfigurationTimeUnitParam

from databricks.bundles.jobs._models.run_if import RunIf, RunIfParam

from databricks.bundles.jobs._models.source import Source, SourceParam

from databricks.bundles.jobs._models.sql_condition_trigger_mode import SqlConditionTriggerMode, SqlConditionTriggerModeParam

from databricks.bundles.jobs._models.storage_mode import StorageMode, StorageModeParam

from databricks.bundles.jobs._models.task_retry_mode import TaskRetryMode, TaskRetryModeParam


if TYPE_CHECKING:
    from typing_extensions import Self

@dataclass(kw_only=True)
class ClusterSpec:
    """
    Contains a snapshot of the latest user specified settings that were used to create/edit the cluster.
    """
    apply_policy_default_values: VariableOrOptional[bool] = None
    """
    When set to true, fixed and default values from the policy will be used for fields that are omitted. When set to false, only fixed values from the policy will be applied.
    """

    autoscale: VariableOrOptional[AutoScale] = None
    """
    Parameters needed in order to automatically scale clusters up and down based on load.
    Note: autoscaling works best with DB runtime versions 3.0 or later.
    """

    autotermination_minutes: VariableOrOptional[int] = None
    """
    Automatically terminates the cluster after it is inactive for this time in minutes. If not set,
    this cluster will not be automatically terminated. If specified, the threshold must be between
    10 and 10000 minutes.
    Users can also set this value to 0 to explicitly disable automatic termination.
    """

    aws_attributes: VariableOrOptional[AwsAttributes] = None
    """
    Attributes related to clusters running on Amazon Web Services.
    If not specified at cluster creation, a set of default values will be used.
    """

    azure_attributes: VariableOrOptional[AzureAttributes] = None
    """
    Attributes related to clusters running on Microsoft Azure.
    If not specified at cluster creation, a set of default values will be used.
    """

    cluster_log_conf: VariableOrOptional[ClusterLogConf] = None
    """
    The configuration for delivering spark logs to a long-term storage destination.
    Three kinds of destinations (DBFS, S3 and Unity Catalog volumes) are supported. Only one destination can be specified
    for one cluster. If the conf is given, the logs will be delivered to the destination every
    `5 mins`. The destination of driver logs is `$destination/$clusterId/driver`, while
    the destination of executor logs is `$destination/$clusterId/executor`.
    """

    cluster_name: VariableOrOptional[str] = None
    """
    Cluster name requested by the user. This doesn't have to be unique.
    If not specified at creation, the cluster name will be an empty string.
    For job clusters, the cluster name is automatically set based on the job and job run IDs.
    """

    custom_tags: VariableOrDict[str] = field(default_factory=dict)
    """
    Additional tags for cluster resources. Databricks will tag all cluster resources (e.g., AWS
    instances and EBS volumes) with these tags in addition to `default_tags`. Notes:
    
    - Currently, Databricks allows at most 45 custom tags
    
    - Clusters can only reuse cloud resources if the resources' tags are a subset of the cluster tags
    """

    data_security_mode: VariableOrOptional[DataSecurityMode] = None
    """
    Data security mode decides what data governance model to use when accessing data
    from a cluster.
    
    * `DATA_SECURITY_MODE_AUTO`: Databricks will choose the most appropriate access mode depending on your compute configuration.
    * `DATA_SECURITY_MODE_STANDARD`: A secure cluster that can be shared by multiple users. Cluster users are fully isolated so that they cannot see each other’s data and credentials. Most data governance features are supported in this mode. But programming languages and cluster features might be limited.
    * `DATA_SECURITY_MODE_DEDICATED`: A secure cluster that can only be exclusively used by a single user specified in `single_user_name`. Most programming languages, cluster features and data governance features are available in this mode.
    
    The following modes are legacy aliases for the above modes:
    
    * `USER_ISOLATION`: Legacy alias for `DATA_SECURITY_MODE_STANDARD`.
    * `SINGLE_USER`: Legacy alias for `DATA_SECURITY_MODE_DEDICATED`.
    
    The following modes are deprecated starting with Databricks Runtime 15.0 and
    will be removed for future Databricks Runtime versions:
    
    * `LEGACY_TABLE_ACL`: This mode is for users migrating from legacy Table ACL clusters.
    * `LEGACY_PASSTHROUGH`: This mode is for users migrating from legacy Passthrough on high concurrency clusters.
    * `LEGACY_SINGLE_USER`: This mode is for users migrating from legacy Passthrough on standard clusters.
    * `LEGACY_SINGLE_USER_STANDARD`: This mode provides a way that doesn’t have UC nor passthrough enabled.
    """

    dependency_mode: VariableOrOptional[DependencyMode] = None
    """
    [Beta] Controls dependency configuration for the cluster.
    """

    docker_image: VariableOrOptional[DockerImage] = None
    """
    Custom docker image BYOC
    """

    driver_instance_pool_id: VariableOrOptional[str] = None
    """
    The optional ID of the instance pool for the driver of the cluster belongs.
    The pool cluster uses the instance pool with id (instance_pool_id) if the driver pool is not
    assigned.
    """

    driver_node_type_flexibility: VariableOrOptional[NodeTypeFlexibility] = None
    """
    Flexible node type configuration for the driver node.
    """

    driver_node_type_id: VariableOrOptional[str] = None
    """
    The node type of the Spark driver.
    Note that this field is optional; if unset, the driver node type will be set as the same value
    as `node_type_id` defined above.
    
    This field, along with node_type_id, should not be set if virtual_cluster_size is set.
    If both driver_node_type_id, node_type_id, and virtual_cluster_size are specified, driver_node_type_id and node_type_id take precedence.
    """

    enable_elastic_disk: VariableOrOptional[bool] = None
    """
    Autoscaling Local Storage: when enabled, this cluster will dynamically acquire additional disk
    space when its Spark workers are running low on disk space.
    """

    enable_local_disk_encryption: VariableOrOptional[bool] = None
    """
    Whether to enable LUKS on cluster VMs' local disks
    """

    gcp_attributes: VariableOrOptional[GcpAttributes] = None
    """
    Attributes related to clusters running on Google Cloud Platform.
    If not specified at cluster creation, a set of default values will be used.
    """

    init_scripts: VariableOrList[InitScriptInfo] = field(default_factory=list)
    """
    The configuration for storing init scripts. Any number of destinations can be specified.
    The scripts are executed sequentially in the order provided.
    If `cluster_log_conf` is specified, init script logs are sent to `<destination>/<cluster-ID>/init_scripts`.
    """

    instance_pool_id: VariableOrOptional[str] = None
    """
    The optional ID of the instance pool to which the cluster belongs.
    """

    is_single_node: VariableOrOptional[bool] = None
    """
    This field can only be used when `kind = CLASSIC_PREVIEW`.
    
    When set to true, Databricks will automatically set single node related `custom_tags`, `spark_conf`, and `num_workers`
    """

    kind: VariableOrOptional[Kind] = None
    """
    The kind of compute described by this compute specification.
    
    Depending on `kind`, different validations and default values will be applied.
    
    Clusters with `kind = CLASSIC_PREVIEW` support the following fields, whereas clusters with no specified `kind` do not.
    * [is_single_node](/api/workspace/clusters/create#is_single_node)
    * [use_ml_runtime](/api/workspace/clusters/create#use_ml_runtime)
    
    By using the [simple form](https://docs.databricks.com/compute/simple-form.html), your clusters are automatically using `kind = CLASSIC_PREVIEW`.
    """

    node_type_id: VariableOrOptional[str] = None
    """
    This field encodes, through a single value, the resources available to each of
    the Spark nodes in this cluster. For example, the Spark nodes can be provisioned
    and optimized for memory or compute intensive workloads. A list of available node
    types can be retrieved by using the [clusters/listNodeTypes](https://docs.databricks.com/api/workspace/clusters/listnodetypes) API call.
    """

    num_workers: VariableOrOptional[int] = None
    """
    Number of worker nodes that this cluster should have. A cluster has one Spark Driver
    and `num_workers` Executors for a total of `num_workers` + 1 Spark nodes.
    
    Note: When reading the properties of a cluster, this field reflects the desired number
    of workers rather than the actual current number of workers. For instance, if a cluster
    is resized from 5 to 10 workers, this field will immediately be updated to reflect
    the target size of 10 workers, whereas the workers listed in `spark_info` will gradually
    increase from 5 to 10 as the new nodes are provisioned.
    """

    policy_id: VariableOrOptional[str] = None
    """
    The ID of the cluster policy used to create the cluster if applicable.
    """

    remote_disk_throughput: VariableOrOptional[int] = None
    """
    If set, what the configurable throughput (in Mb/s) for the remote disk is. Currently only supported for GCP HYPERDISK_BALANCED disks.
    """

    runtime_engine: VariableOrOptional[RuntimeEngine] = None
    """
    Determines the cluster's runtime engine, either standard or Photon.
    
    This field is not compatible with legacy `spark_version` values that contain `-photon-`.
    Remove `-photon-` from the `spark_version` and set `runtime_engine` to `PHOTON`.
    
    If left unspecified, the runtime engine defaults to standard unless the spark_version
    contains -photon-, in which case Photon will be used.
    """

    single_user_name: VariableOrOptional[str] = None
    """
    Single user name if data_security_mode is `SINGLE_USER`
    """

    spark_conf: VariableOrDict[str] = field(default_factory=dict)
    """
    An object containing a set of optional, user-specified Spark configuration key-value pairs.
    Users can also pass in a string of extra JVM options to the driver and the executors via
    `spark.driver.extraJavaOptions` and `spark.executor.extraJavaOptions` respectively.
    """

    spark_env_vars: VariableOrDict[str] = field(default_factory=dict)
    """
    An object containing a set of optional, user-specified environment variable key-value pairs.
    Please note that key-value pair of the form (X,Y) will be exported as is (i.e.,
    `export X='Y'`) while launching the driver and workers.
    
    In order to specify an additional set of `SPARK_DAEMON_JAVA_OPTS`, we recommend appending
    them to `$SPARK_DAEMON_JAVA_OPTS` as shown in the example below. This ensures that all
    default databricks managed environmental variables are included as well.
    
    Example Spark environment variables:
    `{"SPARK_WORKER_MEMORY": "28000m", "SPARK_LOCAL_DIRS": "/local_disk0"}` or
    `{"SPARK_DAEMON_JAVA_OPTS": "$SPARK_DAEMON_JAVA_OPTS -Dspark.shuffle.service.enabled=true"}`
    """

    spark_version: VariableOrOptional[str] = None
    """
    The Spark version of the cluster, e.g. `3.3.x-scala2.11`.
    A list of available Spark versions can be retrieved by using
    the [clusters/sparkVersions](https://docs.databricks.com/api/workspace/clusters/sparkversions) API call.
    """

    ssh_public_keys: VariableOrList[str] = field(default_factory=list)
    """
    SSH public key contents that will be added to each Spark node in this cluster. The
    corresponding private keys can be used to login with the user name `ubuntu` on port `2200`.
    Up to 10 keys can be specified.
    """

    total_initial_remote_disk_size: VariableOrOptional[int] = None
    """
    If set, what the total initial volume size (in GB) of the remote disks should be. Currently only supported for GCP HYPERDISK_BALANCED disks.
    """

    use_ml_runtime: VariableOrOptional[bool] = None
    """
    This field can only be used when `kind = CLASSIC_PREVIEW`.
    
    `effective_spark_version` is determined by `spark_version` (DBR release), this field `use_ml_runtime`, and whether `node_type_id` is gpu node or not.
    """

    worker_node_type_flexibility: VariableOrOptional[NodeTypeFlexibility] = None
    """
    Flexible node type configuration for worker nodes.
    """

    workload_type: VariableOrOptional[WorkloadType] = None
    """
    Cluster Attributes showing for clusters workload types.
    """

    @classmethod
    def from_dict(cls, value: 'ClusterSpecDict') -> 'Self':
        return _transform(cls, value)

    def as_dict(self) -> 'ClusterSpecDict':
        return _transform_to_json_value(self) # type:ignore



class ClusterSpecDict(TypedDict, total=False):
    """"""

    apply_policy_default_values: VariableOrOptional[bool]
    """
    When set to true, fixed and default values from the policy will be used for fields that are omitted. When set to false, only fixed values from the policy will be applied.
    """

    autoscale: VariableOrOptional[AutoScaleParam]
    """
    Parameters needed in order to automatically scale clusters up and down based on load.
    Note: autoscaling works best with DB runtime versions 3.0 or later.
    """

    autotermination_minutes: VariableOrOptional[int]
    """
    Automatically terminates the cluster after it is inactive for this time in minutes. If not set,
    this cluster will not be automatically terminated. If specified, the threshold must be between
    10 and 10000 minutes.
    Users can also set this value to 0 to explicitly disable automatic termination.
    """

    aws_attributes: VariableOrOptional[AwsAttributesParam]
    """
    Attributes related to clusters running on Amazon Web Services.
    If not specified at cluster creation, a set of default values will be used.
    """

    azure_attributes: VariableOrOptional[AzureAttributesParam]
    """
    Attributes related to clusters running on Microsoft Azure.
    If not specified at cluster creation, a set of default values will be used.
    """

    cluster_log_conf: VariableOrOptional[ClusterLogConfParam]
    """
    The configuration for delivering spark logs to a long-term storage destination.
    Three kinds of destinations (DBFS, S3 and Unity Catalog volumes) are supported. Only one destination can be specified
    for one cluster. If the conf is given, the logs will be delivered to the destination every
    `5 mins`. The destination of driver logs is `$destination/$clusterId/driver`, while
    the destination of executor logs is `$destination/$clusterId/executor`.
    """

    cluster_name: VariableOrOptional[str]
    """
    Cluster name requested by the user. This doesn't have to be unique.
    If not specified at creation, the cluster name will be an empty string.
    For job clusters, the cluster name is automatically set based on the job and job run IDs.
    """

    custom_tags: VariableOrDict[str]
    """
    Additional tags for cluster resources. Databricks will tag all cluster resources (e.g., AWS
    instances and EBS volumes) with these tags in addition to `default_tags`. Notes:
    
    - Currently, Databricks allows at most 45 custom tags
    
    - Clusters can only reuse cloud resources if the resources' tags are a subset of the cluster tags
    """

    data_security_mode: VariableOrOptional[DataSecurityModeParam]
    """
    Data security mode decides what data governance model to use when accessing data
    from a cluster.
    
    * `DATA_SECURITY_MODE_AUTO`: Databricks will choose the most appropriate access mode depending on your compute configuration.
    * `DATA_SECURITY_MODE_STANDARD`: A secure cluster that can be shared by multiple users. Cluster users are fully isolated so that they cannot see each other’s data and credentials. Most data governance features are supported in this mode. But programming languages and cluster features might be limited.
    * `DATA_SECURITY_MODE_DEDICATED`: A secure cluster that can only be exclusively used by a single user specified in `single_user_name`. Most programming languages, cluster features and data governance features are available in this mode.
    
    The following modes are legacy aliases for the above modes:
    
    * `USER_ISOLATION`: Legacy alias for `DATA_SECURITY_MODE_STANDARD`.
    * `SINGLE_USER`: Legacy alias for `DATA_SECURITY_MODE_DEDICATED`.
    
    The following modes are deprecated starting with Databricks Runtime 15.0 and
    will be removed for future Databricks Runtime versions:
    
    * `LEGACY_TABLE_ACL`: This mode is for users migrating from legacy Table ACL clusters.
    * `LEGACY_PASSTHROUGH`: This mode is for users migrating from legacy Passthrough on high concurrency clusters.
    * `LEGACY_SINGLE_USER`: This mode is for users migrating from legacy Passthrough on standard clusters.
    * `LEGACY_SINGLE_USER_STANDARD`: This mode provides a way that doesn’t have UC nor passthrough enabled.
    """

    dependency_mode: VariableOrOptional[DependencyModeParam]
    """
    [Beta] Controls dependency configuration for the cluster.
    """

    docker_image: VariableOrOptional[DockerImageParam]
    """
    Custom docker image BYOC
    """

    driver_instance_pool_id: VariableOrOptional[str]
    """
    The optional ID of the instance pool for the driver of the cluster belongs.
    The pool cluster uses the instance pool with id (instance_pool_id) if the driver pool is not
    assigned.
    """

    driver_node_type_flexibility: VariableOrOptional[NodeTypeFlexibilityParam]
    """
    Flexible node type configuration for the driver node.
    """

    driver_node_type_id: VariableOrOptional[str]
    """
    The node type of the Spark driver.
    Note that this field is optional; if unset, the driver node type will be set as the same value
    as `node_type_id` defined above.
    
    This field, along with node_type_id, should not be set if virtual_cluster_size is set.
    If both driver_node_type_id, node_type_id, and virtual_cluster_size are specified, driver_node_type_id and node_type_id take precedence.
    """

    enable_elastic_disk: VariableOrOptional[bool]
    """
    Autoscaling Local Storage: when enabled, this cluster will dynamically acquire additional disk
    space when its Spark workers are running low on disk space.
    """

    enable_local_disk_encryption: VariableOrOptional[bool]
    """
    Whether to enable LUKS on cluster VMs' local disks
    """

    gcp_attributes: VariableOrOptional[GcpAttributesParam]
    """
    Attributes related to clusters running on Google Cloud Platform.
    If not specified at cluster creation, a set of default values will be used.
    """

    init_scripts: VariableOrList[InitScriptInfoParam]
    """
    The configuration for storing init scripts. Any number of destinations can be specified.
    The scripts are executed sequentially in the order provided.
    If `cluster_log_conf` is specified, init script logs are sent to `<destination>/<cluster-ID>/init_scripts`.
    """

    instance_pool_id: VariableOrOptional[str]
    """
    The optional ID of the instance pool to which the cluster belongs.
    """

    is_single_node: VariableOrOptional[bool]
    """
    This field can only be used when `kind = CLASSIC_PREVIEW`.
    
    When set to true, Databricks will automatically set single node related `custom_tags`, `spark_conf`, and `num_workers`
    """

    kind: VariableOrOptional[KindParam]
    """
    The kind of compute described by this compute specification.
    
    Depending on `kind`, different validations and default values will be applied.
    
    Clusters with `kind = CLASSIC_PREVIEW` support the following fields, whereas clusters with no specified `kind` do not.
    * [is_single_node](/api/workspace/clusters/create#is_single_node)
    * [use_ml_runtime](/api/workspace/clusters/create#use_ml_runtime)
    
    By using the [simple form](https://docs.databricks.com/compute/simple-form.html), your clusters are automatically using `kind = CLASSIC_PREVIEW`.
    """

    node_type_id: VariableOrOptional[str]
    """
    This field encodes, through a single value, the resources available to each of
    the Spark nodes in this cluster. For example, the Spark nodes can be provisioned
    and optimized for memory or compute intensive workloads. A list of available node
    types can be retrieved by using the [clusters/listNodeTypes](https://docs.databricks.com/api/workspace/clusters/listnodetypes) API call.
    """

    num_workers: VariableOrOptional[int]
    """
    Number of worker nodes that this cluster should have. A cluster has one Spark Driver
    and `num_workers` Executors for a total of `num_workers` + 1 Spark nodes.
    
    Note: When reading the properties of a cluster, this field reflects the desired number
    of workers rather than the actual current number of workers. For instance, if a cluster
    is resized from 5 to 10 workers, this field will immediately be updated to reflect
    the target size of 10 workers, whereas the workers listed in `spark_info` will gradually
    increase from 5 to 10 as the new nodes are provisioned.
    """

    policy_id: VariableOrOptional[str]
    """
    The ID of the cluster policy used to create the cluster if applicable.
    """

    remote_disk_throughput: VariableOrOptional[int]
    """
    If set, what the configurable throughput (in Mb/s) for the remote disk is. Currently only supported for GCP HYPERDISK_BALANCED disks.
    """

    runtime_engine: VariableOrOptional[RuntimeEngineParam]
    """
    Determines the cluster's runtime engine, either standard or Photon.
    
    This field is not compatible with legacy `spark_version` values that contain `-photon-`.
    Remove `-photon-` from the `spark_version` and set `runtime_engine` to `PHOTON`.
    
    If left unspecified, the runtime engine defaults to standard unless the spark_version
    contains -photon-, in which case Photon will be used.
    """

    single_user_name: VariableOrOptional[str]
    """
    Single user name if data_security_mode is `SINGLE_USER`
    """

    spark_conf: VariableOrDict[str]
    """
    An object containing a set of optional, user-specified Spark configuration key-value pairs.
    Users can also pass in a string of extra JVM options to the driver and the executors via
    `spark.driver.extraJavaOptions` and `spark.executor.extraJavaOptions` respectively.
    """

    spark_env_vars: VariableOrDict[str]
    """
    An object containing a set of optional, user-specified environment variable key-value pairs.
    Please note that key-value pair of the form (X,Y) will be exported as is (i.e.,
    `export X='Y'`) while launching the driver and workers.
    
    In order to specify an additional set of `SPARK_DAEMON_JAVA_OPTS`, we recommend appending
    them to `$SPARK_DAEMON_JAVA_OPTS` as shown in the example below. This ensures that all
    default databricks managed environmental variables are included as well.
    
    Example Spark environment variables:
    `{"SPARK_WORKER_MEMORY": "28000m", "SPARK_LOCAL_DIRS": "/local_disk0"}` or
    `{"SPARK_DAEMON_JAVA_OPTS": "$SPARK_DAEMON_JAVA_OPTS -Dspark.shuffle.service.enabled=true"}`
    """

    spark_version: VariableOrOptional[str]
    """
    The Spark version of the cluster, e.g. `3.3.x-scala2.11`.
    A list of available Spark versions can be retrieved by using
    the [clusters/sparkVersions](https://docs.databricks.com/api/workspace/clusters/sparkversions) API call.
    """

    ssh_public_keys: VariableOrList[str]
    """
    SSH public key contents that will be added to each Spark node in this cluster. The
    corresponding private keys can be used to login with the user name `ubuntu` on port `2200`.
    Up to 10 keys can be specified.
    """

    total_initial_remote_disk_size: VariableOrOptional[int]
    """
    If set, what the total initial volume size (in GB) of the remote disks should be. Currently only supported for GCP HYPERDISK_BALANCED disks.
    """

    use_ml_runtime: VariableOrOptional[bool]
    """
    This field can only be used when `kind = CLASSIC_PREVIEW`.
    
    `effective_spark_version` is determined by `spark_version` (DBR release), this field `use_ml_runtime`, and whether `node_type_id` is gpu node or not.
    """

    worker_node_type_flexibility: VariableOrOptional[NodeTypeFlexibilityParam]
    """
    Flexible node type configuration for worker nodes.
    """

    workload_type: VariableOrOptional[WorkloadTypeParam]
    """
    Cluster Attributes showing for clusters workload types.
    """


ClusterSpecParam = ClusterSpecDict | ClusterSpec
