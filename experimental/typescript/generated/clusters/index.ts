/**
 * Cluster resource types for Databricks Asset Bundles
 *
 * Auto-generated from JSON Schema. Do not edit manually.
 */

import { Resource } from "../../src/core/resource.js";
import type { VariableOr } from "../../src/core/variable.js";

/**
 * Contains a snapshot of the latest user specified settings that were used to create/edit the cluster.
 */
export interface ClusterParams {
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
   * Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
   */
  lifecycle?: VariableOr<Lifecycle>;
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
  permissions?: VariableOr<ClusterPermission[]>;
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

export class Cluster extends Resource<ClusterParams> {
  constructor(params: ClusterParams) {
    super(params, "clusters");
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

export interface ClusterPermission {
  group_name?: VariableOr<string>;
  level: VariableOr<ClusterPermissionLevel>;
  service_principal_name?: VariableOr<string>;
  user_name?: VariableOr<string>;
}

export type ClusterPermissionLevel =
  | "CAN_MANAGE"
  | "CAN_RESTART"
  | "CAN_ATTACH_TO";

export interface Lifecycle {
  /**
   * Lifecycle setting to prevent the resource from being destroyed.
   */
  prevent_destroy?: VariableOr<boolean>;
}
