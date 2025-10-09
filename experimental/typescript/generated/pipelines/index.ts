/**
 * Pipeline resource types for Databricks Asset Bundles
 *
 * Auto-generated from JSON Schema. Do not edit manually.
 */

import { Resource } from "../../src/core/resource.js";
import type { VariableOr } from "../../src/core/variable.js";

export interface PipelineParams {
  /**
   * If false, deployment will fail if name conflicts with that of another pipeline.
   */
  allow_duplicate_names?: VariableOr<boolean>;
  /**
   * Budget policy of this pipeline.
   */
  budget_policy_id?: VariableOr<string>;
  /**
   * A catalog in Unity Catalog to publish data from this pipeline to. If `target` is specified, tables in this pipeline are published to a `target` schema inside `catalog` (for example, `catalog`.`target`.`table`). If `target` is not specified, no data is published to Unity Catalog.
   */
  catalog?: VariableOr<string>;
  /**
   * DLT Release Channel that specifies which version to use.
   */
  channel?: VariableOr<string>;
  /**
   * Cluster settings for this pipeline deployment.
   */
  clusters?: VariableOr<PipelineCluster[]>;
  /**
   * String-String configuration for this pipeline execution.
   */
  configuration?: VariableOr<Record<string, string>>;
  /**
   * Whether the pipeline is continuous or triggered. This replaces `trigger`.
   */
  continuous?: VariableOr<boolean>;
  /**
   * Whether the pipeline is in Development mode. Defaults to false.
   */
  development?: VariableOr<boolean>;
  /**
   * Pipeline product edition.
   */
  edition?: VariableOr<string>;
  /**
   * Environment specification for this pipeline used to install dependencies.
   */
  environment?: VariableOr<PipelinesEnvironment>;
  /**
   * Event log configuration for this pipeline
   */
  event_log?: VariableOr<EventLogSpec>;
  /**
   * Filters on which Pipeline packages to include in the deployed graph.
   */
  filters?: VariableOr<Filters>;
  /**
   * The definition of a gateway pipeline to support change data capture.
   */
  gateway_definition?: VariableOr<IngestionGatewayPipelineDefinition>;
  /**
   * Unique identifier for this pipeline.
   */
  id?: VariableOr<string>;
  /**
   * The configuration for a managed ingestion pipeline. These settings cannot be used with the 'libraries', 'schema', 'target', or 'catalog' settings.
   */
  ingestion_definition?: VariableOr<IngestionPipelineDefinition>;
  /**
   * Libraries or code needed by this deployment.
   */
  libraries?: VariableOr<PipelineLibrary[]>;
  /**
   * Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
   */
  lifecycle?: VariableOr<Lifecycle>;
  /**
   * Friendly identifier for this pipeline.
   */
  name?: VariableOr<string>;
  /**
   * List of notification settings for this pipeline.
   */
  notifications?: VariableOr<Notifications[]>;
  permissions?: VariableOr<PipelinePermission[]>;
  /**
   * Whether Photon is enabled for this pipeline.
   */
  photon?: VariableOr<boolean>;
  /**
   * Restart window of this pipeline.
   */
  restart_window?: VariableOr<RestartWindow>;
  /**
   * Root path for this pipeline.
   * This is used as the root directory when editing the pipeline in the Databricks user interface and it is
   * added to sys.path when executing Python sources during pipeline execution.
   */
  root_path?: VariableOr<string>;
  run_as?: VariableOr<RunAs>;
  /**
   * The default schema (database) where tables are read from or published to.
   */
  schema?: VariableOr<string>;
  /**
   * Whether serverless compute is enabled for this pipeline.
   */
  serverless?: VariableOr<boolean>;
  /**
   * DBFS root directory for storing checkpoints and tables.
   */
  storage?: VariableOr<string>;
  /**
   * A map of tags associated with the pipeline.
   * These are forwarded to the cluster as cluster tags, and are therefore subject to the same limitations.
   * A maximum of 25 tags can be added to the pipeline.
   */
  tags?: VariableOr<Record<string, string>>;
  /**
   * Target schema (database) to add tables in this pipeline to. Exactly one of `schema` or `target` must be specified. To publish to Unity Catalog, also specify `catalog`. This legacy field is deprecated for pipeline creation in favor of the `schema` field.
   * @deprecated
   */
  target?: VariableOr<string>;
  /**
   * Which pipeline trigger to use. Deprecated: Use `continuous` instead.
   * @deprecated
   */
  trigger?: VariableOr<PipelineTrigger>;
}

