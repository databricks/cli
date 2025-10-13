/**
 * SqlWarehouse resource types for Databricks Asset Bundles
 *
 * Auto-generated from JSON Schema. Do not edit manually.
 */

import { Resource } from "../../src/core/resource.js";
import type { VariableOr } from "../../src/core/variable.js";

export interface SqlWarehouseParams {
  /**
   * The amount of time in minutes that a SQL warehouse must be idle (i.e., no RUNNING queries) before
   * it is automatically stopped.
   * 
   * Supported values:
   *   - Must be >= 0 mins for serverless warehouses
   *   - Must be == 0 or >= 10 mins for non-serverless warehouses
   *  - 0 indicates no autostop.
   * 
   * Defaults to 120 mins
   */
  auto_stop_mins?: VariableOr<number>;
  /**
   * Channel Details
   */
  channel?: VariableOr<Channel>;
  /**
   * Size of the clusters allocated for this warehouse.
   * Increasing the size of a spark cluster allows you to run larger queries on it.
   * If you want to increase the number of concurrent queries, please tune max_num_clusters.
   * 
   * Supported values:
   *   - 2X-Small
   *   - X-Small
   *   - Small
   *   - Medium
   *   - Large
   *   - X-Large
   *   - 2X-Large
   *   - 3X-Large
   *   - 4X-Large
   * 
   */
  cluster_size?: VariableOr<string>;
  /**
   * warehouse creator name
   */
  creator_name?: VariableOr<string>;
  /**
   * Configures whether the warehouse should use Photon optimized clusters.
   * 
   * Defaults to true.
   */
  enable_photon?: VariableOr<boolean>;
  /**
   * Configures whether the warehouse should use serverless compute
   */
  enable_serverless_compute?: VariableOr<boolean>;
  /**
   * Deprecated. Instance profile used to pass IAM role to the cluster
   * @deprecated
   */
  instance_profile_arn?: VariableOr<string>;
  /**
   * Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
   */
  lifecycle?: VariableOr<Lifecycle>;
  /**
   * Maximum number of clusters that the autoscaler will create to handle concurrent queries.
   * 
   * Supported values:
   *   - Must be >= min_num_clusters
   *   - Must be <= 30.
   * 
   * Defaults to min_clusters if unset.
   */
  max_num_clusters?: VariableOr<number>;
  /**
   * Minimum number of available clusters that will be maintained for this SQL warehouse.
   * Increasing this will ensure that a larger number of clusters are always running and therefore may reduce
   * the cold start time for new queries. This is similar to reserved vs. revocable cores in a resource manager.
   * 
   * Supported values:
   *   - Must be > 0
   *   - Must be <= min(max_num_clusters, 30)
   * 
   * Defaults to 1
   */
  min_num_clusters?: VariableOr<number>;
  /**
   * Logical name for the cluster.
   * 
   * Supported values:
   *   - Must be unique within an org.
   *   - Must be less than 100 characters.
   */
  name?: VariableOr<string>;
  permissions?: VariableOr<SqlWarehousePermission[]>;
  spot_instance_policy?: VariableOr<SpotInstancePolicy>;
  /**
   * A set of key-value pairs that will be tagged on all resources (e.g., AWS instances and EBS volumes) associated
   * with this SQL warehouse.
   * 
   * Supported values:
   *   - Number of tags < 45.
   */
  tags?: VariableOr<EndpointTags>;
  warehouse_type?: VariableOr<CreateWarehouseRequestWarehouseType>;
}

export class SqlWarehouse extends Resource<SqlWarehouseParams> {
  constructor(name: string, params: SqlWarehouseParams) {
    super(name, params, "sql_warehouses");
  }
}

export interface Lifecycle {
  /**
   * Lifecycle setting to prevent the resource from being destroyed.
   */
  prevent_destroy?: VariableOr<boolean>;
}

export interface SqlWarehousePermission {
  group_name?: VariableOr<string>;
  level: VariableOr<SqlWarehousePermissionLevel>;
  service_principal_name?: VariableOr<string>;
  user_name?: VariableOr<string>;
}

export type SqlWarehousePermissionLevel =
  | "CAN_MANAGE"
  | "CAN_USE"
  | "CAN_MONITOR"
  | "CAN_VIEW";

/**
 * Configures the channel name and DBSQL version of the warehouse. CHANNEL_NAME_CUSTOM should be chosen only when `dbsql_version` is specified.
 */
export interface Channel {
  dbsql_version?: VariableOr<string>;
  name?: VariableOr<ChannelName>;
}

export type ChannelName =
  | "CHANNEL_NAME_PREVIEW"
  | "CHANNEL_NAME_CURRENT"
  | "CHANNEL_NAME_PREVIOUS"
  | "CHANNEL_NAME_CUSTOM";

/**
 * Warehouse type: `PRO` or `CLASSIC`. If you want to use serverless compute, you must set to `PRO` and also set the field `enable_serverless_compute` to `true`.
 */
export type CreateWarehouseRequestWarehouseType =
  | "TYPE_UNSPECIFIED"
  | "CLASSIC"
  | "PRO";

export interface EndpointTagPair {
  key?: VariableOr<string>;
  value?: VariableOr<string>;
}

export interface EndpointTags {
  custom_tags?: VariableOr<EndpointTagPair[]>;
}

/**
 * Configurations whether the warehouse should use spot instances.
 */
export type SpotInstancePolicy =
  | "POLICY_UNSPECIFIED"
  | "COST_OPTIMIZED"
  | "RELIABILITY_OPTIMIZED";
