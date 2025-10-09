/**
 * DatabaseInstance resource types for Databricks Asset Bundles
 *
 * Auto-generated from JSON Schema. Do not edit manually.
 */

import { Resource } from "../../src/core/resource.js";
import type { VariableOr } from "../../src/core/variable.js";

/**
 * A DatabaseInstance represents a logical Postgres instance, comprised of both compute and storage.
 */
export interface DatabaseInstanceParams {
  /**
   * The sku of the instance. Valid values are "CU_1", "CU_2", "CU_4", "CU_8".
   */
  capacity?: VariableOr<string>;
  /**
   * The refs of the child instances. This is only available if the instance is
   * parent instance.
   */
  child_instance_refs?: VariableOr<DatabaseInstanceRef[]>;
  /**
   * The timestamp when the instance was created.
   */
  creation_time?: VariableOr<string>;
  /**
   * The email of the creator of the instance.
   */
  creator?: VariableOr<string>;
  effective_capacity?: VariableOr<string>;
  /**
   * xref AIP-129. `enable_pg_native_login` is owned by the client, while `effective_enable_pg_native_login` is owned by the server.
   * `enable_pg_native_login` will only be set in Create/Update response messages if and only if the user provides the field via the request.
   * `effective_enable_pg_native_login` on the other hand will always bet set in all response messages (Create/Update/Get/List).
   */
  effective_enable_pg_native_login?: VariableOr<boolean>;
  /**
   * xref AIP-129. `enable_readable_secondaries` is owned by the client, while `effective_enable_readable_secondaries` is owned by the server.
   * `enable_readable_secondaries` will only be set in Create/Update response messages if and only if the user provides the field via the request.
   * `effective_enable_readable_secondaries` on the other hand will always bet set in all response messages (Create/Update/Get/List).
   */
  effective_enable_readable_secondaries?: VariableOr<boolean>;
  /**
   * xref AIP-129. `node_count` is owned by the client, while `effective_node_count` is owned by the server.
   * `node_count` will only be set in Create/Update response messages if and only if the user provides the field via the request.
   * `effective_node_count` on the other hand will always bet set in all response messages (Create/Update/Get/List).
   */
  effective_node_count?: VariableOr<number>;
  /**
   * xref AIP-129. `retention_window_in_days` is owned by the client, while `effective_retention_window_in_days` is owned by the server.
   * `retention_window_in_days` will only be set in Create/Update response messages if and only if the user provides the field via the request.
   * `effective_retention_window_in_days` on the other hand will always bet set in all response messages (Create/Update/Get/List).
   */
  effective_retention_window_in_days?: VariableOr<number>;
  /**
   * xref AIP-129. `stopped` is owned by the client, while `effective_stopped` is owned by the server.
   * `stopped` will only be set in Create/Update response messages if and only if the user provides the field via the request.
   * `effective_stopped` on the other hand will always bet set in all response messages (Create/Update/Get/List).
   */
  effective_stopped?: VariableOr<boolean>;
  /**
   * Whether to enable PG native password login on the instance. Defaults to false.
   */
  enable_pg_native_login?: VariableOr<boolean>;
  /**
   * Whether to enable secondaries to serve read-only traffic. Defaults to false.
   */
  enable_readable_secondaries?: VariableOr<boolean>;
  /**
   * Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
   */
  lifecycle?: VariableOr<Lifecycle>;
  /**
   * The name of the instance. This is the unique identifier for the instance.
   */
  name: VariableOr<string>;
  /**
   * The number of nodes in the instance, composed of 1 primary and 0 or more secondaries. Defaults to
   * 1 primary and 0 secondaries. This field is input only, see effective_node_count for the output.
   */
  node_count?: VariableOr<number>;
  /**
   * The ref of the parent instance. This is only available if the instance is
   * child instance.
   * Input: For specifying the parent instance to create a child instance. Optional.
   * Output: Only populated if provided as input to create a child instance.
   */
  parent_instance_ref?: VariableOr<DatabaseInstanceRef>;
  permissions?: VariableOr<DatabaseInstancePermission[]>;
  /**
   * The version of Postgres running on the instance.
   */
  pg_version?: VariableOr<string>;
  /**
   * The DNS endpoint to connect to the instance for read only access. This is only available if
   * enable_readable_secondaries is true.
   */
  read_only_dns?: VariableOr<string>;
  /**
   * The DNS endpoint to connect to the instance for read+write access.
   */
  read_write_dns?: VariableOr<string>;
  /**
   * The retention window for the instance. This is the time window in days
   * for which the historical data is retained. The default value is 7 days.
   * Valid values are 2 to 35 days.
   */
  retention_window_in_days?: VariableOr<number>;
  /**
   * The current state of the instance.
   */
  state?: VariableOr<DatabaseInstanceState>;
  /**
   * Whether to stop the instance. An input only param, see effective_stopped for the output.
   */
  stopped?: VariableOr<boolean>;
  /**
   * An immutable UUID identifier for the instance.
   */
  uid?: VariableOr<string>;
}

export class DatabaseInstance extends Resource<DatabaseInstanceParams> {
  constructor(name: string, params: DatabaseInstanceParams) {
    super(name, params, "database_instances");
  }
}

/**
 * DatabaseInstanceRef is a reference to a database instance. It is used in the
 * DatabaseInstance object to refer to the parent instance of an instance and
 * to refer the child instances of an instance.
 * To specify as a parent instance during creation of an instance,
 * the lsn and branch_time fields are optional. If not specified, the child
 * instance will be created from the latest lsn of the parent.
 * If both lsn and branch_time are specified, the lsn will be used to create
 * the child instance.
 */
export interface DatabaseInstanceRef {
  /**
   * Branch time of the ref database instance.
   * For a parent ref instance, this is the point in time on the parent instance from which the
   * instance was created.
   * For a child ref instance, this is the point in time on the instance from which the child
   * instance was created.
   * Input: For specifying the point in time to create a child instance. Optional.
   * Output: Only populated if provided as input to create a child instance.
   */
  branch_time?: VariableOr<string>;
  /**
   * For a parent ref instance, this is the LSN on the parent instance from which the
   * instance was created.
   * For a child ref instance, this is the LSN on the instance from which the child instance
   * was created.
   */
  effective_lsn?: VariableOr<string>;
  /**
   * User-specified WAL LSN of the ref database instance.
   * 
   * Input: For specifying the WAL LSN to create a child instance. Optional.
   * Output: Only populated if provided as input to create a child instance.
   */
  lsn?: VariableOr<string>;
  /**
   * Name of the ref database instance.
   */
  name?: VariableOr<string>;
  /**
   * Id of the ref database instance.
   */
  uid?: VariableOr<string>;
}

export type DatabaseInstanceState =
  | "STARTING"
  | "AVAILABLE"
  | "DELETING"
  | "STOPPED"
  | "UPDATING"
  | "FAILING_OVER";

export interface DatabaseInstancePermission {
  group_name?: VariableOr<string>;
  level: VariableOr<DatabaseInstancePermissionLevel>;
  service_principal_name?: VariableOr<string>;
  user_name?: VariableOr<string>;
}

export type DatabaseInstancePermissionLevel =
  | "CAN_CREATE"
  | "CAN_USE"
  | "CAN_MANAGE";

export interface Lifecycle {
  /**
   * Lifecycle setting to prevent the resource from being destroyed.
   */
  prevent_destroy?: VariableOr<boolean>;
}
