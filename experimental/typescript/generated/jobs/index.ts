/**
 * Job resource types for Databricks Asset Bundles
 *
 * Auto-generated from JSON Schema. Do not edit manually.
 */

import { Resource } from "../../src/core/resource.js";
import type { VariableOr } from "../../src/core/variable.js";

export interface JobParams {
  /**
   * The id of the user specified budget policy to use for this job.
   * If not specified, a default budget policy may be applied when creating or modifying the job.
   * See `effective_budget_policy_id` for the budget policy used by this workload.
   */
  budget_policy_id?: VariableOr<string>;
  /**
   * An optional continuous property for this job. The continuous property will ensure that there is always one run executing. Only one of `schedule` and `continuous` can be used.
   */
  continuous?: VariableOr<Continuous>;
  /**
   * An optional description for the job. The maximum length is 27700 characters in UTF-8 encoding.
   */
  description?: VariableOr<string>;
  /**
   * An optional set of email addresses that is notified when runs of this job begin or complete as well as when this job is deleted.
   */
  email_notifications?: VariableOr<JobEmailNotifications>;
  /**
   * A list of task execution environment specifications that can be referenced by serverless tasks of this job.
   * An environment is required to be present for serverless tasks.
   * For serverless notebook tasks, the environment is accessible in the notebook environment panel.
   * For other serverless tasks, the task environment is required to be specified using environment_key in the task settings.
   */
  environments?: VariableOr<JobEnvironment[]>;
  /**
   * An optional specification for a remote Git repository containing the source code used by tasks. Version-controlled source code is supported by notebook, dbt, Python script, and SQL File tasks.
   * 
   * If `git_source` is set, these tasks retrieve the file from the remote repository by default. However, this behavior can be overridden by setting `source` to `WORKSPACE` on the task.
   * 
   * Note: dbt and SQL File tasks support only version-controlled sources. If dbt or SQL File tasks are used, `git_source` must be defined on the job.
   */
  git_source?: VariableOr<GitSource>;
  health?: VariableOr<JobsHealthRules>;
  /**
   * A list of job cluster specifications that can be shared and reused by tasks of this job. Libraries cannot be declared in a shared job cluster. You must declare dependent libraries in task settings.
   */
  job_clusters?: VariableOr<JobCluster[]>;
  /**
   * Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
   */
  lifecycle?: VariableOr<Lifecycle>;
  /**
   * An optional maximum allowed number of concurrent runs of the job.
   * Set this value if you want to be able to execute multiple runs of the same job concurrently.
   * This is useful for example if you trigger your job on a frequent schedule and want to allow consecutive runs to overlap with each other, or if you want to trigger multiple runs which differ by their input parameters.
   * This setting affects only new runs. For example, suppose the job’s concurrency is 4 and there are 4 concurrent active runs. Then setting the concurrency to 3 won’t kill any of the active runs.
   * However, from then on, new runs are skipped unless there are fewer than 3 active runs.
   * This value cannot exceed 1000. Setting this value to `0` causes all new runs to be skipped.
   */
  max_concurrent_runs?: VariableOr<number>;
  /**
   * An optional name for the job. The maximum length is 4096 bytes in UTF-8 encoding.
   */
  name?: VariableOr<string>;
  /**
   * Optional notification settings that are used when sending notifications to each of the `email_notifications` and `webhook_notifications` for this job.
   */
  notification_settings?: VariableOr<JobNotificationSettings>;
  /**
   * Job-level parameter definitions
   */
  parameters?: VariableOr<JobParameterDefinition[]>;
  /**
   * The performance mode on a serverless job. This field determines the level of compute performance or cost-efficiency for the run.
   * 
   * * `STANDARD`: Enables cost-efficient execution of serverless workloads.
   * * `PERFORMANCE_OPTIMIZED`: Prioritizes fast startup and execution times through rapid scaling and optimized cluster performance.
   */
  performance_target?: VariableOr<PerformanceTarget>;
  permissions?: VariableOr<JobPermission[]>;
  /**
   * The queue settings of the job.
   */
  queue?: VariableOr<QueueSettings>;
  run_as?: VariableOr<JobRunAs>;
  /**
   * An optional periodic schedule for this job. The default behavior is that the job only runs when triggered by clicking “Run Now” in the Jobs UI or sending an API request to `runNow`.
   */
  schedule?: VariableOr<CronSchedule>;
  /**
   * A map of tags associated with the job. These are forwarded to the cluster as cluster tags for jobs clusters, and are subject to the same limitations as cluster tags. A maximum of 25 tags can be added to the job.
   */
  tags?: VariableOr<Record<string, string>>;
  /**
   * A list of task specifications to be executed by this job.
   * It supports up to 1000 elements in write endpoints (:method:jobs/create, :method:jobs/reset, :method:jobs/update, :method:jobs/submit).
   * Read endpoints return only 100 tasks. If more than 100 tasks are available, you can paginate through them using :method:jobs/get. Use the `next_page_token` field at the object root to determine if more results are available.
   */
  tasks?: VariableOr<Task[]>;
  /**
   * An optional timeout applied to each run of this job. A value of `0` means no timeout.
   */
  timeout_seconds?: VariableOr<number>;
  /**
   * A configuration to trigger a run when certain conditions are met. The default behavior is that the job runs only when triggered by clicking “Run Now” in the Jobs UI or sending an API request to `runNow`.
   */
  trigger?: VariableOr<TriggerSettings>;
  /**
   * The id of the user specified usage policy to use for this job.
   * If not specified, a default usage policy may be applied when creating or modifying the job.
   * See `effective_usage_policy_id` for the usage policy used by this workload.
   */
  usage_policy_id?: VariableOr<string>;
  /**
   * A collection of system notification IDs to notify when runs of this job begin or complete.
   */
  webhook_notifications?: VariableOr<WebhookNotifications>;
}

export class Job extends Resource<JobParams> {
  constructor(params: JobParams) {
    super(params, "jobs");
  }
}

/**
 * A storage location in Adls Gen2
 */
export interface Adlsgen2Info {
  /**
   * abfss destination, e.g. `abfss://<container-name>@<storage-account-name>.dfs.core.windows.net/<directory-name>`.
   */
  destination: VariableOr<string>;
}

export interface AutoScale {
  /**
   * The maximum number of workers to which the cluster can scale up when overloaded.
   * Note that `max_workers` must be strictly greater than `min_workers`.
   */
  max_workers?: VariableOr<number>;
  /**
   * The minimum number of workers to which the cluster can scale down when underutilized.
   * It is also the initial number of workers the cluster will have after creation.
   */
  min_workers?: VariableOr<number>;
}

/**
 * Attributes set during cluster creation which are related to Amazon Web Services.
 */
export interface AwsAttributes {
  availability?: VariableOr<AwsAvailability>;
  /**
   * The number of volumes launched for each instance. Users can choose up to 10 volumes.
   * This feature is only enabled for supported node types. Legacy node types cannot specify
   * custom EBS volumes.
   * For node types with no instance store, at least one EBS volume needs to be specified;
   * otherwise, cluster creation will fail.
   * 
   * These EBS volumes will be mounted at `/ebs0`, `/ebs1`, and etc.
   * Instance store volumes will be mounted at `/local_disk0`, `/local_disk1`, and etc.
   * 
   * If EBS volumes are attached, Databricks will configure Spark to use only the EBS volumes for
   * scratch storage because heterogenously sized scratch devices can lead to inefficient disk
   * utilization. If no EBS volumes are attached, Databricks will configure Spark to use instance
   * store volumes.
   * 
   * Please note that if EBS volumes are specified, then the Spark configuration `spark.local.dir`
   * will be overridden.
   */
  ebs_volume_count?: VariableOr<number>;
  /**
   * If using gp3 volumes, what IOPS to use for the disk. If this is not set, the maximum performance of a gp2 volume with the same volume size will be used.
   */
  ebs_volume_iops?: VariableOr<number>;
  /**
   * The size of each EBS volume (in GiB) launched for each instance. For general purpose
   * SSD, this value must be within the range 100 - 4096. For throughput optimized HDD,
   * this value must be within the range 500 - 4096.
   */
  ebs_volume_size?: VariableOr<number>;
  /**
   * If using gp3 volumes, what throughput to use for the disk. If this is not set, the maximum performance of a gp2 volume with the same volume size will be used.
   */
  ebs_volume_throughput?: VariableOr<number>;
  ebs_volume_type?: VariableOr<EbsVolumeType>;
  /**
   * The first `first_on_demand` nodes of the cluster will be placed on on-demand instances.
   * If this value is greater than 0, the cluster driver node in particular will be placed on an
   * on-demand instance. If this value is greater than or equal to the current cluster size, all
   * nodes will be placed on on-demand instances. If this value is less than the current cluster
   * size, `first_on_demand` nodes will be placed on on-demand instances and the remainder will
   * be placed on `availability` instances. Note that this value does not affect
   * cluster size and cannot currently be mutated over the lifetime of a cluster.
   */
  first_on_demand?: VariableOr<number>;
  /**
   * Nodes for this cluster will only be placed on AWS instances with this instance profile. If
   * ommitted, nodes will be placed on instances without an IAM instance profile. The instance
   * profile must have previously been added to the Databricks environment by an account
   * administrator.
   * 
   * This feature may only be available to certain customer plans.
   */
  instance_profile_arn?: VariableOr<string>;
  /**
   * The bid price for AWS spot instances, as a percentage of the corresponding instance type's
   * on-demand price.
   * For example, if this field is set to 50, and the cluster needs a new `r3.xlarge` spot
   * instance, then the bid price is half of the price of
   * on-demand `r3.xlarge` instances. Similarly, if this field is set to 200, the bid price is twice
   * the price of on-demand `r3.xlarge` instances. If not specified, the default value is 100.
   * When spot instances are requested for this cluster, only spot instances whose bid price
   * percentage matches this field will be considered.
   * Note that, for safety, we enforce this field to be no more than 10000.
   */
  spot_bid_price_percent?: VariableOr<number>;
  /**
   * Identifier for the availability zone/datacenter in which the cluster resides.
   * This string will be of a form like "us-west-2a". The provided availability
   * zone must be in the same region as the Databricks deployment. For example, "us-west-2a"
   * is not a valid zone id if the Databricks deployment resides in the "us-east-1" region.
   * This is an optional field at cluster creation, and if not specified, a default zone will be used.
   * If the zone specified is "auto", will try to place cluster in a zone with high availability,
   * and will retry placement in a different AZ if there is not enough capacity.
   * 
   * The list of available zones as well as the default value can be found by using the
   * `List Zones` method.
   */
  zone_id?: VariableOr<string>;
}

