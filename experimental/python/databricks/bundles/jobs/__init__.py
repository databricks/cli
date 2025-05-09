__all__ = [
    "Adlsgen2Info",
    "Adlsgen2InfoDict",
    "Adlsgen2InfoParam",
    "AuthenticationMethod",
    "AuthenticationMethodParam",
    "AutoScale",
    "AutoScaleDict",
    "AutoScaleParam",
    "AwsAttributes",
    "AwsAttributesDict",
    "AwsAttributesParam",
    "AwsAvailability",
    "AwsAvailabilityParam",
    "AzureAttributes",
    "AzureAttributesDict",
    "AzureAttributesParam",
    "AzureAvailability",
    "AzureAvailabilityParam",
    "CleanRoomsNotebookTask",
    "CleanRoomsNotebookTaskDict",
    "CleanRoomsNotebookTaskParam",
    "ClientsTypes",
    "ClientsTypesDict",
    "ClientsTypesParam",
    "ClusterLogConf",
    "ClusterLogConfDict",
    "ClusterLogConfParam",
    "ClusterSpec",
    "ClusterSpecDict",
    "ClusterSpecParam",
    "ComputeConfig",
    "ComputeConfigDict",
    "ComputeConfigParam",
    "Condition",
    "ConditionParam",
    "ConditionTask",
    "ConditionTaskDict",
    "ConditionTaskOp",
    "ConditionTaskOpParam",
    "ConditionTaskParam",
    "Continuous",
    "ContinuousDict",
    "ContinuousParam",
    "CronSchedule",
    "CronScheduleDict",
    "CronScheduleParam",
    "DashboardTask",
    "DashboardTaskDict",
    "DashboardTaskParam",
    "DataSecurityMode",
    "DataSecurityModeParam",
    "DbfsStorageInfo",
    "DbfsStorageInfoDict",
    "DbfsStorageInfoParam",
    "DbtTask",
    "DbtTaskDict",
    "DbtTaskParam",
    "DockerBasicAuth",
    "DockerBasicAuthDict",
    "DockerBasicAuthParam",
    "DockerImage",
    "DockerImageDict",
    "DockerImageParam",
    "EbsVolumeType",
    "EbsVolumeTypeParam",
    "Environment",
    "EnvironmentDict",
    "EnvironmentParam",
    "FileArrivalTriggerConfiguration",
    "FileArrivalTriggerConfigurationDict",
    "FileArrivalTriggerConfigurationParam",
    "ForEachTask",
    "ForEachTaskDict",
    "ForEachTaskParam",
    "GcpAttributes",
    "GcpAttributesDict",
    "GcpAttributesParam",
    "GcpAvailability",
    "GcpAvailabilityParam",
    "GcsStorageInfo",
    "GcsStorageInfoDict",
    "GcsStorageInfoParam",
    "GenAiComputeTask",
    "GenAiComputeTaskDict",
    "GenAiComputeTaskParam",
    "GitProvider",
    "GitProviderParam",
    "GitSource",
    "GitSourceDict",
    "GitSourceParam",
    "InitScriptInfo",
    "InitScriptInfoDict",
    "InitScriptInfoParam",
    "Job",
    "JobCluster",
    "JobClusterDict",
    "JobClusterParam",
    "JobDict",
    "JobEmailNotifications",
    "JobEmailNotificationsDict",
    "JobEmailNotificationsParam",
    "JobEnvironment",
    "JobEnvironmentDict",
    "JobEnvironmentParam",
    "JobNotificationSettings",
    "JobNotificationSettingsDict",
    "JobNotificationSettingsParam",
    "JobParam",
    "JobParameterDefinition",
    "JobParameterDefinitionDict",
    "JobParameterDefinitionParam",
    "JobPermission",
    "JobPermissionDict",
    "JobPermissionLevel",
    "JobPermissionLevelParam",
    "JobPermissionParam",
    "JobRunAs",
    "JobRunAsDict",
    "JobRunAsParam",
    "JobsHealthMetric",
    "JobsHealthMetricParam",
    "JobsHealthOperator",
    "JobsHealthOperatorParam",
    "JobsHealthRule",
    "JobsHealthRuleDict",
    "JobsHealthRuleParam",
    "JobsHealthRules",
    "JobsHealthRulesDict",
    "JobsHealthRulesParam",
    "Library",
    "LibraryDict",
    "LibraryParam",
    "LocalFileInfo",
    "LocalFileInfoDict",
    "LocalFileInfoParam",
    "LogAnalyticsInfo",
    "LogAnalyticsInfoDict",
    "LogAnalyticsInfoParam",
    "MavenLibrary",
    "MavenLibraryDict",
    "MavenLibraryParam",
    "NotebookTask",
    "NotebookTaskDict",
    "NotebookTaskParam",
    "PauseStatus",
    "PauseStatusParam",
    "PerformanceTarget",
    "PerformanceTargetParam",
    "PeriodicTriggerConfiguration",
    "PeriodicTriggerConfigurationDict",
    "PeriodicTriggerConfigurationParam",
    "PeriodicTriggerConfigurationTimeUnit",
    "PeriodicTriggerConfigurationTimeUnitParam",
    "PipelineParams",
    "PipelineParamsDict",
    "PipelineParamsParam",
    "PipelineTask",
    "PipelineTaskDict",
    "PipelineTaskParam",
    "PowerBiModel",
    "PowerBiModelDict",
    "PowerBiModelParam",
    "PowerBiTable",
    "PowerBiTableDict",
    "PowerBiTableParam",
    "PowerBiTask",
    "PowerBiTaskDict",
    "PowerBiTaskParam",
    "PythonPyPiLibrary",
    "PythonPyPiLibraryDict",
    "PythonPyPiLibraryParam",
    "PythonWheelTask",
    "PythonWheelTaskDict",
    "PythonWheelTaskParam",
    "QueueSettings",
    "QueueSettingsDict",
    "QueueSettingsParam",
    "RCranLibrary",
    "RCranLibraryDict",
    "RCranLibraryParam",
    "RunIf",
    "RunIfParam",
    "RunJobTask",
    "RunJobTaskDict",
    "RunJobTaskParam",
    "RuntimeEngine",
    "RuntimeEngineParam",
    "S3StorageInfo",
    "S3StorageInfoDict",
    "S3StorageInfoParam",
    "Source",
    "SourceParam",
    "SparkJarTask",
    "SparkJarTaskDict",
    "SparkJarTaskParam",
    "SparkPythonTask",
    "SparkPythonTaskDict",
    "SparkPythonTaskParam",
    "SparkSubmitTask",
    "SparkSubmitTaskDict",
    "SparkSubmitTaskParam",
    "SqlTask",
    "SqlTaskAlert",
    "SqlTaskAlertDict",
    "SqlTaskAlertParam",
    "SqlTaskDashboard",
    "SqlTaskDashboardDict",
    "SqlTaskDashboardParam",
    "SqlTaskDict",
    "SqlTaskFile",
    "SqlTaskFileDict",
    "SqlTaskFileParam",
    "SqlTaskParam",
    "SqlTaskQuery",
    "SqlTaskQueryDict",
    "SqlTaskQueryParam",
    "SqlTaskSubscription",
    "SqlTaskSubscriptionDict",
    "SqlTaskSubscriptionParam",
    "StorageMode",
    "StorageModeParam",
    "Subscription",
    "SubscriptionDict",
    "SubscriptionParam",
    "SubscriptionSubscriber",
    "SubscriptionSubscriberDict",
    "SubscriptionSubscriberParam",
    "TableUpdateTriggerConfiguration",
    "TableUpdateTriggerConfigurationDict",
    "TableUpdateTriggerConfigurationParam",
    "Task",
    "TaskDependency",
    "TaskDependencyDict",
    "TaskDependencyParam",
    "TaskDict",
    "TaskEmailNotifications",
    "TaskEmailNotificationsDict",
    "TaskEmailNotificationsParam",
    "TaskNotificationSettings",
    "TaskNotificationSettingsDict",
    "TaskNotificationSettingsParam",
    "TaskParam",
    "TriggerSettings",
    "TriggerSettingsDict",
    "TriggerSettingsParam",
    "VolumesStorageInfo",
    "VolumesStorageInfoDict",
    "VolumesStorageInfoParam",
    "Webhook",
    "WebhookDict",
    "WebhookNotifications",
    "WebhookNotificationsDict",
    "WebhookNotificationsParam",
    "WebhookParam",
    "WorkloadType",
    "WorkloadTypeDict",
    "WorkloadTypeParam",
    "WorkspaceStorageInfo",
    "WorkspaceStorageInfoDict",
    "WorkspaceStorageInfoParam",
]


