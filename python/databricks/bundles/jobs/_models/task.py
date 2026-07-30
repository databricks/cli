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
class Task:
    """"""

    task_key: VariableOr[str]
    """
    A unique name for the task. This field is used to refer to this task from other tasks.
    This field is required and must be unique within its parent job.
    On Update or Reset, this field is used to reference the tasks to be updated or reset.
    """

    ai_runtime_task: VariableOrOptional[AiRuntimeTask] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] The task runs a multi-node GPU compute workload on Databricks AI Runtime.
    External-facing surface; mirrors the AIR CLI (fka SGCLI) v2 YAML schema.
    """

    alert_task: VariableOrOptional[AlertTask] = None
    """
    [Public Preview] The task evaluates a Databricks alert and sends notifications to subscribers
    when the `alert_task` field is present.
    """

    clean_rooms_notebook_task: VariableOrOptional[CleanRoomsNotebookTask] = None
    """
    The task runs a [clean rooms](https://docs.databricks.com/clean-rooms/index.html) notebook
    when the `clean_rooms_notebook_task` field is present.
    """

    compute: VariableOrOptional[Compute] = None
    """
    [Beta] Task level compute configuration.
    """

    condition_task: VariableOrOptional[ConditionTask] = None
    """
    The task evaluates a condition that can be used to control the execution of other tasks when the `condition_task` field is present.
    The condition task does not require a cluster to execute and does not support retries or notifications.
    """

    dashboard_task: VariableOrOptional[DashboardTask] = None
    """
    The task refreshes a dashboard and sends a snapshot to subscribers.
    """

    dbt_platform_task: VariableOrOptional[DbtPlatformTask] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview]
    """

    dbt_task: VariableOrOptional[DbtTask] = None
    """
    The task runs one or more dbt commands when the `dbt_task` field is present. The dbt task requires both Databricks SQL and the ability to use a serverless or a pro SQL warehouse.
    """

    depends_on: VariableOrList[TaskDependency] = field(default_factory=list)
    """
    An optional array of objects specifying the dependency graph of the task. All tasks specified in this field must complete before executing this task. The task will run only if the `run_if` condition is true.
    The key is `task_key`, and the value is the name assigned to the dependent task.
    """

    description: VariableOrOptional[str] = None
    """
    An optional description for this task.
    """

    disable_auto_optimization: VariableOrOptional[bool] = None
    """
    An option to disable auto optimization in serverless
    """

    disabled: VariableOrOptional[bool] = None
    """
    An optional flag to disable the task. If set to true, the task will not run even if it is part of a job.
    """

    email_notifications: VariableOrOptional[TaskEmailNotifications] = None
    """
    An optional set of email addresses that is notified when runs of this task begin or complete as well as when this task is deleted. The default behavior is to not send any emails.
    """

    environment_key: VariableOrOptional[str] = None
    """
    The key that references an environment spec in a job. This field is required for Python script, Python wheel and dbt tasks when using serverless compute.
    """

    existing_cluster_id: VariableOrOptional[str] = None
    """
    If existing_cluster_id, the ID of an existing cluster that is used for all runs.
    When running jobs or tasks on an existing cluster, you may need to manually restart
    the cluster if it stops responding. We suggest running jobs and tasks on new clusters for
    greater reliability
    """

    for_each_task: VariableOrOptional[ForEachTask] = None
    """
    The task executes a nested task for every input provided when the `for_each_task` field is present.
    """

    gen_ai_compute_task: VariableOrOptional[GenAiComputeTask] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] DEPRECATED — use `AiRuntimeTask` for all new BYOT multi-node GPU
    workloads (see ai_runtime_task.proto). `AiRuntimeTask` is the only
    supported BYOT task type for new workloads; this proto is retained only
    for AIR CLI (fka SGCLI) pywheel backwards compatibility and will be
    removed once the pywheel → databricks-cli migration completes (post-
    PuPr).
    """

    health: VariableOrOptional[JobsHealthRules] = None
    """
    An optional set of health rules that can be defined for this job.
    """

    job_cluster_key: VariableOrOptional[str] = None
    """
    If job_cluster_key, this task is executed reusing the cluster specified in `job.settings.job_clusters`.
    """

    libraries: VariableOrList[Library] = field(default_factory=list)
    """
    An optional list of libraries to be installed on the cluster.
    The default value is an empty list.
    """

    max_retries: VariableOrOptional[int] = None
    """
    An optional maximum number of times to retry an unsuccessful run. A run is considered to be unsuccessful if it completes with the `FAILED` result_state or `INTERNAL_ERROR` `life_cycle_state`. The value `-1` means to retry indefinitely and the value `0` means to never retry.
    """

    min_retry_interval_millis: VariableOrOptional[int] = None
    """
    An optional minimal interval in milliseconds between the start of the failed run and the subsequent retry run. The default behavior is that unsuccessful runs are immediately retried.
    """

    new_cluster: VariableOrOptional[ClusterSpec] = None
    """
    If new_cluster, a description of a new cluster that is created for each run.
    """

    notebook_task: VariableOrOptional[NotebookTask] = None
    """
    The task runs a notebook when the `notebook_task` field is present.
    """

    notification_settings: VariableOrOptional[TaskNotificationSettings] = None
    """
    Optional notification settings that are used when sending notifications to each of the `email_notifications` and `webhook_notifications` for this task.
    """

    pipeline_task: VariableOrOptional[PipelineTask] = None
    """
    The task triggers a pipeline update when the `pipeline_task` field is present. Only pipelines configured to use triggered more are supported.
    """

    power_bi_task: VariableOrOptional[PowerBiTask] = None
    """
    [Public Preview] The task triggers a Power BI semantic model update when the `power_bi_task` field is present.
    """

    python_operator_task: VariableOrOptional[PythonOperatorTask] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] The task runs a Python operator task.
    """

    python_wheel_task: VariableOrOptional[PythonWheelTask] = None
    """
    The task runs a Python wheel when the `python_wheel_task` field is present.
    """

    retry_on_timeout: VariableOrOptional[bool] = None
    """
    An optional policy to specify whether to retry a job when it times out. The default behavior
    is to not retry on timeout.
    """

    run_if: VariableOrOptional[RunIf] = None
    """
    An optional value specifying the condition determining whether the task is run once its dependencies have been completed.
    
    * `ALL_SUCCESS`: All dependencies have executed and succeeded
    * `AT_LEAST_ONE_SUCCESS`: At least one dependency has succeeded
    * `NONE_FAILED`: None of the dependencies have failed and at least one was executed
    * `ALL_DONE`: All dependencies have been completed
    * `AT_LEAST_ONE_FAILED`: At least one dependency failed
    * `ALL_FAILED`: ALl dependencies have failed
    """

    run_job_task: VariableOrOptional[RunJobTask] = None
    """
    The task triggers another job when the `run_job_task` field is present.
    """

    spark_jar_task: VariableOrOptional[SparkJarTask] = None
    """
    The task runs a JAR when the `spark_jar_task` field is present.
    """

    spark_python_task: VariableOrOptional[SparkPythonTask] = None
    """
    The task runs a Python file when the `spark_python_task` field is present.
    """

    spark_submit_task: VariableOrOptional[SparkSubmitTask] = None
    """
    [DEPRECATED] (Legacy) The task runs the spark-submit script when the spark_submit_task field is present. Databricks recommends using the spark_jar_task instead; see [Spark Submit task for jobs](/jobs/spark-submit).
    """

    sql_task: VariableOrOptional[SqlTask] = None
    """
    The task runs a SQL query or file, or it refreshes a SQL alert or a legacy SQL dashboard when the `sql_task` field is present.
    """

    timeout_seconds: VariableOrOptional[int] = None
    """
    An optional timeout applied to each run of this job task. A value of `0` means no timeout.
    """

    webhook_notifications: VariableOrOptional[WebhookNotifications] = None
    """
    A collection of system notification IDs to notify when runs of this task begin or complete. The default behavior is to not send any system notifications.
    """

    @classmethod
    def from_dict(cls, value: 'TaskDict') -> 'Self':
        return _transform(cls, value)

    def as_dict(self) -> 'TaskDict':
        return _transform_to_json_value(self) # type:ignore