/**
 * Availability type used for all subsequent nodes past the `first_on_demand` ones.
 * 
 * Note: If `first_on_demand` is zero, this availability type will be used for the entire cluster.
 */
export type AwsAvailability =
  | "SPOT"
  | "ON_DEMAND"
  | "SPOT_WITH_FALLBACK";

/**
 * Attributes set during cluster creation which are related to Microsoft Azure.
 */
export interface AzureAttributes {
  availability?: VariableOr<AzureAvailability>;
  /**
   * The first `first_on_demand` nodes of the cluster will be placed on on-demand instances.
   * This value should be greater than 0, to make sure the cluster driver node is placed on an
   * on-demand instance. If this value is greater than or equal to the current cluster size, all
   * nodes will be placed on on-demand instances. If this value is less than the current cluster
   * size, `first_on_demand` nodes will be placed on on-demand instances and the remainder will
   * be placed on `availability` instances. Note that this value does not affect
   * cluster size and cannot currently be mutated over the lifetime of a cluster.
   */
  first_on_demand?: VariableOr<number>;
  /**
   * Defines values necessary to configure and run Azure Log Analytics agent
   */
  log_analytics_info?: VariableOr<LogAnalyticsInfo>;
  /**
   * The max bid price to be used for Azure spot instances.
   * The Max price for the bid cannot be higher than the on-demand price of the instance.
   * If not specified, the default value is -1, which specifies that the instance cannot be evicted
   * on the basis of price, and only on the basis of availability. Further, the value should > 0 or -1.
   */
  spot_bid_max_price?: VariableOr<number>;
}

/**
 * Availability type used for all subsequent nodes past the `first_on_demand` ones.
 * Note: If `first_on_demand` is zero, this availability type will be used for the entire cluster.
 */
export type AzureAvailability =
  | "SPOT_AZURE"
  | "ON_DEMAND_AZURE"
  | "SPOT_WITH_FALLBACK_AZURE";

export interface ClientsTypes {
  /**
   * With jobs set, the cluster can be used for jobs
   */
  jobs?: VariableOr<boolean>;
  /**
   * With notebooks set, this cluster can be used for notebooks
   */
  notebooks?: VariableOr<boolean>;
}

/**
 * Cluster log delivery config
 */
export interface ClusterLogConf {
  /**
   * destination needs to be provided. e.g.
   * `{ "dbfs" : { "destination" : "dbfs:/home/cluster_log" } }`
   */
  dbfs?: VariableOr<DbfsStorageInfo>;
  /**
   * destination and either the region or endpoint need to be provided. e.g.
   * `{ "s3": { "destination" : "s3://cluster_log_bucket/prefix", "region" : "us-west-2" } }`
   * Cluster iam role is used to access s3, please make sure the cluster iam role in
   * `instance_profile_arn` has permission to write data to the s3 destination.
   */
  s3?: VariableOr<S3StorageInfo>;
  /**
   * destination needs to be provided, e.g.
   * `{ "volumes": { "destination": "/Volumes/catalog/schema/volume/cluster_log" } }`
   */
  volumes?: VariableOr<VolumesStorageInfo>;
}

/**
 * Contains a snapshot of the latest user specified settings that were used to create/edit the cluster.
 */
export interface ClusterSpec {
  /**
   * When set to true, fixed and default values from the policy will be used for fields that are omitted. When set to false, only fixed values from the policy will be applied.
   */
  apply_policy_default_values?: VariableOr<boolean>;
  /**
   * Parameters needed in order to automatically scale clusters up and down based on load.
   * Note: autoscaling works best with DB runtime versions 3.0 or later.
   */
  autoscale?: VariableOr<AutoScale>;
  /**
   * Automatically terminates the cluster after it is inactive for this time in minutes. If not set,
   * this cluster will not be automatically terminated. If specified, the threshold must be between
   * 10 and 10000 minutes.
   * Users can also set this value to 0 to explicitly disable automatic termination.
   */
  autotermination_minutes?: VariableOr<number>;
  /**
   * Attributes related to clusters running on Amazon Web Services.
   * If not specified at cluster creation, a set of default values will be used.
   */
  aws_attributes?: VariableOr<AwsAttributes>;
  /**
   * Attributes related to clusters running on Microsoft Azure.
   * If not specified at cluster creation, a set of default values will be used.
   */
  azure_attributes?: VariableOr<AzureAttributes>;
  /**
   * The configuration for delivering spark logs to a long-term storage destination.
   * Three kinds of destinations (DBFS, S3 and Unity Catalog volumes) are supported. Only one destination can be specified
   * for one cluster. If the conf is given, the logs will be delivered to the destination every
   * `5 mins`. The destination of driver logs is `$destination/$clusterId/driver`, while
   * the destination of executor logs is `$destination/$clusterId/executor`.
   */
  cluster_log_conf?: VariableOr<ClusterLogConf>;
  /**
   * Cluster name requested by the user. This doesn't have to be unique.
   * If not specified at creation, the cluster name will be an empty string.
   * For job clusters, the cluster name is automatically set based on the job and job run IDs.
   */
  cluster_name?: VariableOr<string>;
  /**
   * Additional tags for cluster resources. Databricks will tag all cluster resources (e.g., AWS
   * instances and EBS volumes) with these tags in addition to `default_tags`. Notes:
   * 
   * - Currently, Databricks allows at most 45 custom tags
   * 
   * - Clusters can only reuse cloud resources if the resources' tags are a subset of the cluster tags
   */
  custom_tags?: VariableOr<Record<string, string>>;
  data_security_mode?: VariableOr<DataSecurityMode>;
  docker_image?: VariableOr<DockerImage>;
  /**
   * The optional ID of the instance pool for the driver of the cluster belongs.
   * The pool cluster uses the instance pool with id (instance_pool_id) if the driver pool is not
   * assigned.
   */
  driver_instance_pool_id?: VariableOr<string>;
  /**
   * The node type of the Spark driver.
   * Note that this field is optional; if unset, the driver node type will be set as the same value
   * as `node_type_id` defined above.
   * 
   * This field, along with node_type_id, should not be set if virtual_cluster_size is set.
   * If both driver_node_type_id, node_type_id, and virtual_cluster_size are specified, driver_node_type_id and node_type_id take precedence.
   */
  driver_node_type_id?: VariableOr<string>;
  /**
   * Autoscaling Local Storage: when enabled, this cluster will dynamically acquire additional disk
   * space when its Spark workers are running low on disk space. This feature requires specific AWS
   * permissions to function correctly - refer to the User Guide for more details.
   */
  enable_elastic_disk?: VariableOr<boolean>;
  /**
   * Whether to enable LUKS on cluster VMs' local disks
   */
  enable_local_disk_encryption?: VariableOr<boolean>;
  /**
   * Attributes related to clusters running on Google Cloud Platform.
   * If not specified at cluster creation, a set of default values will be used.
   */
  gcp_attributes?: VariableOr<GcpAttributes>;
  /**
   * The configuration for storing init scripts. Any number of destinations can be specified.
   * The scripts are executed sequentially in the order provided.
   * If `cluster_log_conf` is specified, init script logs are sent to `<destination>/<cluster-ID>/init_scripts`.
   */
  init_scripts?: VariableOr<InitScriptInfo[]>;
  /**
   * The optional ID of the instance pool to which the cluster belongs.
   */
  instance_pool_id?: VariableOr<string>;
  /**
   * This field can only be used when `kind = CLASSIC_PREVIEW`.
   * 
   * When set to true, Databricks will automatically set single node related `custom_tags`, `spark_conf`, and `num_workers`
   */
  is_single_node?: VariableOr<boolean>;
  kind?: VariableOr<Kind>;
  /**
   * This field encodes, through a single value, the resources available to each of
   * the Spark nodes in this cluster. For example, the Spark nodes can be provisioned
   * and optimized for memory or compute intensive workloads. A list of available node
   * types can be retrieved by using the :method:clusters/listNodeTypes API call.
   */
  node_type_id?: VariableOr<string>;
  /**
   * Number of worker nodes that this cluster should have. A cluster has one Spark Driver
   * and `num_workers` Executors for a total of `num_workers` + 1 Spark nodes.
   * 
   * Note: When reading the properties of a cluster, this field reflects the desired number
   * of workers rather than the actual current number of workers. For instance, if a cluster
   * is resized from 5 to 10 workers, this field will immediately be updated to reflect
   * the target size of 10 workers, whereas the workers listed in `spark_info` will gradually
   * increase from 5 to 10 as the new nodes are provisioned.
   */
  num_workers?: VariableOr<number>;
  /**
   * The ID of the cluster policy used to create the cluster if applicable.
   */
  policy_id?: VariableOr<string>;
  /**
   * If set, what the configurable throughput (in Mb/s) for the remote disk is. Currently only supported for GCP HYPERDISK_BALANCED disks.
   */
  remote_disk_throughput?: VariableOr<number>;
  runtime_engine?: VariableOr<RuntimeEngine>;
  /**
   * Single user name if data_security_mode is `SINGLE_USER`
   */
  single_user_name?: VariableOr<string>;
  /**
   * An object containing a set of optional, user-specified Spark configuration key-value pairs.
   * Users can also pass in a string of extra JVM options to the driver and the executors via
   * `spark.driver.extraJavaOptions` and `spark.executor.extraJavaOptions` respectively.
   */
  spark_conf?: VariableOr<Record<string, string>>;
  /**
   * An object containing a set of optional, user-specified environment variable key-value pairs.
   * Please note that key-value pair of the form (X,Y) will be exported as is (i.e.,
   * `export X='Y'`) while launching the driver and workers.
   * 
   * In order to specify an additional set of `SPARK_DAEMON_JAVA_OPTS`, we recommend appending
   * them to `$SPARK_DAEMON_JAVA_OPTS` as shown in the example below. This ensures that all
   * default databricks managed environmental variables are included as well.
   * 
   * Example Spark environment variables:
   * `{"SPARK_WORKER_MEMORY": "28000m", "SPARK_LOCAL_DIRS": "/local_disk0"}` or
   * `{"SPARK_DAEMON_JAVA_OPTS": "$SPARK_DAEMON_JAVA_OPTS -Dspark.shuffle.service.enabled=true"}`
   */
  spark_env_vars?: VariableOr<Record<string, string>>;
  /**
   * The Spark version of the cluster, e.g. `3.3.x-scala2.11`.
   * A list of available Spark versions can be retrieved by using
   * the :method:clusters/sparkVersions API call.
   */
  spark_version?: VariableOr<string>;
  /**
   * SSH public key contents that will be added to each Spark node in this cluster. The
   * corresponding private keys can be used to login with the user name `ubuntu` on port `2200`.
   * Up to 10 keys can be specified.
   */
  ssh_public_keys?: VariableOr<string[]>;
  /**
   * If set, what the total initial volume size (in GB) of the remote disks should be. Currently only supported for GCP HYPERDISK_BALANCED disks.
   */
  total_initial_remote_disk_size?: VariableOr<number>;
  /**
   * This field can only be used when `kind = CLASSIC_PREVIEW`.
   * 
   * `effective_spark_version` is determined by `spark_version` (DBR release), this field `use_ml_runtime`, and whether `node_type_id` is gpu node or not.
   */
  use_ml_runtime?: VariableOr<boolean>;
  workload_type?: VariableOr<WorkloadType>;
}