from databricks.bundles.compute._models.adlsgen2_info import (
    Adlsgen2Info,
    Adlsgen2InfoDict,
    Adlsgen2InfoParam,
)
from databricks.bundles.compute._models.auto_scale import (
    AutoScale,
    AutoScaleDict,
    AutoScaleParam,
)
from databricks.bundles.compute._models.aws_attributes import (
    AwsAttributes,
    AwsAttributesDict,
    AwsAttributesParam,
)
from databricks.bundles.compute._models.aws_availability import (
    AwsAvailability,
    AwsAvailabilityParam,
)
from databricks.bundles.compute._models.azure_attributes import (
    AzureAttributes,
    AzureAttributesDict,
    AzureAttributesParam,
)
from databricks.bundles.compute._models.azure_availability import (
    AzureAvailability,
    AzureAvailabilityParam,
)
from databricks.bundles.compute._models.clients_types import (
    ClientsTypes,
    ClientsTypesDict,
    ClientsTypesParam,
)
from databricks.bundles.compute._models.cluster_log_conf import (
    ClusterLogConf,
    ClusterLogConfDict,
    ClusterLogConfParam,
)
from databricks.bundles.compute._models.cluster_spec import (
    ClusterSpec,
    ClusterSpecDict,
    ClusterSpecParam,
)
from databricks.bundles.compute._models.data_security_mode import (
    DataSecurityMode,
    DataSecurityModeParam,
)
from databricks.bundles.compute._models.dbfs_storage_info import (
    DbfsStorageInfo,
    DbfsStorageInfoDict,
    DbfsStorageInfoParam,
)
from databricks.bundles.compute._models.docker_basic_auth import (
    DockerBasicAuth,
    DockerBasicAuthDict,
    DockerBasicAuthParam,
)
from databricks.bundles.compute._models.docker_image import (
    DockerImage,
    DockerImageDict,
    DockerImageParam,
)
from databricks.bundles.compute._models.ebs_volume_type import (
    EbsVolumeType,
    EbsVolumeTypeParam,
)
from databricks.bundles.compute._models.environment import (
    Environment,
    EnvironmentDict,
    EnvironmentParam,
)
from databricks.bundles.compute._models.gcp_attributes import (
    GcpAttributes,
    GcpAttributesDict,
    GcpAttributesParam,
)
from databricks.bundles.compute._models.gcp_availability import (
    GcpAvailability,
    GcpAvailabilityParam,
)
from databricks.bundles.compute._models.gcs_storage_info import (
    GcsStorageInfo,
    GcsStorageInfoDict,
    GcsStorageInfoParam,
)
from databricks.bundles.compute._models.init_script_info import (
    InitScriptInfo,
    InitScriptInfoDict,
    InitScriptInfoParam,
)
from databricks.bundles.compute._models.library import (
    Library,
    LibraryDict,
    LibraryParam,
)
from databricks.bundles.compute._models.local_file_info import (
    LocalFileInfo,
    LocalFileInfoDict,
    LocalFileInfoParam,
)
from databricks.bundles.compute._models.log_analytics_info import (
    LogAnalyticsInfo,
    LogAnalyticsInfoDict,
    LogAnalyticsInfoParam,
)
from databricks.bundles.compute._models.maven_library import (
    MavenLibrary,
    MavenLibraryDict,
    MavenLibraryParam,
)
from databricks.bundles.compute._models.python_py_pi_library import (
    PythonPyPiLibrary,
    PythonPyPiLibraryDict,
    PythonPyPiLibraryParam,
)
from databricks.bundles.compute._models.r_cran_library import (
    RCranLibrary,
    RCranLibraryDict,
    RCranLibraryParam,
)
from databricks.bundles.compute._models.runtime_engine import (
    RuntimeEngine,
    RuntimeEngineParam,
)
from databricks.bundles.compute._models.s3_storage_info import (
    S3StorageInfo,
    S3StorageInfoDict,
    S3StorageInfoParam,
)
from databricks.bundles.compute._models.volumes_storage_info import (
    VolumesStorageInfo,
    VolumesStorageInfoDict,
    VolumesStorageInfoParam,
)
from databricks.bundles.compute._models.workload_type import (
    WorkloadType,
    WorkloadTypeDict,
    WorkloadTypeParam,
)
from databricks.bundles.compute._models.workspace_storage_info import (
    WorkspaceStorageInfo,
    WorkspaceStorageInfoDict,
    WorkspaceStorageInfoParam,
)
from databricks.bundles.jobs._models.authentication_method import (
    AuthenticationMethod,
    AuthenticationMethodParam,
)
from databricks.bundles.jobs._models.clean_rooms_notebook_task import (
    CleanRoomsNotebookTask,
    CleanRoomsNotebookTaskDict,
    CleanRoomsNotebookTaskParam,
)
from databricks.bundles.jobs._models.compute_config import (
    ComputeConfig,
    ComputeConfigDict,
    ComputeConfigParam,
)
from databricks.bundles.jobs._models.condition import Condition, ConditionParam
from databricks.bundles.jobs._models.condition_task import (
    ConditionTask,
    ConditionTaskDict,
    ConditionTaskParam,
)
from databricks.bundles.jobs._models.condition_task_op import (
    ConditionTaskOp,
    ConditionTaskOpParam,
)
from databricks.bundles.jobs._models.continuous import (
    Continuous,
    ContinuousDict,
    ContinuousParam,
)
from databricks.bundles.jobs._models.cron_schedule import (
    CronSchedule,
    CronScheduleDict,
    CronScheduleParam,
)
from databricks.bundles.jobs._models.dashboard_task import (
    DashboardTask,
    DashboardTaskDict,
    DashboardTaskParam,
)
from databricks.bundles.jobs._models.dbt_task import DbtTask, DbtTaskDict, DbtTaskParam
from databricks.bundles.jobs._models.file_arrival_trigger_configuration import (
    FileArrivalTriggerConfiguration,
    FileArrivalTriggerConfigurationDict,
    FileArrivalTriggerConfigurationParam,
)
from databricks.bundles.jobs._models.for_each_task import (
    ForEachTask,
    ForEachTaskDict,
    ForEachTaskParam,
)
from databricks.bundles.jobs._models.gen_ai_compute_task import (
    GenAiComputeTask,
    GenAiComputeTaskDict,
    GenAiComputeTaskParam,
)
from databricks.bundles.jobs._models.git_provider import GitProvider, GitProviderParam
from databricks.bundles.jobs._models.git_source import (
    GitSource,
    GitSourceDict,
    GitSourceParam,
)
from databricks.bundles.jobs._models.job import Job, JobDict, JobParam
from databricks.bundles.jobs._models.job_cluster import (
    JobCluster,
    JobClusterDict,
    JobClusterParam,
)
from databricks.bundles.jobs._models.job_email_notifications import (
    JobEmailNotifications,
    JobEmailNotificationsDict,
    JobEmailNotificationsParam,
)
from databricks.bundles.jobs._models.job_environment import (
    JobEnvironment,
    JobEnvironmentDict,
    JobEnvironmentParam,
)
from databricks.bundles.jobs._models.job_notification_settings import (
    JobNotificationSettings,
    JobNotificationSettingsDict,
    JobNotificationSettingsParam,
)
from databricks.bundles.jobs._models.job_parameter_definition import (
    JobParameterDefinition,
    JobParameterDefinitionDict,
    JobParameterDefinitionParam,
)
from databricks.bundles.jobs._models.job_permission import (
    JobPermission,
    JobPermissionDict,
    JobPermissionParam,
)
from databricks.bundles.jobs._models.job_permission_level import (
    JobPermissionLevel,
    JobPermissionLevelParam,
)
from databricks.bundles.jobs._models.job_run_as import (
    JobRunAs,
    JobRunAsDict,
    JobRunAsParam,
)
from databricks.bundles.jobs._models.jobs_health_metric import (
    JobsHealthMetric,
    JobsHealthMetricParam,
)
from databricks.bundles.jobs._models.jobs_health_operator import (
    JobsHealthOperator,
    JobsHealthOperatorParam,
)
from databricks.bundles.jobs._models.jobs_health_rule import (
    JobsHealthRule,
    JobsHealthRuleDict,
    JobsHealthRuleParam,
)
from databricks.bundles.jobs._models.jobs_health_rules import (
    JobsHealthRules,
    JobsHealthRulesDict,
    JobsHealthRulesParam,
)
from databricks.bundles.jobs._models.notebook_task import (
    NotebookTask,
    NotebookTaskDict,
    NotebookTaskParam,
)
from databricks.bundles.jobs._models.pause_status import PauseStatus, PauseStatusParam
from databricks.bundles.jobs._models.performance_target import (
    PerformanceTarget,
    PerformanceTargetParam,
)
from databricks.bundles.jobs._models.periodic_trigger_configuration import (
    PeriodicTriggerConfiguration,
    PeriodicTriggerConfigurationDict,
    PeriodicTriggerConfigurationParam,
)
from databricks.bundles.jobs._models.periodic_trigger_configuration_time_unit import (
    PeriodicTriggerConfigurationTimeUnit,
    PeriodicTriggerConfigurationTimeUnitParam,
)
from databricks.bundles.jobs._models.pipeline_params import (
    PipelineParams,
    PipelineParamsDict,
    PipelineParamsParam,
)
from databricks.bundles.jobs._models.pipeline_task import (
    PipelineTask,
    PipelineTaskDict,
    PipelineTaskParam,
)
from databricks.bundles.jobs._models.power_bi_model import (
    PowerBiModel,
    PowerBiModelDict,
    PowerBiModelParam,
)
from databricks.bundles.jobs._models.power_bi_table import (
    PowerBiTable,
    PowerBiTableDict,
    PowerBiTableParam,
)
from databricks.bundles.jobs._models.power_bi_task import (
    PowerBiTask,
    PowerBiTaskDict,
    PowerBiTaskParam,
)
from databricks.bundles.jobs._models.python_wheel_task import (
    PythonWheelTask,
    PythonWheelTaskDict,
    PythonWheelTaskParam,
)
from databricks.bundles.jobs._models.queue_settings import (
    QueueSettings,
    QueueSettingsDict,
    QueueSettingsParam,
)
from databricks.bundles.jobs._models.run_if import RunIf, RunIfParam
from databricks.bundles.jobs._models.run_job_task import (
    RunJobTask,
    RunJobTaskDict,
    RunJobTaskParam,
)
from databricks.bundles.jobs._models.source import Source, SourceParam
from databricks.bundles.jobs._models.spark_jar_task import (
    SparkJarTask,
    SparkJarTaskDict,
    SparkJarTaskParam,
)
from databricks.bundles.jobs._models.spark_python_task import (
    SparkPythonTask,
    SparkPythonTaskDict,
    SparkPythonTaskParam,
)
from databricks.bundles.jobs._models.spark_submit_task import (
    SparkSubmitTask,
    SparkSubmitTaskDict,
    SparkSubmitTaskParam,
)
from databricks.bundles.jobs._models.sql_task import SqlTask, SqlTaskDict, SqlTaskParam
from databricks.bundles.jobs._models.sql_task_alert import (
    SqlTaskAlert,
    SqlTaskAlertDict,
    SqlTaskAlertParam,
)
from databricks.bundles.jobs._models.sql_task_dashboard import (
    SqlTaskDashboard,
    SqlTaskDashboardDict,
    SqlTaskDashboardParam,
)
from databricks.bundles.jobs._models.sql_task_file import (
    SqlTaskFile,
    SqlTaskFileDict,
    SqlTaskFileParam,
)
from databricks.bundles.jobs._models.sql_task_query import (
    SqlTaskQuery,
    SqlTaskQueryDict,
    SqlTaskQueryParam,
)
from databricks.bundles.jobs._models.sql_task_subscription import (
    SqlTaskSubscription,
    SqlTaskSubscriptionDict,
    SqlTaskSubscriptionParam,
)
from databricks.bundles.jobs._models.storage_mode import StorageMode, StorageModeParam
from databricks.bundles.jobs._models.subscription import (
    Subscription,
    SubscriptionDict,
    SubscriptionParam,
)
from databricks.bundles.jobs._models.subscription_subscriber import (
    SubscriptionSubscriber,
    SubscriptionSubscriberDict,
    SubscriptionSubscriberParam,
)
from databricks.bundles.jobs._models.table_update_trigger_configuration import (
    TableUpdateTriggerConfiguration,
    TableUpdateTriggerConfigurationDict,
    TableUpdateTriggerConfigurationParam,
)
from databricks.bundles.jobs._models.task import Task, TaskDict, TaskParam
from databricks.bundles.jobs._models.task_dependency import (
    TaskDependency,
    TaskDependencyDict,
    TaskDependencyParam,
)
from databricks.bundles.jobs._models.task_email_notifications import (
    TaskEmailNotifications,
    TaskEmailNotificationsDict,
    TaskEmailNotificationsParam,
)
from databricks.bundles.jobs._models.task_notification_settings import (
    TaskNotificationSettings,
    TaskNotificationSettingsDict,
    TaskNotificationSettingsParam,
)
from databricks.bundles.jobs._models.trigger_settings import (
    TriggerSettings,
    TriggerSettingsDict,
    TriggerSettingsParam,
)
from databricks.bundles.jobs._models.webhook import Webhook, WebhookDict, WebhookParam
from databricks.bundles.jobs._models.webhook_notifications import (
    WebhookNotifications,
    WebhookNotificationsDict,
    WebhookNotificationsParam,
)


def _resolve_recursive_imports():
    import typing

    from databricks.bundles.core._variable import VariableOr
    from databricks.bundles.jobs._models.task import Task

    ForEachTask.__annotations__ = typing.get_type_hints(
        ForEachTask,
        globalns={"Task": Task, "VariableOr": VariableOr},
    )


_resolve_recursive_imports()
