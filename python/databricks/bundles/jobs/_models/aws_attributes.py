from typing import Literal, Optional, TypedDict, ClassVar, TYPE_CHECKING
from enum import Enum
from dataclasses import dataclass, replace, field

from databricks.bundles.core._resource import Resource
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional, VariableOrList, VariableOrDict

from databricks.bundles.jobs._models.adlsgen2_info import Adlsgen2Info, Adlsgen2InfoDict, Adlsgen2InfoParam
from databricks.bundles.jobs._models.auto_scale import AutoScale, AutoScaleDict, AutoScaleParam
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
class AwsAttributes:
    """
    Attributes set during cluster creation which are related to Amazon Web Services.
    """
    availability: VariableOrOptional[AwsAvailability] = None
    """
    Availability type used for all subsequent nodes past the `first_on_demand` ones.
    
    Note: If `first_on_demand` is zero, this availability type will be used for the entire cluster.
    """

    ebs_volume_count: VariableOrOptional[int] = None
    """
    The number of volumes launched for each instance. Users can choose up to 10 volumes.
    This feature is only enabled for supported node types. Legacy node types cannot specify
    custom EBS volumes.
    For node types with no instance store, at least one EBS volume needs to be specified;
    otherwise, cluster creation will fail.
    
    These EBS volumes will be mounted at `/ebs0`, `/ebs1`, and etc.
    Instance store volumes will be mounted at `/local_disk0`, `/local_disk1`, and etc.
    
    If EBS volumes are attached, Databricks will configure Spark to use only the EBS volumes for
    scratch storage because heterogenously sized scratch devices can lead to inefficient disk
    utilization. If no EBS volumes are attached, Databricks will configure Spark to use instance
    store volumes.
    
    Please note that if EBS volumes are specified, then the Spark configuration `spark.local.dir`
    will be overridden.
    """

    ebs_volume_iops: VariableOrOptional[int] = None
    """
    If using gp3 volumes, what IOPS to use for the disk. If this is not set, the maximum performance of a gp2 volume with the same volume size will be used.
    """

    ebs_volume_size: VariableOrOptional[int] = None
    """
    The size of each EBS volume (in GiB) launched for each instance. For general purpose
    SSD, this value must be within the range 100 - 4096. For throughput optimized HDD,
    this value must be within the range 500 - 4096.
    """

    ebs_volume_throughput: VariableOrOptional[int] = None
    """
    If using gp3 volumes, what throughput to use for the disk. If this is not set, the maximum performance of a gp2 volume with the same volume size will be used.
    """

    ebs_volume_type: VariableOrOptional[EbsVolumeType] = None
    """
    The type of EBS volumes that will be launched with this cluster.
    """

    first_on_demand: VariableOrOptional[int] = None
    """
    The first `first_on_demand` nodes of the cluster will be placed on on-demand instances.
    If this value is greater than 0, the cluster driver node in particular will be placed on an
    on-demand instance. If this value is greater than or equal to the current cluster size, all
    nodes will be placed on on-demand instances. If this value is less than the current cluster
    size, `first_on_demand` nodes will be placed on on-demand instances and the remainder will
    be placed on `availability` instances. Note that this value does not affect
    cluster size and cannot currently be mutated over the lifetime of a cluster.
    """

    instance_profile_arn: VariableOrOptional[str] = None
    """
    Nodes for this cluster will only be placed on AWS instances with this instance profile. If
    ommitted, nodes will be placed on instances without an IAM instance profile. The instance
    profile must have previously been added to the Databricks environment by an account
    administrator.
    
    This feature may only be available to certain customer plans.
    """

    spot_bid_price_percent: VariableOrOptional[int] = None
    """
    The bid price for AWS spot instances, as a percentage of the corresponding instance type's
    on-demand price.
    For example, if this field is set to 50, and the cluster needs a new `r3.xlarge` spot
    instance, then the bid price is half of the price of
    on-demand `r3.xlarge` instances. Similarly, if this field is set to 200, the bid price is twice
    the price of on-demand `r3.xlarge` instances. If not specified, the default value is 100.
    When spot instances are requested for this cluster, only spot instances whose bid price
    percentage matches this field will be considered.
    Note that, for safety, we enforce this field to be no more than 10000.
    """

    zone_id: VariableOrOptional[str] = None
    """
    Identifier for the availability zone/datacenter in which the cluster resides.
    This string will be of a form like "us-west-2a". The provided availability
    zone must be in the same region as the Databricks deployment. For example, "us-west-2a"
    is not a valid zone id if the Databricks deployment resides in the "us-east-1" region.
    This is an optional field at cluster creation, and if not specified, the zone "auto" will be used.
    If the zone specified is "auto", will try to place cluster in a zone with high availability,
    and will retry placement in a different AZ if there is not enough capacity.
    
    The list of available zones as well as the default value can be found by using the
    `List Zones` method.
    """

    @classmethod
    def from_dict(cls, value: 'AwsAttributesDict') -> 'Self':
        return _transform(cls, value)

    def as_dict(self) -> 'AwsAttributesDict':
        return _transform_to_json_value(self) # type:ignore