/**
 * Data security mode decides what data governance model to use when accessing data
 * from a cluster.
 * 
 * The following modes can only be used when `kind = CLASSIC_PREVIEW`.
 * * `DATA_SECURITY_MODE_AUTO`: Databricks will choose the most appropriate access mode depending on your compute configuration.
 * * `DATA_SECURITY_MODE_STANDARD`: Alias for `USER_ISOLATION`.
 * * `DATA_SECURITY_MODE_DEDICATED`: Alias for `SINGLE_USER`.
 * 
 * The following modes can be used regardless of `kind`.
 * * `NONE`: No security isolation for multiple users sharing the cluster. Data governance features are not available in this mode.
 * * `SINGLE_USER`: A secure cluster that can only be exclusively used by a single user specified in `single_user_name`. Most programming languages, cluster features and data governance features are available in this mode.
 * * `USER_ISOLATION`: A secure cluster that can be shared by multiple users. Cluster users are fully isolated so that they cannot see each other's data and credentials. Most data governance features are supported in this mode. But programming languages and cluster features might be limited.
 * 
 * The following modes are deprecated starting with Databricks Runtime 15.0 and
 * will be removed for future Databricks Runtime versions:
 * 
 * * `LEGACY_TABLE_ACL`: This mode is for users migrating from legacy Table ACL clusters.
 * * `LEGACY_PASSTHROUGH`: This mode is for users migrating from legacy Passthrough on high concurrency clusters.
 * * `LEGACY_SINGLE_USER`: This mode is for users migrating from legacy Passthrough on standard clusters.
 * * `LEGACY_SINGLE_USER_STANDARD`: This mode provides a way that doesn’t have UC nor passthrough enabled.
 */
export type DataSecurityMode =
  | "NONE"
  | "SINGLE_USER"
  | "USER_ISOLATION"
  | "LEGACY_TABLE_ACL"
  | "LEGACY_PASSTHROUGH"
  | "LEGACY_SINGLE_USER"
  | "LEGACY_SINGLE_USER_STANDARD"
  | "DATA_SECURITY_MODE_STANDARD"
  | "DATA_SECURITY_MODE_DEDICATED"
  | "DATA_SECURITY_MODE_AUTO";

/**
 * A storage location in DBFS
 */
export interface DbfsStorageInfo {
  /**
   * dbfs destination, e.g. `dbfs:/my/path`
   */
  destination: VariableOr<string>;
}

export interface DockerBasicAuth {
  /**
   * Password of the user
   */
  password?: VariableOr<string>;
  /**
   * Name of the user
   */
  username?: VariableOr<string>;
}

export interface DockerImage {
  basic_auth?: VariableOr<DockerBasicAuth>;
  /**
   * URL of the docker image.
   */
  url?: VariableOr<string>;
}

/**
 * All EBS volume types that Databricks supports.
 * See https://aws.amazon.com/ebs/details/ for details.
 */
export type EbsVolumeType =
  | "GENERAL_PURPOSE_SSD"
  | "THROUGHPUT_OPTIMIZED_HDD";

/**
 * The environment entity used to preserve serverless environment side panel, jobs' environment for non-notebook task, and DLT's environment for classic and serverless pipelines.
 * In this minimal environment spec, only pip dependencies are supported.
 */
export interface Environment {
  /**
   * Use `environment_version` instead.
   * @deprecated
   */
  client?: VariableOr<string>;
  /**
   * List of pip dependencies, as supported by the version of pip in this environment.
   */
  dependencies?: VariableOr<string[]>;
  /**
   * Required. Environment version used by the environment.
   * Each version comes with a specific Python version and a set of Python packages.
   * The version is a string, consisting of an integer.
   */
  environment_version?: VariableOr<string>;
  /**
   * Use `java_dependencies` instead.
   * @deprecated
   */
  jar_dependencies?: VariableOr<string[]>;
  /**
   * List of jar dependencies, should be string representing volume paths. For example: `/Volumes/path/to/test.jar`.
   */
  java_dependencies?: VariableOr<string[]>;
}

/**
 * Attributes set during cluster creation which are related to GCP.
 */
export interface GcpAttributes {
  availability?: VariableOr<GcpAvailability>;
  /**
   * Boot disk size in GB
   */
  boot_disk_size?: VariableOr<number>;
  /**
   * The first `first_on_demand` nodes of the cluster will be placed on on-demand instances.
   * This value should be greater than 0, to make sure the cluster driver node is placed on an
   * on-demand instance. If this value is greater than or equal to the current cluster size, all
   * nodes will be placed on on-demand instances. If this value is less than the current cluster
   * size, `first_on_demand` nodes will be placed on on-demand instances and the remainder will
   * be placed on `availability` instances. Note that this value does not affect
   * cluster size and cannot currently be mutated over the lifetime of a cluster.
   */
  first_on_demand?: VariableOr<number>;
  /**
   * If provided, the cluster will impersonate the google service account when accessing
   * gcloud services (like GCS). The google service account
   * must have previously been added to the Databricks environment by an account
   * administrator.
   */
  google_service_account?: VariableOr<string>;
  /**
   * If provided, each node (workers and driver) in the cluster will have this number of local SSDs attached.
   * Each local SSD is 375GB in size.
   * Refer to [GCP documentation](https://cloud.google.com/compute/docs/disks/local-ssd#choose_number_local_ssds)
   * for the supported number of local SSDs for each instance type.
   */
  local_ssd_count?: VariableOr<number>;
  /**
   * This field determines whether the spark executors will be scheduled to run on preemptible
   * VMs (when set to true) versus standard compute engine VMs (when set to false; default).
   * Note: Soon to be deprecated, use the 'availability' field instead.
   * @deprecated
   */
  use_preemptible_executors?: VariableOr<boolean>;
  /**
   * Identifier for the availability zone in which the cluster resides.
   * This can be one of the following:
   * - "HA" => High availability, spread nodes across availability zones for a Databricks deployment region [default].
   * - "AUTO" => Databricks picks an availability zone to schedule the cluster on.
   * - A GCP availability zone => Pick One of the available zones for (machine type + region) from
   * https://cloud.google.com/compute/docs/regions-zones.
   */
  zone_id?: VariableOr<string>;
}

/**
 * This field determines whether the instance pool will contain preemptible
 * VMs, on-demand VMs, or preemptible VMs with a fallback to on-demand VMs if the former is unavailable.
 */
export type GcpAvailability =
  | "PREEMPTIBLE_GCP"
  | "ON_DEMAND_GCP"
  | "PREEMPTIBLE_WITH_FALLBACK_GCP";

/**
 * A storage location in Google Cloud Platform's GCS
 */
export interface GcsStorageInfo {
  /**
   * GCS destination/URI, e.g. `gs://my-bucket/some-prefix`
   */
  destination: VariableOr<string>;
}

/**
 * Config for an individual init script
 * Next ID: 11
 */
export interface InitScriptInfo {
  /**
   * Contains the Azure Data Lake Storage destination path
   */
  abfss?: VariableOr<Adlsgen2Info>;
  /**
   * destination needs to be provided. e.g.
   * `{ "dbfs": { "destination" : "dbfs:/home/cluster_log" } }`
   * @deprecated
   */
  dbfs?: VariableOr<DbfsStorageInfo>;
  /**
   * destination needs to be provided, e.g.
   * `{ "file": { "destination": "file:/my/local/file.sh" } }`
   */
  file?: VariableOr<LocalFileInfo>;
  /**
   * destination needs to be provided, e.g.
   * `{ "gcs": { "destination": "gs://my-bucket/file.sh" } }`
   */
  gcs?: VariableOr<GcsStorageInfo>;
  /**
   * destination and either the region or endpoint need to be provided. e.g.
   * `{ \"s3\": { \"destination\": \"s3://cluster_log_bucket/prefix\", \"region\": \"us-west-2\" } }`
   * Cluster iam role is used to access s3, please make sure the cluster iam role in
   * `instance_profile_arn` has permission to write data to the s3 destination.
   */
  s3?: VariableOr<S3StorageInfo>;
  /**
   * destination needs to be provided. e.g.
   * `{ \"volumes\" : { \"destination\" : \"/Volumes/my-init.sh\" } }`
   */
  volumes?: VariableOr<VolumesStorageInfo>;
  /**
   * destination needs to be provided, e.g.
   * `{ "workspace": { "destination": "/cluster-init-scripts/setup-datadog.sh" } }`
   */
  workspace?: VariableOr<WorkspaceStorageInfo>;
}

export type Kind =
  | "CLASSIC_PREVIEW";

export interface Library {
  /**
   * Specification of a CRAN library to be installed as part of the library
   */
  cran?: VariableOr<RCranLibrary>;
  /**
   * Deprecated. URI of the egg library to install. Installing Python egg files is deprecated and is not supported in Databricks Runtime 14.0 and above.
   * @deprecated
   */
  egg?: VariableOr<string>;
  /**
   * URI of the JAR library to install. Supported URIs include Workspace paths, Unity Catalog Volumes paths, and S3 URIs.
   * For example: `{ "jar": "/Workspace/path/to/library.jar" }`, `{ "jar" : "/Volumes/path/to/library.jar" }` or
   * `{ "jar": "s3://my-bucket/library.jar" }`.
   * If S3 is used, please make sure the cluster has read access on the library. You may need to
   * launch the cluster with an IAM role to access the S3 URI.
   */
  jar?: VariableOr<string>;
  /**
   * Specification of a maven library to be installed. For example:
   * `{ "coordinates": "org.jsoup:jsoup:1.7.2" }`
   */
  maven?: VariableOr<MavenLibrary>;
  /**
   * Specification of a PyPi library to be installed. For example:
   * `{ "package": "simplejson" }`
   */
  pypi?: VariableOr<PythonPyPiLibrary>;
  /**
   * URI of the requirements.txt file to install. Only Workspace paths and Unity Catalog Volumes paths are supported.
   * For example: `{ "requirements": "/Workspace/path/to/requirements.txt" }` or `{ "requirements" : "/Volumes/path/to/requirements.txt" }`
   */
  requirements?: VariableOr<string>;
  /**
   * URI of the wheel library to install. Supported URIs include Workspace paths, Unity Catalog Volumes paths, and S3 URIs.
   * For example: `{ "whl": "/Workspace/path/to/library.whl" }`, `{ "whl" : "/Volumes/path/to/library.whl" }` or
   * `{ "whl": "s3://my-bucket/library.whl" }`.
   * If S3 is used, please make sure the cluster has read access on the library. You may need to
   * launch the cluster with an IAM role to access the S3 URI.
   */
  whl?: VariableOr<string>;
}