export class Pipeline extends Resource<PipelineParams> {
  constructor(name: string, params: PipelineParams) {
    super(name, params, "pipelines");
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
 * A storage location in DBFS
 */
export interface DbfsStorageInfo {
  /**
   * dbfs destination, e.g. `dbfs:/my/path`
   */
  destination: VariableOr<string>;
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
 * A storage location in Workspace Filesystem (WSFS)
 */
export interface WorkspaceStorageInfo {
  /**
   * wsfs destination, e.g. `workspace:/cluster-init-scripts/setup-datadog.sh`
   */
  destination: VariableOr<string>;
}

export interface CronTrigger {
  quartz_cron_schedule?: VariableOr<string>;
  timezone_id?: VariableOr<string>;
}

/**
 * Days of week in which the restart is allowed to happen (within a five-hour window starting at start_hour).
 * If not specified all days of the week will be used.
 */
export type DayOfWeek =
  | "MONDAY"
  | "TUESDAY"
  | "WEDNESDAY"
  | "THURSDAY"
  | "FRIDAY"
  | "SATURDAY"
  | "SUNDAY";

/**
 * Configurable event log parameters.
 */
export interface EventLogSpec {
  /**
   * The UC catalog the event log is published under.
   */
  catalog?: VariableOr<string>;
  /**
   * The name the event log is published to in UC.
   */
  name?: VariableOr<string>;
  /**
   * The UC schema the event log is published under.
   */
  schema?: VariableOr<string>;
}

export interface FileLibrary {
  /**
   * The absolute path of the source code.
   */
  path?: VariableOr<string>;
}

export interface Filters {
  /**
   * Paths to exclude.
   */
  exclude?: VariableOr<string[]>;
  /**
   * Paths to include.
   */
  include?: VariableOr<string[]>;
}

export interface IngestionConfig {
  /**
   * Select a specific source report.
   */
  report?: VariableOr<ReportSpec>;
  /**
   * Select all tables from a specific source schema.
   */
  schema?: VariableOr<SchemaSpec>;
  /**
   * Select a specific source table.
   */
  table?: VariableOr<TableSpec>;
}

export interface IngestionGatewayPipelineDefinition {
  /**
   * [Deprecated, use connection_name instead] Immutable. The Unity Catalog connection that this gateway pipeline uses to communicate with the source.
   * @deprecated
   */
  connection_id?: VariableOr<string>;
  /**
   * Immutable. The Unity Catalog connection that this gateway pipeline uses to communicate with the source.
   */
  connection_name: VariableOr<string>;
  /**
   * Required, Immutable. The name of the catalog for the gateway pipeline's storage location.
   */
  gateway_storage_catalog: VariableOr<string>;
  /**
   * Optional. The Unity Catalog-compatible name for the gateway storage location.
   * This is the destination to use for the data that is extracted by the gateway.
   * Delta Live Tables system will automatically create the storage location under the catalog and schema.
   */
  gateway_storage_name?: VariableOr<string>;
  /**
   * Required, Immutable. The name of the schema for the gateway pipelines's storage location.
   */
  gateway_storage_schema: VariableOr<string>;
}

export interface IngestionPipelineDefinition {
  /**
   * Immutable. The Unity Catalog connection that this ingestion pipeline uses to communicate with the source. This is used with connectors for applications like Salesforce, Workday, and so on.
   */
  connection_name?: VariableOr<string>;
  /**
   * Immutable. Identifier for the gateway that is used by this ingestion pipeline to communicate with the source database. This is used with connectors to databases like SQL Server.
   */
  ingestion_gateway_id?: VariableOr<string>;
  /**
   * Netsuite only configuration. When the field is set for a netsuite connector,
   * the jar stored in the field will be validated and added to the classpath of
   * pipeline's cluster.
   */
  netsuite_jar_path?: VariableOr<string>;
  /**
   * Required. Settings specifying tables to replicate and the destination for the replicated tables.
   */
  objects?: VariableOr<IngestionConfig[]>;
  /**
   * Top-level source configurations
   */
  source_configurations?: VariableOr<SourceConfig[]>;
  /**
   * The type of the foreign source.
   * The source type will be inferred from the source connection or ingestion gateway.
   * This field is output only and will be ignored if provided.
   */
  source_type?: VariableOr<IngestionSourceType>;
  /**
   * Configuration settings to control the ingestion of tables. These settings are applied to all tables in the pipeline.
   */
  table_configuration?: VariableOr<TableSpecificConfig>;
}

/**
 * Configurations that are only applicable for query-based ingestion connectors.
 */
export interface IngestionPipelineDefinitionTableSpecificConfigQueryBasedConnectorConfig {
  /**
   * The names of the monotonically increasing columns in the source table that are used to enable
   * the table to be read and ingested incrementally through structured streaming.
   * The columns are allowed to have repeated values but have to be non-decreasing.
   * If the source data is merged into the destination (e.g., using SCD Type 1 or Type 2), these
   * columns will implicitly define the `sequence_by` behavior. You can still explicitly set
   * `sequence_by` to override this default.
   */
  cursor_columns?: VariableOr<string[]>;
  /**
   * Specifies a SQL WHERE condition that specifies that the source row has been deleted.
   * This is sometimes referred to as "soft-deletes".
   * For example: "Operation = 'DELETE'" or "is_deleted = true".
   * This field is orthogonal to `hard_deletion_sync_interval_in_seconds`,
   * one for soft-deletes and the other for hard-deletes.
   * See also the hard_deletion_sync_min_interval_in_seconds field for
   * handling of "hard deletes" where the source rows are physically removed from the table.
   */
  deletion_condition?: VariableOr<string>;
  /**
   * Specifies the minimum interval (in seconds) between snapshots on primary keys
   * for detecting and synchronizing hard deletions—i.e., rows that have been
   * physically removed from the source table.
   * This interval acts as a lower bound. If ingestion runs less frequently than
   * this value, hard deletion synchronization will align with the actual ingestion
   * frequency instead of happening more often.
   * If not set, hard deletion synchronization via snapshots is disabled.
   * This field is mutable and can be updated without triggering a full snapshot.
   */
  hard_deletion_sync_min_interval_in_seconds?: VariableOr<number>;
}

export interface IngestionPipelineDefinitionWorkdayReportParameters {
  /**
   * (Optional) Marks the report as incremental.
   * This field is deprecated and should not be used. Use `parameters` instead. The incremental behavior is now
   * controlled by the `parameters` field.
   * @deprecated
   */
  incremental?: VariableOr<boolean>;
  /**
   * Parameters for the Workday report. Each key represents the parameter name (e.g., "start_date", "end_date"),
   * and the corresponding value is a SQL-like expression used to compute the parameter value at runtime.
   * Example:
   * {
   * "start_date": "{ coalesce(current_offset(), date(\"2025-02-01\")) }",
   * "end_date": "{ current_date() - INTERVAL 1 DAY }"
   * }
   */
  parameters?: VariableOr<Record<string, string>>;
  /**
   * (Optional) Additional custom parameters for Workday Report
   * This field is deprecated and should not be used. Use `parameters` instead.
   * @deprecated
   */
  report_parameters?: VariableOr<IngestionPipelineDefinitionWorkdayReportParametersQueryKeyValue[]>;
}

export interface IngestionPipelineDefinitionWorkdayReportParametersQueryKeyValue {
  /**
   * Key for the report parameter, can be a column name or other metadata
   */
  key?: VariableOr<string>;
  /**
   * Value for the report parameter.
   * Possible values it can take are these sql functions:
   * 1. coalesce(current_offset(), date("YYYY-MM-DD")) -> if current_offset() is null, then the passed date, else current_offset()
   * 2. current_date()
   * 3. date_sub(current_date(), x) -> subtract x (some non-negative integer) days from current date
   */
  value?: VariableOr<string>;
}

export type IngestionSourceType =
  | "MYSQL"
  | "POSTGRESQL"
  | "REDSHIFT"
  | "SQLDW"
  | "SQLSERVER"
  | "SALESFORCE"
  | "BIGQUERY"
  | "NETSUITE"
  | "WORKDAY_RAAS"
  | "GA4_RAW_DATA"
  | "SERVICENOW"
  | "MANAGED_POSTGRESQL"
  | "ORACLE"
  | "TERADATA"
  | "SHAREPOINT"
  | "DYNAMICS365"
  | "CONFLUENCE"
  | "META_MARKETING"
  | "FOREIGN_CATALOG";

export interface ManualTrigger {
}

export interface NotebookLibrary {
  /**
   * The absolute path of the source code.
   */
  path?: VariableOr<string>;
}

export interface Notifications {
  /**
   * A list of alerts that trigger the sending of notifications to the configured
   * destinations. The supported alerts are:
   * 
   * * `on-update-success`: A pipeline update completes successfully.
   * * `on-update-failure`: Each time a pipeline update fails.
   * * `on-update-fatal-failure`: A pipeline update fails with a non-retryable (fatal) error.
   * * `on-flow-failure`: A single data flow fails.
   */
  alerts?: VariableOr<string[]>;
  /**
   * A list of email addresses notified when a configured alert is triggered.
   */
  email_recipients?: VariableOr<string[]>;
}

export interface PathPattern {
  /**
   * The source code to include for pipelines
   */
  include?: VariableOr<string>;
}

export interface PipelineCluster {
  /**
   * Note: This field won't be persisted. Only API users will check this field.
   */
  apply_policy_default_values?: VariableOr<boolean>;
  /**
   * Parameters needed in order to automatically scale clusters up and down based on load.
   * Note: autoscaling works best with DB runtime versions 3.0 or later.
   */
  autoscale?: VariableOr<PipelineClusterAutoscale>;
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
   * Only dbfs destinations are supported. Only one destination can be specified
   * for one cluster. If the conf is given, the logs will be delivered to the destination every
   * `5 mins`. The destination of driver logs is `$destination/$clusterId/driver`, while
   * the destination of executor logs is `$destination/$clusterId/executor`.
   */
  cluster_log_conf?: VariableOr<ClusterLogConf>;
  /**
   * Additional tags for cluster resources. Databricks will tag all cluster resources (e.g., AWS
   * instances and EBS volumes) with these tags in addition to `default_tags`. Notes:
   * 
   * - Currently, Databricks allows at most 45 custom tags
   * 
   * - Clusters can only reuse cloud resources if the resources' tags are a subset of the cluster tags
   */
  custom_tags?: VariableOr<Record<string, string>>;
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
   */
  driver_node_type_id?: VariableOr<string>;
  /**
   * Whether to enable local disk encryption for the cluster.
   */
  enable_local_disk_encryption?: VariableOr<boolean>;
  /**
   * Attributes related to clusters running on Google Cloud Platform.
   * If not specified at cluster creation, a set of default values will be used.
   */
  gcp_attributes?: VariableOr<GcpAttributes>;
  /**
   * The configuration for storing init scripts. Any number of destinations can be specified. The scripts are executed sequentially in the order provided. If `cluster_log_conf` is specified, init script logs are sent to `<destination>/<cluster-ID>/init_scripts`.
   */
  init_scripts?: VariableOr<InitScriptInfo[]>;
  /**
   * The optional ID of the instance pool to which the cluster belongs.
   */
  instance_pool_id?: VariableOr<string>;
  /**
   * A label for the cluster specification, either `default` to configure the default cluster, or `maintenance` to configure the maintenance cluster. This field is optional. The default value is `default`.
   */
  label?: VariableOr<string>;
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
   * An object containing a set of optional, user-specified Spark configuration key-value pairs.
   * See :method:clusters/create for more details.
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
   * SSH public key contents that will be added to each Spark node in this cluster. The
   * corresponding private keys can be used to login with the user name `ubuntu` on port `2200`.
   * Up to 10 keys can be specified.
   */
  ssh_public_keys?: VariableOr<string[]>;
}

export interface PipelineClusterAutoscale {
  /**
   * The maximum number of workers to which the cluster can scale up when overloaded. `max_workers` must be strictly greater than `min_workers`.
   */
  max_workers: VariableOr<number>;
  /**
   * The minimum number of workers the cluster can scale down to when underutilized.
   * It is also the initial number of workers the cluster will have after creation.
   */
  min_workers: VariableOr<number>;
  /**
   * Databricks Enhanced Autoscaling optimizes cluster utilization by automatically
   * allocating cluster resources based on workload volume, with minimal impact to
   * the data processing latency of your pipelines. Enhanced Autoscaling is available
   * for `updates` clusters only. The legacy autoscaling feature is used for `maintenance`
   * clusters.
   */
  mode?: VariableOr<PipelineClusterAutoscaleMode>;
}

/**
 * Databricks Enhanced Autoscaling optimizes cluster utilization by automatically
 * allocating cluster resources based on workload volume, with minimal impact to
 * the data processing latency of your pipelines. Enhanced Autoscaling is available
 * for `updates` clusters only. The legacy autoscaling feature is used for `maintenance`
 * clusters.
 */
export type PipelineClusterAutoscaleMode =
  | "ENHANCED"
  | "LEGACY";

export interface PipelineLibrary {
  /**
   * The path to a file that defines a pipeline and is stored in the Databricks Repos.
   */
  file?: VariableOr<FileLibrary>;
  /**
   * The unified field to include source codes.
   * Each entry can be a notebook path, a file path, or a folder path that ends `/**`.
   * This field cannot be used together with `notebook` or `file`.
   */
  glob?: VariableOr<PathPattern>;
  /**
   * URI of the jar to be installed. Currently only DBFS is supported.
   */
  jar?: VariableOr<string>;
  /**
   * Specification of a maven library to be installed.
   */
  maven?: VariableOr<MavenLibrary>;
  /**
   * The path to a notebook that defines a pipeline and is stored in the Databricks workspace.
   */
  notebook?: VariableOr<NotebookLibrary>;
  /**
   * URI of the whl to be installed.
   * @deprecated
   */
  whl?: VariableOr<string>;
}

export interface PipelineTrigger {
  cron?: VariableOr<CronTrigger>;
  manual?: VariableOr<ManualTrigger>;
}

/**
 * The environment entity used to preserve serverless environment side panel, jobs' environment for non-notebook task, and DLT's environment for classic and serverless pipelines.
 * In this minimal environment spec, only pip dependencies are supported.
 */
export interface PipelinesEnvironment {
  /**
   * List of pip dependencies, as supported by the version of pip in this environment.
   * Each dependency is a pip requirement file line https://pip.pypa.io/en/stable/reference/requirements-file-format/
   * Allowed dependency could be <requirement specifier>, <archive url/path>, <local project path>(WSFS or Volumes in Databricks), <vcs project url>
   */
  dependencies?: VariableOr<string[]>;
}

/**
 * PG-specific catalog-level configuration parameters
 */
export interface PostgresCatalogConfig {
  /**
   * Optional. The Postgres slot configuration to use for logical replication
   */
  slot_config?: VariableOr<PostgresSlotConfig>;
}

/**
 * PostgresSlotConfig contains the configuration for a Postgres logical replication slot
 */
export interface PostgresSlotConfig {
  /**
   * The name of the publication to use for the Postgres source
   */
  publication_name?: VariableOr<string>;
  /**
   * The name of the logical replication slot to use for the Postgres source
   */
  slot_name?: VariableOr<string>;
}

export interface ReportSpec {
  /**
   * Required. Destination catalog to store table.
   */
  destination_catalog: VariableOr<string>;
  /**
   * Required. Destination schema to store table.
   */
  destination_schema: VariableOr<string>;
  /**
   * Required. Destination table name. The pipeline fails if a table with that name already exists.
   */
  destination_table?: VariableOr<string>;
  /**
   * Required. Report URL in the source system.
   */
  source_url: VariableOr<string>;
  /**
   * Configuration settings to control the ingestion of tables. These settings override the table_configuration defined in the IngestionPipelineDefinition object.
   */
  table_configuration?: VariableOr<TableSpecificConfig>;
}

export interface RestartWindow {
  /**
   * Days of week in which the restart is allowed to happen (within a five-hour window starting at start_hour).
   * If not specified all days of the week will be used.
   */
  days_of_week?: VariableOr<DayOfWeek[]>;
  /**
   * An integer between 0 and 23 denoting the start hour for the restart window in the 24-hour day.
   * Continuous pipeline restart is triggered only within a five-hour window starting at this hour.
   */
  start_hour: VariableOr<number>;
  /**
   * Time zone id of restart window. See https://docs.databricks.com/sql/language-manual/sql-ref-syntax-aux-conf-mgmt-set-timezone.html for details.
   * If not specified, UTC will be used.
   */
  time_zone_id?: VariableOr<string>;
}

/**
 * Write-only setting, available only in Create/Update calls. Specifies the user or service principal that the pipeline runs as. If not specified, the pipeline runs as the user who created the pipeline.
 * 
 * Only `user_name` or `service_principal_name` can be specified. If both are specified, an error is thrown.
 */
export interface RunAs {
  /**
   * Application ID of an active service principal. Setting this field requires the `servicePrincipal/user` role.
   */
  service_principal_name?: VariableOr<string>;
  /**
   * The email of an active workspace user. Users can only set this field to their own email.
   */
  user_name?: VariableOr<string>;
}

export interface SchemaSpec {
  /**
   * Required. Destination catalog to store tables.
   */
  destination_catalog: VariableOr<string>;
  /**
   * Required. Destination schema to store tables in. Tables with the same name as the source tables are created in this destination schema. The pipeline fails If a table with the same name already exists.
   */
  destination_schema: VariableOr<string>;
  /**
   * The source catalog name. Might be optional depending on the type of source.
   */
  source_catalog?: VariableOr<string>;
  /**
   * Required. Schema name in the source database.
   */
  source_schema: VariableOr<string>;
  /**
   * Configuration settings to control the ingestion of tables. These settings are applied to all tables in this schema and override the table_configuration defined in the IngestionPipelineDefinition object.
   */
  table_configuration?: VariableOr<TableSpecificConfig>;
}

/**
 * SourceCatalogConfig contains catalog-level custom configuration parameters for each source
 */
export interface SourceCatalogConfig {
  /**
   * Postgres-specific catalog-level configuration parameters
   */
  postgres?: VariableOr<PostgresCatalogConfig>;
  /**
   * Source catalog name
   */
  source_catalog?: VariableOr<string>;
}

export interface SourceConfig {
  /**
   * Catalog-level source configuration parameters
   */
  catalog?: VariableOr<SourceCatalogConfig>;
}

export interface TableSpec {
  /**
   * Required. Destination catalog to store table.
   */
  destination_catalog: VariableOr<string>;
  /**
   * Required. Destination schema to store table.
   */
  destination_schema: VariableOr<string>;
  /**
   * Optional. Destination table name. The pipeline fails if a table with that name already exists. If not set, the source table name is used.
   */
  destination_table?: VariableOr<string>;
  /**
   * Source catalog name. Might be optional depending on the type of source.
   */
  source_catalog?: VariableOr<string>;
  /**
   * Schema name in the source database. Might be optional depending on the type of source.
   */
  source_schema?: VariableOr<string>;
  /**
   * Required. Table name in the source database.
   */
  source_table: VariableOr<string>;
  /**
   * Configuration settings to control the ingestion of tables. These settings override the table_configuration defined in the IngestionPipelineDefinition object and the SchemaSpec.
   */
  table_configuration?: VariableOr<TableSpecificConfig>;
}

export interface TableSpecificConfig {
  /**
   * A list of column names to be excluded for the ingestion.
   * When not specified, include_columns fully controls what columns to be ingested.
   * When specified, all other columns including future ones will be automatically included for ingestion.
   * This field in mutually exclusive with `include_columns`.
   */
  exclude_columns?: VariableOr<string[]>;
  /**
   * A list of column names to be included for the ingestion.
   * When not specified, all columns except ones in exclude_columns will be included. Future
   * columns will be automatically included.
   * When specified, all other future columns will be automatically excluded from ingestion.
   * This field in mutually exclusive with `exclude_columns`.
   */
  include_columns?: VariableOr<string[]>;
  /**
   * The primary key of the table used to apply changes.
   */
  primary_keys?: VariableOr<string[]>;
  /**
   * Configurations that are only applicable for query-based ingestion connectors.
   */
  query_based_connector_config?: VariableOr<IngestionPipelineDefinitionTableSpecificConfigQueryBasedConnectorConfig>;
  /**
   * If true, formula fields defined in the table are included in the ingestion. This setting is only valid for the Salesforce connector
   */
  salesforce_include_formula_fields?: VariableOr<boolean>;
  /**
   * The SCD type to use to ingest the table.
   */
  scd_type?: VariableOr<TableSpecificConfigScdType>;
  /**
   * The column names specifying the logical order of events in the source data. Delta Live Tables uses this sequencing to handle change events that arrive out of order.
   */
  sequence_by?: VariableOr<string[]>;
  /**
   * (Optional) Additional custom parameters for Workday Report
   */
  workday_report_parameters?: VariableOr<IngestionPipelineDefinitionWorkdayReportParameters>;
}

/**
 * The SCD type to use to ingest the table.
 */
export type TableSpecificConfigScdType =
  | "SCD_TYPE_1"
  | "SCD_TYPE_2"
  | "APPEND_ONLY";

export interface Lifecycle {
  /**
   * Lifecycle setting to prevent the resource from being destroyed.
   */
  prevent_destroy?: VariableOr<boolean>;
}

export interface PipelinePermission {
  group_name?: VariableOr<string>;
  level: VariableOr<PipelinePermissionLevel>;
  service_principal_name?: VariableOr<string>;
  user_name?: VariableOr<string>;
}

export type PipelinePermissionLevel =
  | "CAN_MANAGE"
  | "IS_OWNER"
  | "CAN_RUN"
  | "CAN_VIEW";