class AwsAttributesDict(TypedDict, total=False):
    """"""

    availability: VariableOrOptional[AwsAvailabilityParam]
    """
    Availability type used for all subsequent nodes past the `first_on_demand` ones.
    
    Note: If `first_on_demand` is zero, this availability type will be used for the entire cluster.
    """

    ebs_volume_count: VariableOrOptional[int]
    """
    The number of volumes launched for each instance. Users can choose up to 10 volumes.
    This feature is only enabled for supported node types. Legacy node types cannot specify
    custom EBS volumes.
    For node types with no instance store, at least one EBS volume needs to be specified;
    otherwise, cluster creation will fail.
    
    These EBS volumes will be mounted at `/ebs0`, `/ebs1`, and etc.
    Instance store volumes will be mounted at `/local_disk0`, `/local_disk1`, and etc.
    
    If EBS volumes are attached, Databricks will configure Spark to use only the EBS volumes for
    scratch storage because heterogenously sized scratch devices can lead to inefficient disk
    utilization. If no EBS volumes are attached, Databricks will configure Spark to use instance
    store volumes.
    
    Please note that if EBS volumes are specified, then the Spark configuration `spark.local.dir`
    will be overridden.
    """

    ebs_volume_iops: VariableOrOptional[int]
    """
    If using gp3 volumes, what IOPS to use for the disk. If this is not set, the maximum performance of a gp2 volume with the same volume size will be used.
    """

    ebs_volume_size: VariableOrOptional[int]
    """
    The size of each EBS volume (in GiB) launched for each instance. For general purpose
    SSD, this value must be within the range 100 - 4096. For throughput optimized HDD,
    this value must be within the range 500 - 4096.
    """

    ebs_volume_throughput: VariableOrOptional[int]
    """
    If using gp3 volumes, what throughput to use for the disk. If this is not set, the maximum performance of a gp2 volume with the same volume size will be used.
    """

    ebs_volume_type: VariableOrOptional[EbsVolumeTypeParam]
    """
    The type of EBS volumes that will be launched with this cluster.
    """

    first_on_demand: VariableOrOptional[int]
    """
    The first `first_on_demand` nodes of the cluster will be placed on on-demand instances.
    If this value is greater than 0, the cluster driver node in particular will be placed on an
    on-demand instance. If this value is greater than or equal to the current cluster size, all
    nodes will be placed on on-demand instances. If this value is less than the current cluster
    size, `first_on_demand` nodes will be placed on on-demand instances and the remainder will
    be placed on `availability` instances. Note that this value does not affect
    cluster size and cannot currently be mutated over the lifetime of a cluster.
    """

    instance_profile_arn: VariableOrOptional[str]
    """
    Nodes for this cluster will only be placed on AWS instances with this instance profile. If
    ommitted, nodes will be placed on instances without an IAM instance profile. The instance
    profile must have previously been added to the Databricks environment by an account
    administrator.
    
    This feature may only be available to certain customer plans.
    """

    spot_bid_price_percent: VariableOrOptional[int]
    """
    The bid price for AWS spot instances, as a percentage of the corresponding instance type's
    on-demand price.
    For example, if this field is set to 50, and the cluster needs a new `r3.xlarge` spot
    instance, then the bid price is half of the price of
    on-demand `r3.xlarge` instances. Similarly, if this field is set to 200, the bid price is twice
    the price of on-demand `r3.xlarge` instances. If not specified, the default value is 100.
    When spot instances are requested for this cluster, only spot instances whose bid price
    percentage matches this field will be considered.
    Note that, for safety, we enforce this field to be no more than 10000.
    """

    zone_id: VariableOrOptional[str]
    """
    Identifier for the availability zone/datacenter in which the cluster resides.
    This string will be of a form like "us-west-2a". The provided availability
    zone must be in the same region as the Databricks deployment. For example, "us-west-2a"
    is not a valid zone id if the Databricks deployment resides in the "us-east-1" region.
    This is an optional field at cluster creation, and if not specified, the zone "auto" will be used.
    If the zone specified is "auto", will try to place cluster in a zone with high availability,
    and will retry placement in a different AZ if there is not enough capacity.
    
    The list of available zones as well as the default value can be found by using the
    `List Zones` method.
    """


AwsAttributesParam = AwsAttributesDict | AwsAttributes