export interface LocalFileInfo {
  /**
   * local file destination, e.g. `file:/my/local/file.sh`
   */
  destination: VariableOr<string>;
}

export interface LogAnalyticsInfo {
  /**
   * The primary key for the Azure Log Analytics agent configuration
   */
  log_analytics_primary_key?: VariableOr<string>;
  /**
   * The workspace ID for the Azure Log Analytics agent configuration
   */
  log_analytics_workspace_id?: VariableOr<string>;
}

export interface MavenLibrary {
  /**
   * Gradle-style maven coordinates. For example: "org.jsoup:jsoup:1.7.2".
   */
  coordinates: VariableOr<string>;
  /**
   * List of dependences to exclude. For example: `["slf4j:slf4j", "*:hadoop-client"]`.
   * 
   * Maven dependency exclusions:
   * https://maven.apache.org/guides/introduction/introduction-to-optional-and-excludes-dependencies.html.
   */
  exclusions?: VariableOr<string[]>;
  /**
   * Maven repo to install the Maven package from. If omitted, both Maven Central Repository
   * and Spark Packages are searched.
   */
  repo?: VariableOr<string>;
}

export interface PythonPyPiLibrary {
  /**
   * The name of the pypi package to install. An optional exact version specification is also
   * supported. Examples: "simplejson" and "simplejson==3.8.0".
   */
  package: VariableOr<string>;
  /**
   * The repository where the package can be found. If not specified, the default pip index is
   * used.
   */
  repo?: VariableOr<string>;
}

export interface RCranLibrary {
  /**
   * The name of the CRAN package to install.
   */
  package: VariableOr<string>;
  /**
   * The repository where the package can be found. If not specified, the default CRAN repo is used.
   */
  repo?: VariableOr<string>;
}

export type RuntimeEngine =
  | "NULL"
  | "STANDARD"
  | "PHOTON";

/**
 * A storage location in Amazon S3
 */
export interface S3StorageInfo {
  /**
   * (Optional) Set canned access control list for the logs, e.g. `bucket-owner-full-control`.
   * If `canned_cal` is set, please make sure the cluster iam role has `s3:PutObjectAcl` permission on
   * the destination bucket and prefix. The full list of possible canned acl can be found at
   * http://docs.aws.amazon.com/AmazonS3/latest/dev/acl-overview.html#canned-acl.
   * Please also note that by default only the object owner gets full controls. If you are using cross account
   * role for writing data, you may want to set `bucket-owner-full-control` to make bucket owner able to
   * read the logs.
   */
  canned_acl?: VariableOr<string>;
  /**
   * S3 destination, e.g. `s3://my-bucket/some-prefix` Note that logs will be delivered using
   * cluster iam role, please make sure you set cluster iam role and the role has write access to the
   * destination. Please also note that you cannot use AWS keys to deliver logs.
   */
  destination: VariableOr<string>;
  /**
   * (Optional) Flag to enable server side encryption, `false` by default.
   */
  enable_encryption?: VariableOr<boolean>;
  /**
   * (Optional) The encryption type, it could be `sse-s3` or `sse-kms`. It will be used only when
   * encryption is enabled and the default type is `sse-s3`.
   */
  encryption_type?: VariableOr<string>;
  /**
   * S3 endpoint, e.g. `https://s3-us-west-2.amazonaws.com`. Either region or endpoint needs to be set.
   * If both are set, endpoint will be used.
   */
  endpoint?: VariableOr<string>;
  /**
   * (Optional) Kms key which will be used if encryption is enabled and encryption type is set to `sse-kms`.
   */
  kms_key?: VariableOr<string>;
  /**
   * S3 region, e.g. `us-west-2`. Either region or endpoint needs to be set. If both are set,
   * endpoint will be used.
   */
  region?: VariableOr<string>;
}

/**
 * A storage location back by UC Volumes.
 */
export interface VolumesStorageInfo {
  /**
   * UC Volumes destination, e.g. `/Volumes/catalog/schema/vol1/init-scripts/setup-datadog.sh`
   * or `dbfs:/Volumes/catalog/schema/vol1/init-scripts/setup-datadog.sh`
   */
  destination: VariableOr<string>;
}

/**
 * Cluster Attributes showing for clusters workload types.
 */
export interface WorkloadType {
  /**
   * defined what type of clients can use the cluster. E.g. Notebooks, Jobs
   */
  clients: VariableOr<ClientsTypes>;
}

/**
 * A storage location in Workspace Filesystem (WSFS)
 */
export interface WorkspaceStorageInfo {
  /**
   * wsfs destination, e.g. `workspace:/cluster-init-scripts/setup-datadog.sh`
   */
  destination: VariableOr<string>;
}

export type AuthenticationMethod =
  | "OAUTH"
  | "PAT";

export interface CleanRoomsNotebookTask {
  /**
   * The clean room that the notebook belongs to.
   */
  clean_room_name: VariableOr<string>;
  /**
   * Checksum to validate the freshness of the notebook resource (i.e. the notebook being run is the latest version).
   * It can be fetched by calling the :method:cleanroomassets/get API.
   */
  etag?: VariableOr<string>;
  /**
   * Base parameters to be used for the clean room notebook job.
   */
  notebook_base_parameters?: VariableOr<Record<string, string>>;
  /**
   * Name of the notebook being run.
   */
  notebook_name: VariableOr<string>;
}

export interface ComputeConfig {
  /**
   * IDof the GPU pool to use.
   */
  gpu_node_pool_id?: VariableOr<string>;
  /**
   * GPU type.
   */
  gpu_type?: VariableOr<string>;
  /**
   * Number of GPUs.
   */
  num_gpus: VariableOr<number>;
}

export type Condition =
  | "ANY_UPDATED"
  | "ALL_UPDATED";

export interface ConditionTask {
  /**
   * The left operand of the condition task. Can be either a string value or a job state or parameter reference.
   */
  left: VariableOr<string>;
  /**
   * * `EQUAL_TO`, `NOT_EQUAL` operators perform string comparison of their operands. This means that `“12.0” == “12”` will evaluate to `false`.
   * * `GREATER_THAN`, `GREATER_THAN_OR_EQUAL`, `LESS_THAN`, `LESS_THAN_OR_EQUAL` operators perform numeric comparison of their operands. `“12.0” >= “12”` will evaluate to `true`, `“10.0” >= “12”` will evaluate to `false`.
   * 
   * The boolean comparison to task values can be implemented with operators `EQUAL_TO`, `NOT_EQUAL`. If a task value was set to a boolean value, it will be serialized to `“true”` or `“false”` for the comparison.
   */
  op: VariableOr<ConditionTaskOp>;
  /**
   * The right operand of the condition task. Can be either a string value or a job state or parameter reference.
   */
  right: VariableOr<string>;
}

/**
 * * `EQUAL_TO`, `NOT_EQUAL` operators perform string comparison of their operands. This means that `“12.0” == “12”` will evaluate to `false`.
 * * `GREATER_THAN`, `GREATER_THAN_OR_EQUAL`, `LESS_THAN`, `LESS_THAN_OR_EQUAL` operators perform numeric comparison of their operands. `“12.0” >= “12”` will evaluate to `true`, `“10.0” >= “12”` will evaluate to `false`.
 * 
 * The boolean comparison to task values can be implemented with operators `EQUAL_TO`, `NOT_EQUAL`. If a task value was set to a boolean value, it will be serialized to `“true”` or `“false”` for the comparison.
 */
export type ConditionTaskOp =
  | "EQUAL_TO"
  | "GREATER_THAN"
  | "GREATER_THAN_OR_EQUAL"
  | "LESS_THAN"
  | "LESS_THAN_OR_EQUAL"
  | "NOT_EQUAL";

export interface Continuous {
  /**
   * Indicate whether the continuous execution of the job is paused or not. Defaults to UNPAUSED.
   */
  pause_status?: VariableOr<PauseStatus>;
  /**
   * Indicate whether the continuous job is applying task level retries or not. Defaults to NEVER.
   */
  task_retry_mode?: VariableOr<TaskRetryMode>;
}

export interface CronSchedule {
  /**
   * Indicate whether this schedule is paused or not.
   */
  pause_status?: VariableOr<PauseStatus>;
  /**
   * A Cron expression using Quartz syntax that describes the schedule for a job. See [Cron Trigger](http://www.quartz-scheduler.org/documentation/quartz-2.3.0/tutorials/crontrigger.html) for details. This field is required.
   */
  quartz_cron_expression: VariableOr<string>;
  /**
   * A Java timezone ID. The schedule for a job is resolved with respect to this timezone. See [Java TimeZone](https://docs.oracle.com/javase/7/docs/api/java/util/TimeZone.html) for details. This field is required.
   */
  timezone_id: VariableOr<string>;
}

/**
 * Configures the Lakeview Dashboard job task type.
 */
export interface DashboardTask {
  dashboard_id?: VariableOr<string>;
  subscription?: VariableOr<Subscription>;
  /**
   * Optional: The warehouse id to execute the dashboard with for the schedule.
   * If not specified, the default warehouse of the dashboard will be used.
   */
  warehouse_id?: VariableOr<string>;
}

/**
 * Deprecated in favor of DbtPlatformTask
 */
export interface DbtCloudTask {
  /**
   * The resource name of the UC connection that authenticates the dbt Cloud for this task
   */
  connection_resource_name?: VariableOr<string>;
  /**
   * Id of the dbt Cloud job to be triggered
   */
  dbt_cloud_job_id?: VariableOr<number>;
}

export interface DbtPlatformTask {
  /**
   * The resource name of the UC connection that authenticates the dbt platform for this task
   */
  connection_resource_name?: VariableOr<string>;
  /**
   * Id of the dbt platform job to be triggered. Specified as a string for maximum compatibility with clients.
   */
  dbt_platform_job_id?: VariableOr<string>;
}

