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
from databricks.bundles.jobs._models.task import Task, TaskDict, TaskParam
from databricks.bundles.jobs._models.task_dependency import TaskDependency, TaskDependencyDict, TaskDependencyParam
from databricks.bundles.jobs._models.task_email_notifications import TaskEmailNotifications, TaskEmailNotificationsDict, TaskEmailNotificationsParam
from databricks.bundles.jobs._models.task_notification_settings import TaskNotificationSettings, TaskNotificationSettingsDict, TaskNotificationSettingsParam
from databricks.bundles.jobs._models.trigger_settings import TriggerSettings, TriggerSettingsDict, TriggerSettingsParam
from databricks.bundles.jobs._models.webhook import Webhook, WebhookDict, WebhookParam
from databricks.bundles.jobs._models.webhook_notifications import WebhookNotifications, WebhookNotificationsDict, WebhookNotificationsParam
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
class Job(Resource):
    """"""

    budget_policy_id: VariableOrOptional[str] = None
    """
    [Public Preview] The id of the user specified budget policy to use for this job.
    If not specified, a default budget policy may be applied when creating or modifying the job.
    See `effective_budget_policy_id` for the budget policy used by this workload.
    """

    continuous: VariableOrOptional[Continuous] = None
    """
    An optional continuous property for this job. The continuous property will ensure that there is always one run executing. Only one of `schedule` and `continuous` can be used.
    """

    description: VariableOrOptional[str] = None
    """
    An optional description for the job. The maximum length is 27700 characters in UTF-8 encoding.
    """

    email_notifications: VariableOrOptional[JobEmailNotifications] = None
    """
    An optional set of email addresses that is notified when runs of this job begin or complete as well as when this job is deleted.
    """

    environments: VariableOrList[JobEnvironment] = field(default_factory=list)
    """
    A list of task execution environment specifications that can be referenced by serverless tasks of this job.
    For serverless notebook tasks, if the environment_key is not specified, the notebook environment will be used if present. If a jobs environment is specified, it will override the notebook environment.
    For other serverless tasks, the task environment is required to be specified using environment_key in the task settings.
    """

    git_source: VariableOrOptional[GitSource] = None
    """
    An optional specification for a remote Git repository containing the source code used by tasks. Version-controlled source code is supported by notebook, dbt, Python script, and SQL File tasks.
    
    If `git_source` is set, these tasks retrieve the file from the remote repository by default. However, this behavior can be overridden by setting `source` to `WORKSPACE` on the task.
    
    Note: dbt and SQL File tasks support only version-controlled sources. If dbt or SQL File tasks are used, `git_source` must be defined on the job.
    """

    health: VariableOrOptional[JobsHealthRules] = None
    """
    An optional set of health rules that can be defined for this job.
    """

    job_clusters: VariableOrList[JobCluster] = field(default_factory=list)
    """
    A list of job cluster specifications that can be shared and reused by tasks of this job. Libraries cannot be declared in a shared job cluster. You must declare dependent libraries in task settings.
    """

    lifecycle: VariableOrOptional[Lifecycle] = None
    """
    Settings that control the deployment lifecycle of the resource, such as preventing it from being destroyed.
    """

    max_concurrent_runs: VariableOrOptional[int] = None
    """
    An optional maximum allowed number of concurrent runs of the job.
    Set this value if you want to be able to execute multiple runs of the same job concurrently.
    This is useful for example if you trigger your job on a frequent schedule and want to allow consecutive runs to overlap with each other, or if you want to trigger multiple runs which differ by their input parameters.
    This setting affects only new runs. For example, suppose the job’s concurrency is 4 and there are 4 concurrent active runs. Then setting the concurrency to 3 won’t kill any of the active runs.
    However, from then on, new runs are skipped unless there are fewer than 3 active runs.
    This value cannot exceed 1000. Setting this value to `0` causes all new runs to be skipped.
    """

    name: VariableOrOptional[str] = None
    """
    An optional name for the job. The maximum length is 4096 bytes in UTF-8 encoding.
    """

    notification_settings: VariableOrOptional[JobNotificationSettings] = None
    """
    Optional notification settings that are used when sending notifications to each of the `email_notifications` and `webhook_notifications` for this job.
    """

    parameters: VariableOrList[JobParameterDefinition] = field(default_factory=list)
    """
    Job-level parameter definitions
    """

    parent_path: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Path of the job parent folder in workspace file tree. If absent, the job doesn't have a workspace object.
    """

    performance_target: VariableOrOptional[PerformanceTarget] = None
    """
    The performance mode on a serverless job. This field determines the level of compute performance or cost-efficiency for the run.
    The performance target does not apply to tasks that run on Serverless GPU compute.
    
    * `STANDARD`: Enables cost-efficient execution of serverless workloads.
    * `PERFORMANCE_OPTIMIZED`: Prioritizes fast startup and execution times through rapid scaling and optimized cluster performance.
    """

    permissions: VariableOrList[JobPermission] = field(default_factory=list)
    """
    The permissions to apply to this resource.
    """

    queue: VariableOrOptional[QueueSettings] = None
    """
    The queue settings of the job.
    """

    run_as: VariableOrOptional[JobRunAs] = None
    """
    The user or service principal that the job runs as, if specified in the request.
    This field indicates the explicit configuration of `run_as` for the job.
    To find the value in all cases, explicit or implicit, use `run_as_user_name`.
    """

    schedule: VariableOrOptional[CronSchedule] = None
    """
    An optional periodic schedule for this job. The default behavior is that the job only runs when triggered by clicking “Run Now” in the Jobs UI or sending an API request to `runNow`.
    """

    tags: VariableOrDict[str] = field(default_factory=dict)
    """
    A map of tags associated with the job. These are forwarded to the cluster as cluster tags for jobs clusters, and are subject to the same limitations as cluster tags. A maximum of 25 tags can be added to the job.
    """

    tasks: VariableOrList[Task] = field(default_factory=list)
    """
    A list of task specifications to be executed by this job.
    It supports up to 1000 elements in write endpoints (:method:jobs/create, :method:jobs/reset, :method:jobs/update, :method:jobs/submit).
    Read endpoints return only 100 tasks. If more than 100 tasks are available, you can paginate through them using :method:jobs/get. Use the `next_page_token` field at the object root to determine if more results are available.
    """

    timeout_seconds: VariableOrOptional[int] = None
    """
    An optional timeout applied to each run of this job. A value of `0` means no timeout.
    """

    trigger: VariableOrOptional[TriggerSettings] = None
    """
    A configuration to trigger a run when certain conditions are met. The default behavior is that the job runs only when triggered by clicking “Run Now” in the Jobs UI or sending an API request to `runNow`.
    """

    usage_policy_id: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] The id of the user specified usage policy to use for this job.
    If not specified, a default usage policy may be applied when creating or modifying the job.
    See `effective_usage_policy_id` for the usage policy used by this workload.
    """

    webhook_notifications: VariableOrOptional[WebhookNotifications] = None
    """
    A collection of system notification IDs to notify when runs of this job begin or complete.
    """

    @classmethod
    def from_dict(cls, value: 'JobDict') -> 'Self':
        return _transform(cls, value)

    def as_dict(self) -> 'JobDict':
        return _transform_to_json_value(self) # type:ignore



