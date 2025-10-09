/**
 * Dashboard resource types for Databricks Asset Bundles
 *
 * Auto-generated from JSON Schema. Do not edit manually.
 */

import { Resource } from "../../src/core/resource.js";
import type { VariableOr } from "../../src/core/variable.js";

export interface DashboardParams {
  /**
   * The timestamp of when the dashboard was created.
   */
  create_time?: VariableOr<string>;
  /**
   * UUID identifying the dashboard.
   */
  dashboard_id?: VariableOr<string>;
  /**
   * The display name of the dashboard.
   */
  display_name?: VariableOr<string>;
  embed_credentials?: VariableOr<boolean>;
  /**
   * The etag for the dashboard. Can be optionally provided on updates to ensure that the dashboard
   * has not been modified since the last read.
   * This field is excluded in List Dashboards responses.
   */
  etag?: VariableOr<string>;
  file_path?: VariableOr<string>;
  /**
   * Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
   */
  lifecycle?: VariableOr<Lifecycle>;
  /**
   * The state of the dashboard resource. Used for tracking trashed status.
   */
  lifecycle_state?: VariableOr<LifecycleState>;
  /**
   * The workspace path of the folder containing the dashboard. Includes leading slash and no
   * trailing slash.
   * This field is excluded in List Dashboards responses.
   */
  parent_path?: VariableOr<string>;
  /**
   * The workspace path of the dashboard asset, including the file name.
   * Exported dashboards always have the file extension `.lvdash.json`.
   * This field is excluded in List Dashboards responses.
   */
  path?: VariableOr<string>;
  permissions?: VariableOr<DashboardPermission[]>;
  /**
   * The contents of the dashboard in serialized string form.
   * This field is excluded in List Dashboards responses.
   * Use the [get dashboard API](https://docs.databricks.com/api/workspace/lakeview/get)
   * to retrieve an example response, which includes the `serialized_dashboard` field.
   * This field provides the structure of the JSON string that represents the dashboard's
   * layout and components.
   */
  serialized_dashboard?: VariableOr<any>;
  /**
   * The timestamp of when the dashboard was last updated by the user.
   * This field is excluded in List Dashboards responses.
   */
  update_time?: VariableOr<string>;
  /**
   * The warehouse ID used to run the dashboard.
   */
  warehouse_id?: VariableOr<string>;
}

export class Dashboard extends Resource<DashboardParams> {
  constructor(name: string, params: DashboardParams) {
    super(name, params, "dashboards");
  }
}

export type LifecycleState =
  | "ACTIVE"
  | "TRASHED";

export interface DashboardPermission {
  group_name?: VariableOr<string>;
  level: VariableOr<DashboardPermissionLevel>;
  service_principal_name?: VariableOr<string>;
  user_name?: VariableOr<string>;
}

export type DashboardPermissionLevel =
  | "CAN_READ"
  | "CAN_RUN"
  | "CAN_EDIT"
  | "CAN_MANAGE";

export interface Lifecycle {
  /**
   * Lifecycle setting to prevent the resource from being destroyed.
   */
  prevent_destroy?: VariableOr<boolean>;
}