export interface DbtTask {
  /**
   * Optional name of the catalog to use. The value is the top level in the 3-level namespace of Unity Catalog (catalog / schema / relation). The catalog value can only be specified if a warehouse_id is specified. Requires dbt-databricks >= 1.1.1.
   */
  catalog?: VariableOr<string>;
  /**
   * A list of dbt commands to execute. All commands must start with `dbt`. This parameter must not be empty. A maximum of up to 10 commands can be provided.
   */
  commands: VariableOr<string[]>;
  /**
   * Optional (relative) path to the profiles directory. Can only be specified if no warehouse_id is specified. If no warehouse_id is specified and this folder is unset, the root directory is used.
   */
  profiles_directory?: VariableOr<string>;
  /**
   * Path to the project directory. Optional for Git sourced tasks, in which
   * case if no value is provided, the root of the Git repository is used.
   */
  project_directory?: VariableOr<string>;
  /**
   * Optional schema to write to. This parameter is only used when a warehouse_id is also provided. If not provided, the `default` schema is used.
   */
  schema?: VariableOr<string>;
  /**
   * Optional location type of the project directory. When set to `WORKSPACE`, the project will be retrieved
   * from the local Databricks workspace. When set to `GIT`, the project will be retrieved from a Git repository
   * defined in `git_source`. If the value is empty, the task will use `GIT` if `git_source` is defined and `WORKSPACE` otherwise.
   * 
   * * `WORKSPACE`: Project is located in Databricks workspace.
   * * `GIT`: Project is located in cloud Git provider.
   */
  source?: VariableOr<Source>;
  /**
   * ID of the SQL warehouse to connect to. If provided, we automatically generate and provide the profile and connection details to dbt. It can be overridden on a per-command basis by using the `--profiles-dir` command line argument.
   */
  warehouse_id?: VariableOr<string>;
}

export interface FileArrivalTriggerConfiguration {
  /**
   * If set, the trigger starts a run only after the specified amount of time passed since
   * the last time the trigger fired. The minimum allowed value is 60 seconds
   */
  min_time_between_triggers_seconds?: VariableOr<number>;
  /**
   * URL to be monitored for file arrivals. The path must point to the root or a subpath of the external location.
   */
  url: VariableOr<string>;
  /**
   * If set, the trigger starts a run only after no file activity has occurred for the specified amount of time.
   * This makes it possible to wait for a batch of incoming files to arrive before triggering a run. The
   * minimum allowed value is 60 seconds.
   */
  wait_after_last_change_seconds?: VariableOr<number>;
}

export interface ForEachTask {
  /**
   * An optional maximum allowed number of concurrent runs of the task.
   * Set this value if you want to be able to execute multiple runs of the task concurrently.
   */
  concurrency?: VariableOr<number>;
  /**
   * Array for task to iterate on. This can be a JSON string or a reference to
   * an array parameter.
   */
  inputs: VariableOr<string>;
  /**
   * Configuration for the task that will be run for each element in the array
   */
  task: VariableOr<Task>;
}

export interface GenAiComputeTask {
  /**
   * Command launcher to run the actual script, e.g. bash, python etc.
   */
  command?: VariableOr<string>;
  compute?: VariableOr<ComputeConfig>;
  /**
   * Runtime image
   */
  dl_runtime_image: VariableOr<string>;
  /**
   * Optional string containing the name of the MLflow experiment to log the run to. If name is not
   * found, backend will create the mlflow experiment using the name.
   */
  mlflow_experiment_name?: VariableOr<string>;
  /**
   * Optional location type of the training script. When set to `WORKSPACE`, the script will be retrieved from the local Databricks workspace. When set to `GIT`, the script will be retrieved from a Git repository
   * defined in `git_source`. If the value is empty, the task will use `GIT` if `git_source` is defined and `WORKSPACE` otherwise.
   * * `WORKSPACE`: Script is located in Databricks workspace.
   * * `GIT`: Script is located in cloud Git provider.
   */
  source?: VariableOr<Source>;
  /**
   * The training script file path to be executed. Cloud file URIs (such as dbfs:/, s3:/, adls:/, gcs:/) and workspace paths are supported. For python files stored in the Databricks workspace, the path must be absolute and begin with `/`. For files stored in a remote repository, the path must be relative. This field is required.
   */
  training_script_path?: VariableOr<string>;
  /**
   * Optional string containing model parameters passed to the training script in yaml format.
   * If present, then the content in yaml_parameters_file_path will be ignored.
   */
  yaml_parameters?: VariableOr<string>;
  /**
   * Optional path to a YAML file containing model parameters passed to the training script.
   */
  yaml_parameters_file_path?: VariableOr<string>;
}

export type GitProvider =
  | "gitHub"
  | "bitbucketCloud"
  | "azureDevOpsServices"
  | "gitHubEnterprise"
  | "bitbucketServer"
  | "gitLab"
  | "gitLabEnterpriseEdition"
  | "awsCodeCommit";

/**
 * An optional specification for a remote Git repository containing the source code used by tasks. Version-controlled source code is supported by notebook, dbt, Python script, and SQL File tasks.
 * 
 * If `git_source` is set, these tasks retrieve the file from the remote repository by default. However, this behavior can be overridden by setting `source` to `WORKSPACE` on the task.
 * 
 * Note: dbt and SQL File tasks support only version-controlled sources. If dbt or SQL File tasks are used, `git_source` must be defined on the job.
 */
export interface GitSource {
  /**
   * Name of the branch to be checked out and used by this job. This field cannot be specified in conjunction with git_tag or git_commit.
   */
  git_branch?: VariableOr<string>;
  /**
   * Commit to be checked out and used by this job. This field cannot be specified in conjunction with git_branch or git_tag.
   */
  git_commit?: VariableOr<string>;
  /**
   * Unique identifier of the service used to host the Git repository. The value is case insensitive.
   */
  git_provider: VariableOr<GitProvider>;
  /**
   * Name of the tag to be checked out and used by this job. This field cannot be specified in conjunction with git_branch or git_commit.
   */
  git_tag?: VariableOr<string>;
  /**
   * URL of the repository to be cloned by this job.
   */
  git_url: VariableOr<string>;
}

export interface JobCluster {
  /**
   * A unique name for the job cluster. This field is required and must be unique within the job.
   * `JobTaskSettings` may refer to this field to determine which cluster to launch for the task execution.
   */
  job_cluster_key: VariableOr<string>;
  /**
   * If new_cluster, a description of a cluster that is created for each task.
   */
  new_cluster: VariableOr<ClusterSpec>;
}

export interface JobEmailNotifications {
  /**
   * If true, do not send email to recipients specified in `on_failure` if the run is skipped.
   * This field is `deprecated`. Please use the `notification_settings.no_alert_for_skipped_runs` field.
   * @deprecated
   */
  no_alert_for_skipped_runs?: VariableOr<boolean>;
  /**
   * A list of email addresses to be notified when the duration of a run exceeds the threshold specified for the `RUN_DURATION_SECONDS` metric in the `health` field. If no rule for the `RUN_DURATION_SECONDS` metric is specified in the `health` field for the job, notifications are not sent.
   */
  on_duration_warning_threshold_exceeded?: VariableOr<string[]>;
  /**
   * A list of email addresses to be notified when a run unsuccessfully completes. A run is considered to have completed unsuccessfully if it ends with an `INTERNAL_ERROR` `life_cycle_state` or a `FAILED`, or `TIMED_OUT` result_state. If this is not specified on job creation, reset, or update the list is empty, and notifications are not sent.
   */
  on_failure?: VariableOr<string[]>;
  /**
   * A list of email addresses to be notified when a run begins. If not specified on job creation, reset, or update, the list is empty, and notifications are not sent.
   */
  on_start?: VariableOr<string[]>;
  /**
   * A list of email addresses to notify when any streaming backlog thresholds are exceeded for any stream.
   * Streaming backlog thresholds can be set in the `health` field using the following metrics: `STREAMING_BACKLOG_BYTES`, `STREAMING_BACKLOG_RECORDS`, `STREAMING_BACKLOG_SECONDS`, or `STREAMING_BACKLOG_FILES`.
   * Alerting is based on the 10-minute average of these metrics. If the issue persists, notifications are resent every 30 minutes.
   */
  on_streaming_backlog_exceeded?: VariableOr<string[]>;
  /**
   * A list of email addresses to be notified when a run successfully completes. A run is considered to have completed successfully if it ends with a `TERMINATED` `life_cycle_state` and a `SUCCESS` result_state. If not specified on job creation, reset, or update, the list is empty, and notifications are not sent.
   */
  on_success?: VariableOr<string[]>;
}

export interface JobEnvironment {
  /**
   * The key of an environment. It has to be unique within a job.
   */
  environment_key: VariableOr<string>;
  spec?: VariableOr<Environment>;
}

export interface JobNotificationSettings {
  /**
   * If true, do not send notifications to recipients specified in `on_failure` if the run is canceled.
   */
  no_alert_for_canceled_runs?: VariableOr<boolean>;
  /**
   * If true, do not send notifications to recipients specified in `on_failure` if the run is skipped.
   */
  no_alert_for_skipped_runs?: VariableOr<boolean>;
}

export interface JobParameterDefinition {
  /**
   * Default value of the parameter.
   */
  default: VariableOr<string>;
  /**
   * The name of the defined parameter. May only contain alphanumeric characters, `_`, `-`, and `.`
   */
  name: VariableOr<string>;
}

/**
 * Write-only setting. Specifies the user or service principal that the job runs as. If not specified, the job runs as the user who created the job.
 * 
 * Either `user_name` or `service_principal_name` should be specified. If not, an error is thrown.
 */
export interface JobRunAs {
  /**
   * The application ID of an active service principal. Setting this field requires the `servicePrincipal/user` role.
   */
  service_principal_name?: VariableOr<string>;
  /**
   * The email of an active workspace user. Non-admin users can only set this field to their own email.
   */
  user_name?: VariableOr<string>;
}

/**
 * Specifies the health metric that is being evaluated for a particular health rule.
 * 
 * * `RUN_DURATION_SECONDS`: Expected total time for a run in seconds.
 * * `STREAMING_BACKLOG_BYTES`: An estimate of the maximum bytes of data waiting to be consumed across all streams. This metric is in Public Preview.
 * * `STREAMING_BACKLOG_RECORDS`: An estimate of the maximum offset lag across all streams. This metric is in Public Preview.
 * * `STREAMING_BACKLOG_SECONDS`: An estimate of the maximum consumer delay across all streams. This metric is in Public Preview.
 * * `STREAMING_BACKLOG_FILES`: An estimate of the maximum number of outstanding files across all streams. This metric is in Public Preview.
 */
