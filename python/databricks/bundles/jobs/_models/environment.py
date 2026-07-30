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
from databricks.bundles.jobs._models.cluster_spec import ClusterSpec, ClusterSpecDict, ClusterSpecParam
from databricks.bundles.jobs._models.dbfs_storage_info import DbfsStorageInfo, DbfsStorageInfoDict, DbfsStorageInfoParam
from databricks.bundles.jobs._models.docker_basic_auth import DockerBasicAuth, DockerBasicAuthDict, DockerBasicAuthParam
from databricks.bundles.jobs._models.docker_image import DockerImage, DockerImageDict, DockerImageParam
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
class Environment:
    """
    The environment entity used to preserve serverless environment side panel, jobs' environment for non-notebook task, and SDP's environment for classic and serverless pipelines.
    In this minimal environment spec, only pip and java dependencies are supported.
    """
    base_environment: VariableOrOptional[str] = None
    """
    The base environment this environment is built on top of. A base environment defines the environment version and a
    list of dependencies for serverless compute. The value can be a file path to a custom `env.yaml` file
    (e.g., `/Workspace/path/to/env.yaml`). Support for a Databricks-provided base environment ID
    (e.g., `workspace-base-environments/databricks_ai_v4`) and workspace base environment ID
    (e.g., `workspace-base-environments/dbe_b849b66e-b31a-4cb5-b161-1f2b10877fb7`) is in Beta.
    Either `environment_version` or `base_environment` can be provided.
    For more information about Databricks-provided base environments, see the
    [list workspace base environments](:method:Environments/ListWorkspaceBaseEnvironments) API.
    For more information, see
    """

    client: VariableOrOptional[str] = None
    """
    [DEPRECATED] Use `environment_version` instead.
    """

    dependencies: VariableOrList[str] = field(default_factory=list)
    """
    List of pip dependencies, as supported by the version of pip in this environment.
    """

    environment_version: VariableOrOptional[str] = None
    """
    Either `environment_version` or `base_environment` needs to be provided. Environment version used by the environment.
    Each version comes with a specific Python version and a set of Python packages.
    The version is a string, consisting of an integer.
    """

    java_dependencies: VariableOrList[str] = field(default_factory=list)
    """
    [Public Preview] List of java dependencies. Each dependency is a string representing a java library path. For example: `/Volumes/path/to/test.jar`.
    """

    @classmethod
    def from_dict(cls, value: 'EnvironmentDict') -> 'Self':
        return _transform(cls, value)

    def as_dict(self) -> 'EnvironmentDict':
        return _transform_to_json_value(self) # type:ignore



class EnvironmentDict(TypedDict, total=False):
    """"""

    base_environment: VariableOrOptional[str]
    """
    The base environment this environment is built on top of. A base environment defines the environment version and a
    list of dependencies for serverless compute. The value can be a file path to a custom `env.yaml` file
    (e.g., `/Workspace/path/to/env.yaml`). Support for a Databricks-provided base environment ID
    (e.g., `workspace-base-environments/databricks_ai_v4`) and workspace base environment ID
    (e.g., `workspace-base-environments/dbe_b849b66e-b31a-4cb5-b161-1f2b10877fb7`) is in Beta.
    Either `environment_version` or `base_environment` can be provided.
    For more information about Databricks-provided base environments, see the
    [list workspace base environments](:method:Environments/ListWorkspaceBaseEnvironments) API.
    For more information, see
    """

    client: VariableOrOptional[str]
    """
    [DEPRECATED] Use `environment_version` instead.
    """

    dependencies: VariableOrList[str]
    """
    List of pip dependencies, as supported by the version of pip in this environment.
    """

    environment_version: VariableOrOptional[str]
    """
    Either `environment_version` or `base_environment` needs to be provided. Environment version used by the environment.
    Each version comes with a specific Python version and a set of Python packages.
    The version is a string, consisting of an integer.
    """

    java_dependencies: VariableOrList[str]
    """
    [Public Preview] List of java dependencies. Each dependency is a string representing a java library path. For example: `/Volumes/path/to/test.jar`.
    """


EnvironmentParam = EnvironmentDict | Environment
