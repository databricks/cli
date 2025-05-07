from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.compute._models.cluster_spec import (
    ClusterSpec,
    ClusterSpecParam,
)
from databricks.bundles.compute._models.library import (
    Library,
    LibraryParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOr,
    VariableOrList,
    VariableOrOptional,
)
from databricks.bundles.jobs._models.clean_rooms_notebook_task import (
    CleanRoomsNotebookTask,
    CleanRoomsNotebookTaskParam,
)
from databricks.bundles.jobs._models.condition_task import (
    ConditionTask,
    ConditionTaskParam,
)
from databricks.bundles.jobs._models.dashboard_task import (
    DashboardTask,
    DashboardTaskParam,
)
from databricks.bundles.jobs._models.dbt_task import DbtTask, DbtTaskParam
from databricks.bundles.jobs._models.for_each_task import (
    ForEachTask,
    ForEachTaskParam,
)
from databricks.bundles.jobs._models.gen_ai_compute_task import (
    GenAiComputeTask,
    GenAiComputeTaskParam,
)
from databricks.bundles.jobs._models.jobs_health_rules import (
    JobsHealthRules,
    JobsHealthRulesParam,
)
from databricks.bundles.jobs._models.notebook_task import (
    NotebookTask,
    NotebookTaskParam,
)
from databricks.bundles.jobs._models.pipeline_task import (
    PipelineTask,
    PipelineTaskParam,
)
from databricks.bundles.jobs._models.power_bi_task import PowerBiTask, PowerBiTaskParam
from databricks.bundles.jobs._models.python_wheel_task import (
    PythonWheelTask,
    PythonWheelTaskParam,
)
from databricks.bundles.jobs._models.run_if import RunIf, RunIfParam
from databricks.bundles.jobs._models.run_job_task import (
    RunJobTask,
    RunJobTaskParam,
)
from databricks.bundles.jobs._models.spark_jar_task import (
    SparkJarTask,
    SparkJarTaskParam,
)
from databricks.bundles.jobs._models.spark_python_task import (
    SparkPythonTask,
    SparkPythonTaskParam,
)
from databricks.bundles.jobs._models.spark_submit_task import (
    SparkSubmitTask,
    SparkSubmitTaskParam,
)
from databricks.bundles.jobs._models.sql_task import SqlTask, SqlTaskParam
from databricks.bundles.jobs._models.task_dependency import (
    TaskDependency,
    TaskDependencyParam,
)
from databricks.bundles.jobs._models.task_email_notifications import (
    TaskEmailNotifications,
    TaskEmailNotificationsParam,
)
from databricks.bundles.jobs._models.task_notification_settings import (
    TaskNotificationSettings,
    TaskNotificationSettingsParam,
)
from databricks.bundles.jobs._models.webhook_notifications import (
    WebhookNotifications,
    WebhookNotificationsParam,
)

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

    clean_rooms_notebook_task: VariableOrOptional[CleanRoomsNotebookTask] = None
    """
    The task runs a [clean rooms](https://docs.databricks.com/en/clean-rooms/index.html) notebook
    when the `clean_rooms_notebook_task` field is present.
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
    """

    health: VariableOrOptional[JobsHealthRules] = None

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
    :meta private: [EXPERIMENTAL]
    
    The task triggers a Power BI semantic model update when the `power_bi_task` field is present.
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
    (Legacy) The task runs the spark-submit script when the `spark_submit_task` field is present. This task can run only on new clusters and is not compatible with serverless compute.
    
    In the `new_cluster` specification, `libraries` and `spark_conf` are not supported. Instead, use `--jars` and `--py-files` to add Java and Python libraries and `--conf` to set the Spark configurations.
    
    `master`, `deploy-mode`, and `executor-cores` are automatically configured by Databricks; you _cannot_ specify them in parameters.
    
    By default, the Spark submit job uses all available memory (excluding reserved memory for Databricks services). You can set `--driver-memory`, and `--executor-memory` to a smaller value to leave some room for off-heap usage.
    
    The `--jars`, `--py-files`, `--files` arguments support DBFS and S3 paths.
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
    def from_dict(cls, value: "TaskDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "TaskDict":
        return _transform_to_json_value(self)  # type:ignore


class TaskDict(TypedDict, total=False):
    """"""

    task_key: VariableOr[str]
    """
    A unique name for the task. This field is used to refer to this task from other tasks.
    This field is required and must be unique within its parent job.
    On Update or Reset, this field is used to reference the tasks to be updated or reset.
    """

    clean_rooms_notebook_task: VariableOrOptional[CleanRoomsNotebookTaskParam]
    """
    The task runs a [clean rooms](https://docs.databricks.com/en/clean-rooms/index.html) notebook
    when the `clean_rooms_notebook_task` field is present.
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
    """

    health: VariableOrOptional[JobsHealthRulesParam]

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
    :meta private: [EXPERIMENTAL]
    
    The task triggers a Power BI semantic model update when the `power_bi_task` field is present.
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
    (Legacy) The task runs the spark-submit script when the `spark_submit_task` field is present. This task can run only on new clusters and is not compatible with serverless compute.
    
    In the `new_cluster` specification, `libraries` and `spark_conf` are not supported. Instead, use `--jars` and `--py-files` to add Java and Python libraries and `--conf` to set the Spark configurations.
    
    `master`, `deploy-mode`, and `executor-cores` are automatically configured by Databricks; you _cannot_ specify them in parameters.
    
    By default, the Spark submit job uses all available memory (excluding reserved memory for Databricks services). You can set `--driver-memory`, and `--executor-memory` to a smaller value to leave some room for off-heap usage.
    
    The `--jars`, `--py-files`, `--files` arguments support DBFS and S3 paths.
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