export type JobsHealthMetric =
  | "RUN_DURATION_SECONDS"
  | "STREAMING_BACKLOG_BYTES"
  | "STREAMING_BACKLOG_RECORDS"
  | "STREAMING_BACKLOG_SECONDS"
  | "STREAMING_BACKLOG_FILES";

/**
 * Specifies the operator used to compare the health metric value with the specified threshold.
 */
export type JobsHealthOperator =
  | "GREATER_THAN";

export interface JobsHealthRule {
  metric: VariableOr<JobsHealthMetric>;
  op: VariableOr<JobsHealthOperator>;
  /**
   * Specifies the threshold value that the health metric should obey to satisfy the health rule.
   */
  value: VariableOr<number>;
}

/**
 * An optional set of health rules that can be defined for this job.
 */
export interface JobsHealthRules {
  rules?: VariableOr<JobsHealthRule[]>;
}

export interface NotebookTask {
  /**
   * Base parameters to be used for each run of this job. If the run is initiated by a call to :method:jobs/run
   * Now with parameters specified, the two parameters maps are merged. If the same key is specified in
   * `base_parameters` and in `run-now`, the value from `run-now` is used.
   * Use [Task parameter variables](https://docs.databricks.com/jobs.html#parameter-variables) to set parameters containing information about job runs.
   * 
   * If the notebook takes a parameter that is not specified in the job’s `base_parameters` or the `run-now` override parameters,
   * the default value from the notebook is used.
   * 
   * Retrieve these parameters in a notebook using [dbutils.widgets.get](https://docs.databricks.com/dev-tools/databricks-utils.html#dbutils-widgets).
   * 
   * The JSON representation of this field cannot exceed 1MB.
   */
  base_parameters?: VariableOr<Record<string, string>>;
  /**
   * The path of the notebook to be run in the Databricks workspace or remote repository.
   * For notebooks stored in the Databricks workspace, the path must be absolute and begin with a slash.
   * For notebooks stored in a remote repository, the path must be relative. This field is required.
   */
  notebook_path: VariableOr<string>;
  /**
   * Optional location type of the notebook. When set to `WORKSPACE`, the notebook will be retrieved from the local Databricks workspace. When set to `GIT`, the notebook will be retrieved from a Git repository
   * defined in `git_source`. If the value is empty, the task will use `GIT` if `git_source` is defined and `WORKSPACE` otherwise.
   * * `WORKSPACE`: Notebook is located in Databricks workspace.
   * * `GIT`: Notebook is located in cloud Git provider.
   */
  source?: VariableOr<Source>;
  /**
   * Optional `warehouse_id` to run the notebook on a SQL warehouse. Classic SQL warehouses are NOT supported, please use serverless or pro SQL warehouses.
   * 
   * Note that SQL warehouses only support SQL cells; if the notebook contains non-SQL cells, the run will fail.
   */
  warehouse_id?: VariableOr<string>;
}

export type PauseStatus =
  | "UNPAUSED"
  | "PAUSED";

/**
 * PerformanceTarget defines how performant (lower latency) or cost efficient the execution of run on serverless compute should be.
 * The performance mode on the job or pipeline should map to a performance setting that is passed to Cluster Manager
 * (see cluster-common PerformanceTarget).
 */
export type PerformanceTarget =
  | "PERFORMANCE_OPTIMIZED"
  | "STANDARD";

export interface PeriodicTriggerConfiguration {
  /**
   * The interval at which the trigger should run.
   */
  interval: VariableOr<number>;
  /**
   * The unit of time for the interval.
   */
  unit: VariableOr<PeriodicTriggerConfigurationTimeUnit>;
}

export type PeriodicTriggerConfigurationTimeUnit =
  | "HOURS"
  | "DAYS"
  | "WEEKS";

export interface PipelineParams {
  /**
   * If true, triggers a full refresh on the delta live table.
   */
  full_refresh?: VariableOr<boolean>;
}

export interface PipelineTask {
  /**
   * If true, triggers a full refresh on the delta live table.
   */
  full_refresh?: VariableOr<boolean>;
  /**
   * The full name of the pipeline task to execute.
   */
  pipeline_id: VariableOr<string>;
}

export interface PowerBiModel {
  /**
   * How the published Power BI model authenticates to Databricks
   */
  authentication_method?: VariableOr<AuthenticationMethod>;
  /**
   * The name of the Power BI model
   */
  model_name?: VariableOr<string>;
  /**
   * Whether to overwrite existing Power BI models
   */
  overwrite_existing?: VariableOr<boolean>;
  /**
   * The default storage mode of the Power BI model
   */
  storage_mode?: VariableOr<StorageMode>;
  /**
   * The name of the Power BI workspace of the model
   */
  workspace_name?: VariableOr<string>;
}

export interface PowerBiTable {
  /**
   * The catalog name in Databricks
   */
  catalog?: VariableOr<string>;
  /**
   * The table name in Databricks
   */
  name?: VariableOr<string>;
  /**
   * The schema name in Databricks
   */
  schema?: VariableOr<string>;
  /**
   * The Power BI storage mode of the table
   */
  storage_mode?: VariableOr<StorageMode>;
}

export interface PowerBiTask {
  /**
   * The resource name of the UC connection to authenticate from Databricks to Power BI
   */
  connection_resource_name?: VariableOr<string>;
  /**
   * The semantic model to update
   */
  power_bi_model?: VariableOr<PowerBiModel>;
  /**
   * Whether the model should be refreshed after the update
   */
  refresh_after_update?: VariableOr<boolean>;
  /**
   * The tables to be exported to Power BI
   */
  tables?: VariableOr<PowerBiTable[]>;
  /**
   * The SQL warehouse ID to use as the Power BI data source
   */
  warehouse_id?: VariableOr<string>;
}

export interface PythonWheelTask {
  /**
   * Named entry point to use, if it does not exist in the metadata of the package it executes the function from the package directly using `$packageName.$entryPoint()`
   */
  entry_point: VariableOr<string>;
  /**
   * Command-line parameters passed to Python wheel task in the form of `["--name=task", "--data=dbfs:/path/to/data.json"]`. Leave it empty if `parameters` is not null.
   */
  named_parameters?: VariableOr<Record<string, string>>;
  /**
   * Name of the package to execute
   */
  package_name: VariableOr<string>;
  /**
   * Command-line parameters passed to Python wheel task. Leave it empty if `named_parameters` is not null.
   */
  parameters?: VariableOr<string[]>;
}

export interface QueueSettings {
  /**
   * If true, enable queueing for the job. This is a required field.
   */
  enabled: VariableOr<boolean>;
}

/**
 * An optional value indicating the condition that determines whether the task should be run once its dependencies have been completed. When omitted, defaults to `ALL_SUCCESS`.
 * 
 * Possible values are:
 * * `ALL_SUCCESS`: All dependencies have executed and succeeded
 * * `AT_LEAST_ONE_SUCCESS`: At least one dependency has succeeded
 * * `NONE_FAILED`: None of the dependencies have failed and at least one was executed
 * * `ALL_DONE`: All dependencies have been completed
 * * `AT_LEAST_ONE_FAILED`: At least one dependency failed
 * * `ALL_FAILED`: ALl dependencies have failed
 */
export type RunIf =
  | "ALL_SUCCESS"
  | "ALL_DONE"
  | "NONE_FAILED"
  | "AT_LEAST_ONE_SUCCESS"
  | "ALL_FAILED"
  | "AT_LEAST_ONE_FAILED";

export interface RunJobTask {
  /**
   * An array of commands to execute for jobs with the dbt task, for example `"dbt_commands": ["dbt deps", "dbt seed", "dbt deps", "dbt seed", "dbt run"]`
   * @deprecated
   */
  dbt_commands?: VariableOr<string[]>;
  /**
   * A list of parameters for jobs with Spark JAR tasks, for example `"jar_params": ["john doe", "35"]`.
   * The parameters are used to invoke the main function of the main class specified in the Spark JAR task.
   * If not specified upon `run-now`, it defaults to an empty list.
   * jar_params cannot be specified in conjunction with notebook_params.
   * The JSON representation of this field (for example `{"jar_params":["john doe","35"]}`) cannot exceed 10,000 bytes.
   * 
   * Use [Task parameter variables](https://docs.databricks.com/jobs.html#parameter-variables) to set parameters containing information about job runs.
   * @deprecated
   */
  jar_params?: VariableOr<string[]>;
  /**
   * ID of the job to trigger.
   */
  job_id: VariableOr<number>;
  /**
   * Job-level parameters used to trigger the job.
   */
  job_parameters?: VariableOr<Record<string, string>>;
  /**
   * A map from keys to values for jobs with notebook task, for example `"notebook_params": {"name": "john doe", "age": "35"}`.
   * The map is passed to the notebook and is accessible through the [dbutils.widgets.get](https://docs.databricks.com/dev-tools/databricks-utils.html) function.
   * 
   * If not specified upon `run-now`, the triggered run uses the job’s base parameters.
   * 
   * notebook_params cannot be specified in conjunction with jar_params.
   * 
   * Use [Task parameter variables](https://docs.databricks.com/jobs.html#parameter-variables) to set parameters containing information about job runs.
   * 
   * The JSON representation of this field (for example `{"notebook_params":{"name":"john doe","age":"35"}}`) cannot exceed 10,000 bytes.
   * @deprecated
   */
  notebook_params?: VariableOr<Record<string, string>>;
  /**
   * Controls whether the pipeline should perform a full refresh
   */
  pipeline_params?: VariableOr<PipelineParams>;
  python_named_params?: VariableOr<Record<string, string>>;
  /**
   * A list of parameters for jobs with Python tasks, for example `"python_params": ["john doe", "35"]`.
   * The parameters are passed to Python file as command-line parameters. If specified upon `run-now`, it would overwrite
   * the parameters specified in job setting. The JSON representation of this field (for example `{"python_params":["john doe","35"]}`)
   * cannot exceed 10,000 bytes.
   * 
   * Use [Task parameter variables](https://docs.databricks.com/jobs.html#parameter-variables) to set parameters containing information about job runs.
   * 
   * Important
   * 
   * These parameters accept only Latin characters (ASCII character set). Using non-ASCII characters returns an error.
   * Examples of invalid, non-ASCII characters are Chinese, Japanese kanjis, and emojis.
   * @deprecated
   */
  python_params?: VariableOr<string[]>;
  /**
   * A list of parameters for jobs with spark submit task, for example `"spark_submit_params": ["--class", "org.apache.spark.examples.SparkPi"]`.
   * The parameters are passed to spark-submit script as command-line parameters. If specified upon `run-now`, it would overwrite the
   * parameters specified in job setting. The JSON representation of this field (for example `{"python_params":["john doe","35"]}`)
   * cannot exceed 10,000 bytes.
   * 
   * Use [Task parameter variables](https://docs.databricks.com/jobs.html#parameter-variables) to set parameters containing information about job runs
   * 
   * Important
   * 
   * These parameters accept only Latin characters (ASCII character set). Using non-ASCII characters returns an error.
   * Examples of invalid, non-ASCII characters are Chinese, Japanese kanjis, and emojis.
   * @deprecated
   */
  spark_submit_params?: VariableOr<string[]>;
  /**
   * A map from keys to values for jobs with SQL task, for example `"sql_params": {"name": "john doe", "age": "35"}`. The SQL alert task does not support custom parameters.
   * @deprecated
   */
  sql_params?: VariableOr<Record<string, string>>;
}

