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
from databricks.bundles.jobs._models.continuous import (
    Continuous,
    ContinuousParam,
)
from databricks.bundles.jobs._models.cron_schedule import (
    CronSchedule,
    CronScheduleParam,
)
from databricks.bundles.jobs._models.git_source import (
    GitSource,
    GitSourceParam,
)
from databricks.bundles.jobs._models.job_cluster import JobCluster, JobClusterParam
from databricks.bundles.jobs._models.job_email_notifications import (
    JobEmailNotifications,
    JobEmailNotificationsParam,
)
from databricks.bundles.jobs._models.job_environment import (
    JobEnvironment,
    JobEnvironmentParam,
)
from databricks.bundles.jobs._models.job_notification_settings import (
    JobNotificationSettings,
    JobNotificationSettingsParam,
)
from databricks.bundles.jobs._models.job_parameter_definition import (
    JobParameterDefinition,
    JobParameterDefinitionParam,
)
from databricks.bundles.jobs._models.job_permission import (
    JobPermission,
    JobPermissionParam,
)
from databricks.bundles.jobs._models.job_run_as import JobRunAs, JobRunAsParam
from databricks.bundles.jobs._models.jobs_health_rules import (
    JobsHealthRules,
    JobsHealthRulesParam,
)
from databricks.bundles.jobs._models.performance_target import (
    PerformanceTarget,
    PerformanceTargetParam,
)
from databricks.bundles.jobs._models.queue_settings import (
    QueueSettings,
    QueueSettingsParam,
)
from databricks.bundles.jobs._models.task import Task, TaskParam
from databricks.bundles.jobs._models.trigger_settings import (
    TriggerSettings,
    TriggerSettingsParam,
)
from databricks.bundles.jobs._models.webhook_notifications import (
    WebhookNotifications,
    WebhookNotificationsParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class Job(Resource):
    """"""

    budget_policy_id: VariableOrOptional[str] = None
    """
    The id of the user specified budget policy to use for this job.
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
    An environment is required to be present for serverless tasks.
    For serverless notebook tasks, the environment is accessible in the notebook environment panel.
    For other serverless tasks, the task environment is required to be specified using environment_key in the task settings.
    """

    git_source: VariableOrOptional[GitSource] = None
    """
    An optional specification for a remote Git repository containing the source code used by tasks. Version-controlled source code is supported by notebook, dbt, Python script, and SQL File tasks.
    
    If `git_source` is set, these tasks retrieve the file from the remote repository by default. However, this behavior can be overridden by setting `source` to `WORKSPACE` on the task.
    
    Note: dbt and SQL File tasks support only version-controlled sources. If dbt or SQL File tasks are used, `git_source` must be defined on the job.
    """

    health: VariableOrOptional[JobsHealthRules] = None

    job_clusters: VariableOrList[JobCluster] = field(default_factory=list)
    """
    A list of job cluster specifications that can be shared and reused by tasks of this job. Libraries cannot be declared in a shared job cluster. You must declare dependent libraries in task settings.
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

    performance_target: VariableOrOptional[PerformanceTarget] = None
    """
    The performance mode on a serverless job. This field determines the level of compute performance or cost-efficiency for the run.
    
    * `STANDARD`: Enables cost-efficient execution of serverless workloads.
    * `PERFORMANCE_OPTIMIZED`: Prioritizes fast startup and execution times through rapid scaling and optimized cluster performance.
    """

    permissions: VariableOrList[JobPermission] = field(default_factory=list)

    queue: VariableOrOptional[QueueSettings] = None
    """
    The queue settings of the job.
    """

    run_as: VariableOrOptional[JobRunAs] = None

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

    webhook_notifications: VariableOrOptional[WebhookNotifications] = None
    """
    A collection of system notification IDs to notify when runs of this job begin or complete.
    """

    @classmethod
    def from_dict(cls, value: "JobDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "JobDict":
        return _transform_to_json_value(self)  # type:ignore


class JobDict(TypedDict, total=False):
    """"""

    budget_policy_id: VariableOrOptional[str]
    """
    The id of the user specified budget policy to use for this job.
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
    An environment is required to be present for serverless tasks.
    For serverless notebook tasks, the environment is accessible in the notebook environment panel.
    For other serverless tasks, the task environment is required to be specified using environment_key in the task settings.
    """

    git_source: VariableOrOptional[GitSourceParam]
    """
    An optional specification for a remote Git repository containing the source code used by tasks. Version-controlled source code is supported by notebook, dbt, Python script, and SQL File tasks.
    
    If `git_source` is set, these tasks retrieve the file from the remote repository by default. However, this behavior can be overridden by setting `source` to `WORKSPACE` on the task.
    
    Note: dbt and SQL File tasks support only version-controlled sources. If dbt or SQL File tasks are used, `git_source` must be defined on the job.
    """

    health: VariableOrOptional[JobsHealthRulesParam]

    job_clusters: VariableOrList[JobClusterParam]
    """
    A list of job cluster specifications that can be shared and reused by tasks of this job. Libraries cannot be declared in a shared job cluster. You must declare dependent libraries in task settings.
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

    performance_target: VariableOrOptional[PerformanceTargetParam]
    """
    The performance mode on a serverless job. This field determines the level of compute performance or cost-efficiency for the run.
    
    * `STANDARD`: Enables cost-efficient execution of serverless workloads.
    * `PERFORMANCE_OPTIMIZED`: Prioritizes fast startup and execution times through rapid scaling and optimized cluster performance.
    """

    permissions: VariableOrList[JobPermissionParam]

    queue: VariableOrOptional[QueueSettingsParam]
    """
    The queue settings of the job.
    """

    run_as: VariableOrOptional[JobRunAsParam]

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

    webhook_notifications: VariableOrOptional[WebhookNotificationsParam]
    """
    A collection of system notification IDs to notify when runs of this job begin or complete.
    """


JobParam = JobDict | Job
