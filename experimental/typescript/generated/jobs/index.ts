/**
 * Job resource types for Databricks Asset Bundles
 *
 * These are placeholder types that will be replaced by auto-generated types
 * from OpenAPI specifications in the future.
 */

import type { Resource } from "../../src/core/resource.js";
import type { VariableOr } from "../../src/core/variable.js";

/**
 * Represents a Databricks Job resource
 */
export interface Job extends Resource {
  /**
   * Optional user-defined name for the job
   */
  name?: VariableOr<string>;

  /**
   * List of tasks that make up the job
   */
  tasks?: VariableOr<Task[]>;

  /**
   * Optional job clusters used by tasks
   */
  job_clusters?: VariableOr<JobCluster[]>;

  /**
   * Optional schedule configuration
   */
  schedule?: VariableOr<CronSchedule>;

  /**
   * Maximum number of concurrent runs
   */
  max_concurrent_runs?: VariableOr<number>;

  /**
   * Timeout in seconds
   */
  timeout_seconds?: VariableOr<number>;

  /**
   * Email notifications configuration
   */
  email_notifications?: VariableOr<JobEmailNotifications>;

  /**
   * Webhook notifications
   */
  webhook_notifications?: VariableOr<WebhookNotifications>;

  /**
   * Tags for the job
   */
  tags?: VariableOr<Record<string, string>>;

  /**
   * Job parameters
   */
  parameters?: VariableOr<JobParameterDefinition[]>;
}

/**
 * Task within a job
 */
export interface Task {
  /**
   * Unique task key
   */
  task_key: VariableOr<string>;

  /**
   * Task dependencies
   */
  depends_on?: VariableOr<TaskDependency[]>;

  /**
   * Notebook task configuration
   */
  notebook_task?: VariableOr<NotebookTask>;

  /**
   * Spark Python task configuration
   */
  spark_python_task?: VariableOr<SparkPythonTask>;

  /**
   * Python wheel task configuration
   */
  python_wheel_task?: VariableOr<PythonWheelTask>;

  /**
   * SQL task configuration
   */
  sql_task?: VariableOr<SqlTask>;

  /**
   * Pipeline task configuration
   */
  pipeline_task?: VariableOr<PipelineTask>;

  /**
   * DBT task configuration
   */
  dbt_task?: VariableOr<DbtTask>;

  /**
   * Libraries to install
   */
  libraries?: VariableOr<Library[]>;

  /**
   * Existing cluster ID
   */
  existing_cluster_id?: VariableOr<string>;

  /**
   * New cluster configuration
   */
  new_cluster?: VariableOr<ClusterSpec>;

  /**
   * Job cluster key
   */
  job_cluster_key?: VariableOr<string>;

  /**
   * Timeout in seconds
   */
  timeout_seconds?: VariableOr<number>;

  /**
   * Maximum number of retries
   */
  max_retries?: VariableOr<number>;

  /**
   * Minimum retry interval in milliseconds
   */
  min_retry_interval_millis?: VariableOr<number>;
}

/**
 * Task dependency
 */
export interface TaskDependency {
  task_key: VariableOr<string>;
}

/**
 * Notebook task configuration
 */
export interface NotebookTask {
  notebook_path: VariableOr<string>;
  base_parameters?: VariableOr<Record<string, string>>;
  source?: VariableOr<string>;
}

/**
 * Spark Python task configuration
 */
export interface SparkPythonTask {
  python_file: VariableOr<string>;
  parameters?: VariableOr<string[]>;
  source?: VariableOr<string>;
}

/**
 * Python wheel task configuration
 */
export interface PythonWheelTask {
  package_name: VariableOr<string>;
  entry_point: VariableOr<string>;
  parameters?: VariableOr<string[]>;
  named_parameters?: VariableOr<Record<string, string>>;
}

/**
 * SQL task configuration
 */
export interface SqlTask {
  warehouse_id: VariableOr<string>;
  query?: VariableOr<SqlTaskQuery>;
  dashboard?: VariableOr<SqlTaskDashboard>;
  alert?: VariableOr<SqlTaskAlert>;
  file?: VariableOr<SqlTaskFile>;
  parameters?: VariableOr<Record<string, string>>;
}

/**
 * SQL query configuration
 */
export interface SqlTaskQuery {
  query_id: VariableOr<string>;
}

/**
 * SQL dashboard configuration
 */
export interface SqlTaskDashboard {
  dashboard_id: VariableOr<string>;
  custom_subject?: VariableOr<string>;
  subscriptions?: VariableOr<SqlTaskSubscription[]>;
}

/**
 * SQL alert configuration
 */