/**
 * Optional location type of the SQL file. When set to `WORKSPACE`, the SQL file will be retrieved\
 * from the local Databricks workspace. When set to `GIT`, the SQL file will be retrieved from a Git repository
 * defined in `git_source`. If the value is empty, the task will use `GIT` if `git_source` is defined and `WORKSPACE` otherwise.
 * 
 * * `WORKSPACE`: SQL file is located in Databricks workspace.
 * * `GIT`: SQL file is located in cloud Git provider.
 */
export type Source =
  | "WORKSPACE"
  | "GIT";

export interface SparkJarTask {
  /**
   * Deprecated since 04/2016. Provide a `jar` through the `libraries` field instead. For an example, see :method:jobs/create.
   * @deprecated
   */
  jar_uri?: VariableOr<string>;
  /**
   * The full name of the class containing the main method to be executed. This class must be contained in a JAR provided as a library.
   * 
   * The code must use `SparkContext.getOrCreate` to obtain a Spark context; otherwise, runs of the job fail.
   */
  main_class_name?: VariableOr<string>;
  /**
   * Parameters passed to the main method.
   * 
   * Use [Task parameter variables](https://docs.databricks.com/jobs.html#parameter-variables) to set parameters containing information about job runs.
   */
  parameters?: VariableOr<string[]>;
  /**
   * Deprecated. A value of `false` is no longer supported.
   * @deprecated
   */
  run_as_repl?: VariableOr<boolean>;
}

export interface SparkPythonTask {
  /**
   * Command line parameters passed to the Python file.
   * 
   * Use [Task parameter variables](https://docs.databricks.com/jobs.html#parameter-variables) to set parameters containing information about job runs.
   */
  parameters?: VariableOr<string[]>;
  /**
   * The Python file to be executed. Cloud file URIs (such as dbfs:/, s3:/, adls:/, gcs:/) and workspace paths are supported. For python files stored in the Databricks workspace, the path must be absolute and begin with `/`. For files stored in a remote repository, the path must be relative. This field is required.
   */
  python_file: VariableOr<string>;
  /**
   * Optional location type of the Python file. When set to `WORKSPACE` or not specified, the file will be retrieved from the local
   * Databricks workspace or cloud location (if the `python_file` has a URI format). When set to `GIT`,
   * the Python file will be retrieved from a Git repository defined in `git_source`.
   * 
   * * `WORKSPACE`: The Python file is located in a Databricks workspace or at a cloud filesystem URI.
   * * `GIT`: The Python file is located in a remote Git repository.
   */
  source?: VariableOr<Source>;
}

export interface SparkSubmitTask {
  /**
   * Command-line parameters passed to spark submit.
   * 
   * Use [Task parameter variables](https://docs.databricks.com/jobs.html#parameter-variables) to set parameters containing information about job runs.
   */
  parameters?: VariableOr<string[]>;
}

export interface SqlTask {
  /**
   * If alert, indicates that this job must refresh a SQL alert.
   */
  alert?: VariableOr<SqlTaskAlert>;
  /**
   * If dashboard, indicates that this job must refresh a SQL dashboard.
   */
  dashboard?: VariableOr<SqlTaskDashboard>;
  /**
   * If file, indicates that this job runs a SQL file in a remote Git repository.
   */
  file?: VariableOr<SqlTaskFile>;
  /**
   * Parameters to be used for each run of this job. The SQL alert task does not support custom parameters.
   */
  parameters?: VariableOr<Record<string, string>>;
  /**
   * If query, indicates that this job must execute a SQL query.
   */
  query?: VariableOr<SqlTaskQuery>;
  /**
   * The canonical identifier of the SQL warehouse. Recommended to use with serverless or pro SQL warehouses. Classic SQL warehouses are only supported for SQL alert, dashboard and query tasks and are limited to scheduled single-task jobs.
   */
  warehouse_id: VariableOr<string>;
}

export interface SqlTaskAlert {
  /**
   * The canonical identifier of the SQL alert.
   */
  alert_id: VariableOr<string>;
  /**
   * If true, the alert notifications are not sent to subscribers.
   */
  pause_subscriptions?: VariableOr<boolean>;
  /**
   * If specified, alert notifications are sent to subscribers.
   */
  subscriptions?: VariableOr<SqlTaskSubscription[]>;
}

export interface SqlTaskDashboard {
  /**
   * Subject of the email sent to subscribers of this task.
   */
  custom_subject?: VariableOr<string>;
  /**
   * The canonical identifier of the SQL dashboard.
   */
  dashboard_id: VariableOr<string>;
  /**
   * If true, the dashboard snapshot is not taken, and emails are not sent to subscribers.
   */
  pause_subscriptions?: VariableOr<boolean>;
  /**
   * If specified, dashboard snapshots are sent to subscriptions.
   */
  subscriptions?: VariableOr<SqlTaskSubscription[]>;
}

export interface SqlTaskFile {
  /**
   * Path of the SQL file. Must be relative if the source is a remote Git repository and absolute for workspace paths.
   */
  path: VariableOr<string>;
  /**
   * Optional location type of the SQL file. When set to `WORKSPACE`, the SQL file will be retrieved
   * from the local Databricks workspace. When set to `GIT`, the SQL file will be retrieved from a Git repository
   * defined in `git_source`. If the value is empty, the task will use `GIT` if `git_source` is defined and `WORKSPACE` otherwise.
   * 
   * * `WORKSPACE`: SQL file is located in Databricks workspace.
   * * `GIT`: SQL file is located in cloud Git provider.
   */
  source?: VariableOr<Source>;
}

export interface SqlTaskQuery {
  /**
   * The canonical identifier of the SQL query.
   */
  query_id: VariableOr<string>;
}

export interface SqlTaskSubscription {
  /**
   * The canonical identifier of the destination to receive email notification. This parameter is mutually exclusive with user_name. You cannot set both destination_id and user_name for subscription notifications.
   */
  destination_id?: VariableOr<string>;
  /**
   * The user name to receive the subscription email. This parameter is mutually exclusive with destination_id. You cannot set both destination_id and user_name for subscription notifications.
   */
  user_name?: VariableOr<string>;
}

export type StorageMode =
  | "DIRECT_QUERY"
  | "IMPORT"
  | "DUAL";

export interface Subscription {
  /**
   * Optional: Allows users to specify a custom subject line on the email sent
   * to subscribers.
   */
  custom_subject?: VariableOr<string>;
  /**
   * When true, the subscription will not send emails.
   */
  paused?: VariableOr<boolean>;
  subscribers?: VariableOr<SubscriptionSubscriber[]>;
}

export interface SubscriptionSubscriber {
  destination_id?: VariableOr<string>;
  user_name?: VariableOr<string>;
}

export interface TableUpdateTriggerConfiguration {
  /**
   * The table(s) condition based on which to trigger a job run.
   */
  condition?: VariableOr<Condition>;
  /**
   * If set, the trigger starts a run only after the specified amount of time has passed since
   * the last time the trigger fired. The minimum allowed value is 60 seconds.
   */
  min_time_between_triggers_seconds?: VariableOr<number>;
  /**
   * A list of Delta tables to monitor for changes. The table name must be in the format `catalog_name.schema_name.table_name`.
   */
  table_names?: VariableOr<string[]>;
  /**
   * If set, the trigger starts a run only after no table updates have occurred for the specified time
   * and can be used to wait for a series of table updates before triggering a run. The
   * minimum allowed value is 60 seconds.
   */
  wait_after_last_change_seconds?: VariableOr<number>;
}