class TaskDict(TypedDict, total=False):
    """"""

    task_key: VariableOr[str]
    """
    A unique name for the task. This field is used to refer to this task from other tasks.
    This field is required and must be unique within its parent job.
    On Update or Reset, this field is used to reference the tasks to be updated or reset.
    """

    ai_runtime_task: VariableOrOptional[AiRuntimeTaskParam]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] The task runs a multi-node GPU compute workload on Databricks AI Runtime.
    External-facing surface; mirrors the AIR CLI (fka SGCLI) v2 YAML schema.
    """

    alert_task: VariableOrOptional[AlertTaskParam]
    """
    [Public Preview] The task evaluates a Databricks alert and sends notifications to subscribers
    when the `alert_task` field is present.
    """

    clean_rooms_notebook_task: VariableOrOptional[CleanRoomsNotebookTaskParam]
    """
    The task runs a [clean rooms](https://docs.databricks.com/clean-rooms/index.html) notebook
    when the `clean_rooms_notebook_task` field is present.
    """

    compute: VariableOrOptional[ComputeParam]
    """
    [Beta] Task level compute configuration.
    """

    condition_task: VariableOrOptional[ConditionTaskParam]
    """
    The task evaluates a condition that can be used to control the execution of other tasks when the `condition_task` field is present.
    The condition task does not require a cluster to execute and does not support retries or notifications.
    """

    dashboard_task: VariableOrOptional[DashboardTaskParam]
    """
    The task refreshes a dashboard and sends a snapshot to subscribers.
    """

    dbt_platform_task: VariableOrOptional[DbtPlatformTaskParam]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview]
    """

    dbt_task: VariableOrOptional[DbtTaskParam]
    """
    The task runs one or more dbt commands when the `dbt_task` field is present. The dbt task requires both Databricks SQL and the ability to use a serverless or a pro SQL warehouse.
    """

    depends_on: VariableOrList[TaskDependencyParam]
    """
    An optional array of objects specifying the dependency graph of the task. All tasks specified in this field must complete before executing this task. The task will run only if the `run_if` condition is true.
    The key is `task_key`, and the value is the name assigned to the dependent task.
    """

    description: VariableOrOptional[str]
    """
    An optional description for this task.
    """

    disable_auto_optimization: VariableOrOptional[bool]
    """
    An option to disable auto optimization in serverless
    """

    disabled: VariableOrOptional[bool]
    """
    An optional flag to disable the task. If set to true, the task will not run even if it is part of a job.
    """

    email_notifications: VariableOrOptional[TaskEmailNotificationsParam]
    """
    An optional set of email addresses that is notified when runs of this task begin or complete as well as when this task is deleted. The default behavior is to not send any emails.
    """

    environment_key: VariableOrOptional[str]
    """
    The key that references an environment spec in a job. This field is required for Python script, Python wheel and dbt tasks when using serverless compute.
    """

    existing_cluster_id: VariableOrOptional[str]
    """
    If existing_cluster_id, the ID of an existing cluster that is used for all runs.
    When running jobs or tasks on an existing cluster, you may need to manually restart
    the cluster if it stops responding. We suggest running jobs and tasks on new clusters for
    greater reliability
    """

    for_each_task: VariableOrOptional[ForEachTaskParam]
    """
    The task executes a nested task for every input provided when the `for_each_task` field is present.
    """

    gen_ai_compute_task: VariableOrOptional[GenAiComputeTaskParam]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] DEPRECATED — use `AiRuntimeTask` for all new BYOT multi-node GPU
    workloads (see ai_runtime_task.proto). `AiRuntimeTask` is the only
    supported BYOT task type for new workloads; this proto is retained only
    for AIR CLI (fka SGCLI) pywheel backwards compatibility and will be
    removed once the pywheel → databricks-cli migration completes (post-
    PuPr).
    """

    health: VariableOrOptional[JobsHealthRulesParam]
    """
    An optional set of health rules that can be defined for this job.
    """

    job_cluster_key: VariableOrOptional[str]
    """
    If job_cluster_key, this task is executed reusing the cluster specified in `job.settings.job_clusters`.
    """

    libraries: VariableOrList[LibraryParam]
    """
    An optional list of libraries to be installed on the cluster.
    The default value is an empty list.
    """

    max_retries: VariableOrOptional[int]
    """
    An optional maximum number of times to retry an unsuccessful run. A run is considered to be unsuccessful if it completes with the `FAILED` result_state or `INTERNAL_ERROR` `life_cycle_state`. The value `-1` means to retry indefinitely and the value `0` means to never retry.
    """

    min_retry_interval_millis: VariableOrOptional[int]
    """
    An optional minimal interval in milliseconds between the start of the failed run and the subsequent retry run. The default behavior is that unsuccessful runs are immediately retried.
    """

    new_cluster: VariableOrOptional[ClusterSpecParam]
    """
    If new_cluster, a description of a new cluster that is created for each run.
    """

    notebook_task: VariableOrOptional[NotebookTaskParam]
    """
    The task runs a notebook when the `notebook_task` field is present.
    """

    notification_settings: VariableOrOptional[TaskNotificationSettingsParam]
    """
    Optional notification settings that are used when sending notifications to each of the `email_notifications` and `webhook_notifications` for this task.
    """

    pipeline_task: VariableOrOptional[PipelineTaskParam]
    """
    The task triggers a pipeline update when the `pipeline_task` field is present. Only pipelines configured to use triggered more are supported.
    """

    power_bi_task: VariableOrOptional[PowerBiTaskParam]
    """
    [Public Preview] The task triggers a Power BI semantic model update when the `power_bi_task` field is present.
    """

    python_operator_task: VariableOrOptional[PythonOperatorTaskParam]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] The task runs a Python operator task.
    """

    python_wheel_task: VariableOrOptional[PythonWheelTaskParam]
    """
    The task runs a Python wheel when the `python_wheel_task` field is present.
    """

    retry_on_timeout: VariableOrOptional[bool]
    """
    An optional policy to specify whether to retry a job when it times out. The default behavior
    is to not retry on timeout.
    """

    run_if: VariableOrOptional[RunIfParam]
    """
    An optional value specifying the condition determining whether the task is run once its dependencies have been completed.
    
    * `ALL_SUCCESS`: All dependencies have executed and succeeded
    * `AT_LEAST_ONE_SUCCESS`: At least one dependency has succeeded
    * `NONE_FAILED`: None of the dependencies have failed and at least one was executed
    * `ALL_DONE`: All dependencies have been completed
    * `AT_LEAST_ONE_FAILED`: At least one dependency failed
    * `ALL_FAILED`: ALl dependencies have failed
    """

    run_job_task: VariableOrOptional[RunJobTaskParam]
    """
    The task triggers another job when the `run_job_task` field is present.
    """

    spark_jar_task: VariableOrOptional[SparkJarTaskParam]
    """
    The task runs a JAR when the `spark_jar_task` field is present.
    """

    spark_python_task: VariableOrOptional[SparkPythonTaskParam]
    """
    The task runs a Python file when the `spark_python_task` field is present.
    """

    spark_submit_task: VariableOrOptional[SparkSubmitTaskParam]
    """
    [DEPRECATED] (Legacy) The task runs the spark-submit script when the spark_submit_task field is present. Databricks recommends using the spark_jar_task instead; see [Spark Submit task for jobs](/jobs/spark-submit).
    """

    sql_task: VariableOrOptional[SqlTaskParam]
    """
    The task runs a SQL query or file, or it refreshes a SQL alert or a legacy SQL dashboard when the `sql_task` field is present.
    """

    timeout_seconds: VariableOrOptional[int]
    """
    An optional timeout applied to each run of this job task. A value of `0` means no timeout.
    """

    webhook_notifications: VariableOrOptional[WebhookNotificationsParam]
    """
    A collection of system notification IDs to notify when runs of this task begin or complete. The default behavior is to not send any system notifications.
    """


TaskParam = TaskDict | Task