export interface SqlTaskAlert {
  alert_id: VariableOr<string>;
  subscriptions?: VariableOr<SqlTaskSubscription[]>;
}

/**
 * SQL file configuration
 */
export interface SqlTaskFile {
  path: VariableOr<string>;
  source?: VariableOr<string>;
}

/**
 * SQL task subscription
 */
export interface SqlTaskSubscription {
  user_name?: VariableOr<string>;
  destination_id?: VariableOr<string>;
}

/**
 * Pipeline task configuration
 */
export interface PipelineTask {
  pipeline_id: VariableOr<string>;
  full_refresh?: VariableOr<boolean>;
}

/**
 * DBT task configuration
 */
export interface DbtTask {
  commands: VariableOr<string[]>;
  project_directory?: VariableOr<string>;
  profiles_directory?: VariableOr<string>;
  catalog?: VariableOr<string>;
  schema?: VariableOr<string>;
  warehouse_id?: VariableOr<string>;
}

/**
 * Library dependency
 */
export interface Library {
  jar?: VariableOr<string>;
  egg?: VariableOr<string>;
  whl?: VariableOr<string>;
  pypi?: VariableOr<PythonPyPiLibrary>;
  maven?: VariableOr<MavenLibrary>;
  cran?: VariableOr<RCranLibrary>;
}

/**
 * PyPI library
 */
export interface PythonPyPiLibrary {
  package: VariableOr<string>;
  repo?: VariableOr<string>;
}

/**
 * Maven library
 */
export interface MavenLibrary {
  coordinates: VariableOr<string>;
  repo?: VariableOr<string>;
  exclusions?: VariableOr<string[]>;
}

/**
 * CRAN library
 */
export interface RCranLibrary {
  package: VariableOr<string>;
  repo?: VariableOr<string>;
}

/**
 * Cluster specification
 */
export interface ClusterSpec {
  spark_version: VariableOr<string>;
  node_type_id?: VariableOr<string>;
  driver_node_type_id?: VariableOr<string>;
  num_workers?: VariableOr<number>;
  autoscale?: VariableOr<AutoScale>;
  spark_conf?: VariableOr<Record<string, string>>;
  spark_env_vars?: VariableOr<Record<string, string>>;
  custom_tags?: VariableOr<Record<string, string>>;
  init_scripts?: VariableOr<InitScriptInfo[]>;
  data_security_mode?: VariableOr<string>;
  runtime_engine?: VariableOr<string>;
}

/**
 * Autoscale configuration
 */
export interface AutoScale {
  min_workers: VariableOr<number>;
  max_workers: VariableOr<number>;
}

/**
 * Init script configuration
 */
export interface InitScriptInfo {
  dbfs?: VariableOr<DbfsStorageInfo>;
  workspace?: VariableOr<WorkspaceStorageInfo>;
  volumes?: VariableOr<VolumesStorageInfo>;
}

/**
 * DBFS storage info
 */
export interface DbfsStorageInfo {
  destination: VariableOr<string>;
}

/**
 * Workspace storage info
 */
export interface WorkspaceStorageInfo {
  destination: VariableOr<string>;
}

/**
 * Volumes storage info
 */
export interface VolumesStorageInfo {
  destination: VariableOr<string>;
}

/**
 * Job cluster configuration
 */
export interface JobCluster {
  job_cluster_key: VariableOr<string>;
  new_cluster: VariableOr<ClusterSpec>;
}

/**
 * Cron schedule configuration
 */
export interface CronSchedule {
  quartz_cron_expression: VariableOr<string>;
  timezone_id: VariableOr<string>;
  pause_status?: VariableOr<string>;
}

/**
 * Email notifications
 */
export interface JobEmailNotifications {
  on_start?: VariableOr<string[]>;
  on_success?: VariableOr<string[]>;
  on_failure?: VariableOr<string[]>;
  no_alert_for_skipped_runs?: VariableOr<boolean>;
}

/**
 * Webhook notifications
 */
export interface WebhookNotifications {
  on_start?: VariableOr<Webhook[]>;
  on_success?: VariableOr<Webhook[]>;
  on_failure?: VariableOr<Webhook[]>;
}

/**
 * Webhook configuration
 */
export interface Webhook {
  id: VariableOr<string>;
}

/**
 * Job parameter definition
 */
export interface JobParameterDefinition {
  name: VariableOr<string>;
  default: VariableOr<string>;
}

/**
 * Helper function to create a Job with type safety
 */
export function createJob(config: Job): Job {
  return config;
}

/**
 * Helper function to create a Task with type safety
 */
export function createTask(config: Task): Task {
  return config;
}