export interface Task {
  /**
   * The task runs a [clean rooms](https://docs.databricks.com/clean-rooms/index.html) notebook
   * when the `clean_rooms_notebook_task` field is present.
   */
  clean_rooms_notebook_task?: VariableOr<CleanRoomsNotebookTask>;
  /**
   * The task evaluates a condition that can be used to control the execution of other tasks when the `condition_task` field is present.
   * The condition task does not require a cluster to execute and does not support retries or notifications.
   */
  condition_task?: VariableOr<ConditionTask>;
  /**
   * The task refreshes a dashboard and sends a snapshot to subscribers.
   */
  dashboard_task?: VariableOr<DashboardTask>;
  /**
   * Task type for dbt cloud, deprecated in favor of the new name dbt_platform_task
   * @deprecated
   */
  dbt_cloud_task?: VariableOr<DbtCloudTask>;
  dbt_platform_task?: VariableOr<DbtPlatformTask>;
  /**
   * The task runs one or more dbt commands when the `dbt_task` field is present. The dbt task requires both Databricks SQL and the ability to use a serverless or a pro SQL warehouse.
   */
  dbt_task?: VariableOr<DbtTask>;
  /**
   * An optional array of objects specifying the dependency graph of the task. All tasks specified in this field must complete before executing this task. The task will run only if the `run_if` condition is true.
   * The key is `task_key`, and the value is the name assigned to the dependent task.
   */
  depends_on?: VariableOr<TaskDependency[]>;
  /**
   * An optional description for this task.
   */
  description?: VariableOr<string>;
  /**
   * An option to disable auto optimization in serverless
   */
  disable_auto_optimization?: VariableOr<boolean>;
  /**
   * An optional flag to disable the task. If set to true, the task will not run even if it is part of a job.
   */
  disabled?: VariableOr<boolean>;
  /**
   * An optional set of email addresses that is notified when runs of this task begin or complete as well as when this task is deleted. The default behavior is to not send any emails.
   */
  email_notifications?: VariableOr<TaskEmailNotifications>;
  /**
   * The key that references an environment spec in a job. This field is required for Python script, Python wheel and dbt tasks when using serverless compute.
   */
  environment_key?: VariableOr<string>;
  /**
   * If existing_cluster_id, the ID of an existing cluster that is used for all runs.
   * When running jobs or tasks on an existing cluster, you may need to manually restart
   * the cluster if it stops responding. We suggest running jobs and tasks on new clusters for
   * greater reliability
   */
  existing_cluster_id?: VariableOr<string>;
  /**
   * The task executes a nested task for every input provided when the `for_each_task` field is present.
   */
  for_each_task?: VariableOr<ForEachTask>;
  gen_ai_compute_task?: VariableOr<GenAiComputeTask>;
  health?: VariableOr<JobsHealthRules>;
  /**
   * If job_cluster_key, this task is executed reusing the cluster specified in `job.settings.job_clusters`.
   */
  job_cluster_key?: VariableOr<string>;
  /**
   * An optional list of libraries to be installed on the cluster.
   * The default value is an empty list.
   */
  libraries?: VariableOr<Library[]>;
  /**
   * An optional maximum number of times to retry an unsuccessful run. A run is considered to be unsuccessful if it completes with the `FAILED` result_state or `INTERNAL_ERROR` `life_cycle_state`. The value `-1` means to retry indefinitely and the value `0` means to never retry.
   */
  max_retries?: VariableOr<number>;
  /**
   * An optional minimal interval in milliseconds between the start of the failed run and the subsequent retry run. The default behavior is that unsuccessful runs are immediately retried.
   */
  min_retry_interval_millis?: VariableOr<number>;
  /**
   * If new_cluster, a description of a new cluster that is created for each run.
   */
  new_cluster?: VariableOr<ClusterSpec>;
  /**
   * The task runs a notebook when the `notebook_task` field is present.
   */
  notebook_task?: VariableOr<NotebookTask>;
  /**
   * Optional notification settings that are used when sending notifications to each of the `email_notifications` and `webhook_notifications` for this task.
   */
  notification_settings?: VariableOr<TaskNotificationSettings>;
  /**
   * The task triggers a pipeline update when the `pipeline_task` field is present. Only pipelines configured to use triggered more are supported.
   */
  pipeline_task?: VariableOr<PipelineTask>;
  /**
   * The task triggers a Power BI semantic model update when the `power_bi_task` field is present.
   */
  power_bi_task?: VariableOr<PowerBiTask>;
  /**
   * The task runs a Python wheel when the `python_wheel_task` field is present.
   */
  python_wheel_task?: VariableOr<PythonWheelTask>;
  /**
   * An optional policy to specify whether to retry a job when it times out. The default behavior
   * is to not retry on timeout.
   */
  retry_on_timeout?: VariableOr<boolean>;
  /**
   * An optional value specifying the condition determining whether the task is run once its dependencies have been completed.
   * 
   * * `ALL_SUCCESS`: All dependencies have executed and succeeded
   * * `AT_LEAST_ONE_SUCCESS`: At least one dependency has succeeded
   * * `NONE_FAILED`: None of the dependencies have failed and at least one was executed
   * * `ALL_DONE`: All dependencies have been completed
   * * `AT_LEAST_ONE_FAILED`: At least one dependency failed
   * * `ALL_FAILED`: ALl dependencies have failed
   */
  run_if?: VariableOr<RunIf>;
  /**
   * The task triggers another job when the `run_job_task` field is present.
   */
  run_job_task?: VariableOr<RunJobTask>;
  /**
   * The task runs a JAR when the `spark_jar_task` field is present.
   */
  spark_jar_task?: VariableOr<SparkJarTask>;
  /**
   * The task runs a Python file when the `spark_python_task` field is present.
   */
  spark_python_task?: VariableOr<SparkPythonTask>;
  /**
   * (Legacy) The task runs the spark-submit script when the spark_submit_task field is present. Databricks recommends using the spark_jar_task instead; see [Spark Submit task for jobs](/jobs/spark-submit).
   * @deprecated
   */
  spark_submit_task?: VariableOr<SparkSubmitTask>;
  /**
   * The task runs a SQL query or file, or it refreshes a SQL alert or a legacy SQL dashboard when the `sql_task` field is present.
   */
  sql_task?: VariableOr<SqlTask>;
  /**
   * A unique name for the task. This field is used to refer to this task from other tasks.
   * This field is required and must be unique within its parent job.
   * On Update or Reset, this field is used to reference the tasks to be updated or reset.
   */
  task_key: VariableOr<string>;
  /**
   * An optional timeout applied to each run of this job task. A value of `0` means no timeout.
   */
  timeout_seconds?: VariableOr<number>;
  /**
   * A collection of system notification IDs to notify when runs of this task begin or complete. The default behavior is to not send any system notifications.
   */
  webhook_notifications?: VariableOr<WebhookNotifications>;
}

export interface TaskDependency {
  /**
   * Can only be specified on condition task dependencies. The outcome of the dependent task that must be met for this task to run.
   */
  outcome?: VariableOr<string>;
  /**
   * The name of the task this task depends on.
   */
  task_key: VariableOr<string>;
}

export interface TaskEmailNotifications {
  /**
   * If true, do not send email to recipients specified in `on_failure` if the run is skipped.
   * This field is `deprecated`. Please use the `notification_settings.no_alert_for_skipped_runs` field.
   * @deprecated
   */
  no_alert_for_skipped_runs?: VariableOr<boolean>;
  /**
   * A list of email addresses to be notified when the duration of a run exceeds the threshold specified for the `RUN_DURATION_SECONDS` metric in the `health` field. If no rule for the `RUN_DURATION_SECONDS` metric is specified in the `health` field for the job, notifications are not sent.
   */
  on_duration_warning_threshold_exceeded?: VariableOr<string[]>;
  /**
   * A list of email addresses to be notified when a run unsuccessfully completes. A run is considered to have completed unsuccessfully if it ends with an `INTERNAL_ERROR` `life_cycle_state` or a `FAILED`, or `TIMED_OUT` result_state. If this is not specified on job creation, reset, or update the list is empty, and notifications are not sent.
   */
  on_failure?: VariableOr<string[]>;
  /**
   * A list of email addresses to be notified when a run begins. If not specified on job creation, reset, or update, the list is empty, and notifications are not sent.
   */
  on_start?: VariableOr<string[]>;
  /**
   * A list of email addresses to notify when any streaming backlog thresholds are exceeded for any stream.
   * Streaming backlog thresholds can be set in the `health` field using the following metrics: `STREAMING_BACKLOG_BYTES`, `STREAMING_BACKLOG_RECORDS`, `STREAMING_BACKLOG_SECONDS`, or `STREAMING_BACKLOG_FILES`.
   * Alerting is based on the 10-minute average of these metrics. If the issue persists, notifications are resent every 30 minutes.
   */
  on_streaming_backlog_exceeded?: VariableOr<string[]>;
  /**
   * A list of email addresses to be notified when a run successfully completes. A run is considered to have completed successfully if it ends with a `TERMINATED` `life_cycle_state` and a `SUCCESS` result_state. If not specified on job creation, reset, or update, the list is empty, and notifications are not sent.
   */
  on_success?: VariableOr<string[]>;
}

export interface TaskNotificationSettings {
  /**
   * If true, do not send notifications to recipients specified in `on_start` for the retried runs and do not send notifications to recipients specified in `on_failure` until the last retry of the run.
   */
  alert_on_last_attempt?: VariableOr<boolean>;
  /**
   * If true, do not send notifications to recipients specified in `on_failure` if the run is canceled.
   */
  no_alert_for_canceled_runs?: VariableOr<boolean>;
  /**
   * If true, do not send notifications to recipients specified in `on_failure` if the run is skipped.
   */
  no_alert_for_skipped_runs?: VariableOr<boolean>;
}

/**
 * task retry mode of the continuous job
 * * NEVER: The failed task will not be retried.
 * * ON_FAILURE: Retry a failed task if at least one other task in the job is still running its first attempt.
 * When this condition is no longer met or the retry limit is reached, the job run is cancelled and a new run is started.
 */
export type TaskRetryMode =
  | "NEVER"
  | "ON_FAILURE";

export interface TriggerSettings {
  /**
   * File arrival trigger settings.
   */
  file_arrival?: VariableOr<FileArrivalTriggerConfiguration>;
  /**
   * Whether this trigger is paused or not.
   */
  pause_status?: VariableOr<PauseStatus>;
  /**
   * Periodic trigger settings.
   */
  periodic?: VariableOr<PeriodicTriggerConfiguration>;
  /**
   * Old table trigger settings name. Deprecated in favor of `table_update`.
   * @deprecated
   */
  table?: VariableOr<TableUpdateTriggerConfiguration>;
  table_update?: VariableOr<TableUpdateTriggerConfiguration>;
}

export interface Webhook {
  id: VariableOr<string>;
}

export interface WebhookNotifications {
  /**
   * An optional list of system notification IDs to call when the duration of a run exceeds the threshold specified for the `RUN_DURATION_SECONDS` metric in the `health` field. A maximum of 3 destinations can be specified for the `on_duration_warning_threshold_exceeded` property.
   */
  on_duration_warning_threshold_exceeded?: VariableOr<Webhook[]>;
  /**
   * An optional list of system notification IDs to call when the run fails. A maximum of 3 destinations can be specified for the `on_failure` property.
   */
  on_failure?: VariableOr<Webhook[]>;
  /**
   * An optional list of system notification IDs to call when the run starts. A maximum of 3 destinations can be specified for the `on_start` property.
   */
  on_start?: VariableOr<Webhook[]>;
  /**
   * An optional list of system notification IDs to call when any streaming backlog thresholds are exceeded for any stream.
   * Streaming backlog thresholds can be set in the `health` field using the following metrics: `STREAMING_BACKLOG_BYTES`, `STREAMING_BACKLOG_RECORDS`, `STREAMING_BACKLOG_SECONDS`, or `STREAMING_BACKLOG_FILES`.
   * Alerting is based on the 10-minute average of these metrics. If the issue persists, notifications are resent every 30 minutes.
   * A maximum of 3 destinations can be specified for the `on_streaming_backlog_exceeded` property.
   */
  on_streaming_backlog_exceeded?: VariableOr<Webhook[]>;
  /**
   * An optional list of system notification IDs to call when the run completes successfully. A maximum of 3 destinations can be specified for the `on_success` property.
   */
  on_success?: VariableOr<Webhook[]>;
}

export interface JobPermission {
  group_name?: VariableOr<string>;
  level: VariableOr<JobPermissionLevel>;
  service_principal_name?: VariableOr<string>;
  user_name?: VariableOr<string>;
}

export type JobPermissionLevel =
  | "CAN_MANAGE"
  | "CAN_MANAGE_RUN"
  | "CAN_VIEW"
  | "IS_OWNER";

export interface Lifecycle {
  /**
   * Lifecycle setting to prevent the resource from being destroyed.
   */
  prevent_destroy?: VariableOr<boolean>;
}