class JobDict(TypedDict, total=False):
    """"""

    budget_policy_id: VariableOrOptional[str]
    """
    [Public Preview] The id of the user specified budget policy to use for this job.
    If not specified, a default budget policy may be applied when creating or modifying the job.
    See `effective_budget_policy_id` for the budget policy used by this workload.
    """

    continuous: VariableOrOptional[ContinuousParam]
    """
    An optional continuous property for this job. The continuous property will ensure that there is always one run executing. Only one of `schedule` and `continuous` can be used.
    """

    description: VariableOrOptional[str]
    """
    An optional description for the job. The maximum length is 27700 characters in UTF-8 encoding.
    """

    email_notifications: VariableOrOptional[JobEmailNotificationsParam]
    """
    An optional set of email addresses that is notified when runs of this job begin or complete as well as when this job is deleted.
    """

    environments: VariableOrList[JobEnvironmentParam]
    """
    A list of task execution environment specifications that can be referenced by serverless tasks of this job.
    For serverless notebook tasks, if the environment_key is not specified, the notebook environment will be used if present. If a jobs environment is specified, it will override the notebook environment.
    For other serverless tasks, the task environment is required to be specified using environment_key in the task settings.
    """

    git_source: VariableOrOptional[GitSourceParam]
    """
    An optional specification for a remote Git repository containing the source code used by tasks. Version-controlled source code is supported by notebook, dbt, Python script, and SQL File tasks.
    
    If `git_source` is set, these tasks retrieve the file from the remote repository by default. However, this behavior can be overridden by setting `source` to `WORKSPACE` on the task.
    
    Note: dbt and SQL File tasks support only version-controlled sources. If dbt or SQL File tasks are used, `git_source` must be defined on the job.
    """

    health: VariableOrOptional[JobsHealthRulesParam]
    """
    An optional set of health rules that can be defined for this job.
    """

    job_clusters: VariableOrList[JobClusterParam]
    """
    A list of job cluster specifications that can be shared and reused by tasks of this job. Libraries cannot be declared in a shared job cluster. You must declare dependent libraries in task settings.
    """

    lifecycle: VariableOrOptional[LifecycleParam]
    """
    Settings that control the deployment lifecycle of the resource, such as preventing it from being destroyed.
    """

    max_concurrent_runs: VariableOrOptional[int]
    """
    An optional maximum allowed number of concurrent runs of the job.
    Set this value if you want to be able to execute multiple runs of the same job concurrently.
    This is useful for example if you trigger your job on a frequent schedule and want to allow consecutive runs to overlap with each other, or if you want to trigger multiple runs which differ by their input parameters.
    This setting affects only new runs. For example, suppose the job’s concurrency is 4 and there are 4 concurrent active runs. Then setting the concurrency to 3 won’t kill any of the active runs.
    However, from then on, new runs are skipped unless there are fewer than 3 active runs.
    This value cannot exceed 1000. Setting this value to `0` causes all new runs to be skipped.
    """

    name: VariableOrOptional[str]
    """
    An optional name for the job. The maximum length is 4096 bytes in UTF-8 encoding.
    """

    notification_settings: VariableOrOptional[JobNotificationSettingsParam]
    """
    Optional notification settings that are used when sending notifications to each of the `email_notifications` and `webhook_notifications` for this job.
    """

    parameters: VariableOrList[JobParameterDefinitionParam]
    """
    Job-level parameter definitions
    """

    parent_path: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Path of the job parent folder in workspace file tree. If absent, the job doesn't have a workspace object.
    """

    performance_target: VariableOrOptional[PerformanceTargetParam]
    """
    The performance mode on a serverless job. This field determines the level of compute performance or cost-efficiency for the run.
    The performance target does not apply to tasks that run on Serverless GPU compute.
    
    * `STANDARD`: Enables cost-efficient execution of serverless workloads.
    * `PERFORMANCE_OPTIMIZED`: Prioritizes fast startup and execution times through rapid scaling and optimized cluster performance.
    """

    permissions: VariableOrList[JobPermissionParam]
    """
    The permissions to apply to this resource.
    """

    queue: VariableOrOptional[QueueSettingsParam]
    """
    The queue settings of the job.
    """

    run_as: VariableOrOptional[JobRunAsParam]
    """
    The user or service principal that the job runs as, if specified in the request.
    This field indicates the explicit configuration of `run_as` for the job.
    To find the value in all cases, explicit or implicit, use `run_as_user_name`.
    """

    schedule: VariableOrOptional[CronScheduleParam]
    """
    An optional periodic schedule for this job. The default behavior is that the job only runs when triggered by clicking “Run Now” in the Jobs UI or sending an API request to `runNow`.
    """

    tags: VariableOrDict[str]
    """
    A map of tags associated with the job. These are forwarded to the cluster as cluster tags for jobs clusters, and are subject to the same limitations as cluster tags. A maximum of 25 tags can be added to the job.
    """

    tasks: VariableOrList[TaskParam]
    """
    A list of task specifications to be executed by this job.
    It supports up to 1000 elements in write endpoints (:method:jobs/create, :method:jobs/reset, :method:jobs/update, :method:jobs/submit).
    Read endpoints return only 100 tasks. If more than 100 tasks are available, you can paginate through them using :method:jobs/get. Use the `next_page_token` field at the object root to determine if more results are available.
    """

    timeout_seconds: VariableOrOptional[int]
    """
    An optional timeout applied to each run of this job. A value of `0` means no timeout.
    """

    trigger: VariableOrOptional[TriggerSettingsParam]
    """
    A configuration to trigger a run when certain conditions are met. The default behavior is that the job runs only when triggered by clicking “Run Now” in the Jobs UI or sending an API request to `runNow`.
    """

    usage_policy_id: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] The id of the user specified usage policy to use for this job.
    If not specified, a default usage policy may be applied when creating or modifying the job.
    See `effective_usage_policy_id` for the usage policy used by this workload.
    """

    webhook_notifications: VariableOrOptional[WebhookNotificationsParam]
    """
    A collection of system notification IDs to notify when runs of this job begin or complete.
    """


JobParam = JobDict | Job
