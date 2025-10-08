/**
 * App resource types for Databricks Asset Bundles
 *
 * Auto-generated from JSON Schema. Do not edit manually.
 */

import type { Resource } from "../../src/core/resource.js";
import type { VariableOr } from "../../src/core/variable.js";

export interface App extends Resource {
  /**
   * The active deployment of the app. A deployment is considered active when it has been deployed
   * to the app compute.
   */
  active_deployment?: VariableOr<AppDeployment>;
  app_status?: VariableOr<ApplicationStatus>;
  budget_policy_id?: VariableOr<string>;
  compute_status?: VariableOr<ComputeStatus>;
  /**
   * The creation time of the app. Formatted timestamp in ISO 6801.
   */
  create_time?: VariableOr<string>;
  /**
   * The email of the user that created the app.
   */
  creator?: VariableOr<string>;
  /**
   * The default workspace file system path of the source code from which app deployment are
   * created. This field tracks the workspace source code path of the last active deployment.
   */
  default_source_code_path?: VariableOr<string>;
  /**
   * The description of the app.
   */
  description?: VariableOr<string>;
  effective_budget_policy_id?: VariableOr<string>;
  /**
   * The effective api scopes granted to the user access token.
   */
  effective_user_api_scopes?: VariableOr<string[]>;
  /**
   * The unique identifier of the app.
   */
  id?: VariableOr<string>;
  /**
   * Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
   */
  lifecycle?: VariableOr<Lifecycle>;
  /**
   * The name of the app. The name must contain only lowercase alphanumeric characters and hyphens.
   * It must be unique within the workspace.
   */
  name: VariableOr<string>;
  oauth2_app_client_id?: VariableOr<string>;
  oauth2_app_integration_id?: VariableOr<string>;
  /**
   * The pending deployment of the app. A deployment is considered pending when it is being prepared
   * for deployment to the app compute.
   */
  pending_deployment?: VariableOr<AppDeployment>;
  permissions?: VariableOr<AppPermission[]>;
  /**
   * Resources for the app.
   */
  resources?: VariableOr<AppResource[]>;
  service_principal_client_id?: VariableOr<string>;
  service_principal_id?: VariableOr<number>;
  service_principal_name?: VariableOr<string>;
  source_code_path: VariableOr<string>;
  /**
   * The update time of the app. Formatted timestamp in ISO 6801.
   */
  update_time?: VariableOr<string>;
  /**
   * The email of the user that last updated the app.
   */
  updater?: VariableOr<string>;
  /**
   * The URL of the app once it is deployed.
   */
  url?: VariableOr<string>;
  user_api_scopes?: VariableOr<string[]>;
}

export interface AppDeployment extends Resource {
  create_time?: VariableOr<string>;
  creator?: VariableOr<string>;
  deployment_artifacts?: VariableOr<AppDeploymentArtifacts>;
  deployment_id?: VariableOr<string>;
  mode?: VariableOr<AppDeploymentMode>;
  source_code_path?: VariableOr<string>;
  status?: VariableOr<AppDeploymentStatus>;
  update_time?: VariableOr<string>;
}

export interface AppDeploymentArtifacts extends Resource {
  source_code_path?: VariableOr<string>;
}

export type AppDeploymentMode =
  | "SNAPSHOT"
  | "AUTO_SYNC";

export type AppDeploymentState =
  | "SUCCEEDED"
  | "FAILED"
  | "IN_PROGRESS"
  | "CANCELLED";

export interface AppDeploymentStatus extends Resource {
  message?: VariableOr<string>;
  state?: VariableOr<AppDeploymentState>;
}

export interface AppResource extends Resource {
  database?: VariableOr<AppResourceDatabase>;
  /**
   * Description of the App Resource.
   */
  description?: VariableOr<string>;
  job?: VariableOr<AppResourceJob>;
  /**
   * Name of the App Resource.
   */
  name: VariableOr<string>;
  secret?: VariableOr<AppResourceSecret>;
  serving_endpoint?: VariableOr<AppResourceServingEndpoint>;
  sql_warehouse?: VariableOr<AppResourceSqlWarehouse>;
  uc_securable?: VariableOr<AppResourceUcSecurable>;
}

export interface AppResourceDatabase extends Resource {
  database_name: VariableOr<string>;
  instance_name: VariableOr<string>;
  permission: VariableOr<AppResourceDatabaseDatabasePermission>;
}

export type AppResourceDatabaseDatabasePermission =
  | "CAN_CONNECT_AND_CREATE";

export interface AppResourceJob extends Resource {
  id: VariableOr<string>;
  permission: VariableOr<AppResourceJobJobPermission>;
}

export type AppResourceJobJobPermission =
  | "CAN_MANAGE"
  | "IS_OWNER"
  | "CAN_MANAGE_RUN"
  | "CAN_VIEW";

export interface AppResourceSecret extends Resource {
  key: VariableOr<string>;
  permission: VariableOr<AppResourceSecretSecretPermission>;
  scope: VariableOr<string>;
}

/**
 * Permission to grant on the secret scope. Supported permissions are: "READ", "WRITE", "MANAGE".
 */
export type AppResourceSecretSecretPermission =
  | "READ"
  | "WRITE"
  | "MANAGE";

export interface AppResourceServingEndpoint extends Resource {
  name: VariableOr<string>;
  permission: VariableOr<AppResourceServingEndpointServingEndpointPermission>;
}

export type AppResourceServingEndpointServingEndpointPermission =
  | "CAN_MANAGE"
  | "CAN_QUERY"
  | "CAN_VIEW";

export interface AppResourceSqlWarehouse extends Resource {
  id: VariableOr<string>;
  permission: VariableOr<AppResourceSqlWarehouseSqlWarehousePermission>;
}

export type AppResourceSqlWarehouseSqlWarehousePermission =
  | "CAN_MANAGE"
  | "CAN_USE"
  | "IS_OWNER";

export interface AppResourceUcSecurable extends Resource {
  permission: VariableOr<AppResourceUcSecurableUcSecurablePermission>;
  securable_full_name: VariableOr<string>;
  securable_type: VariableOr<AppResourceUcSecurableUcSecurableType>;
}

export type AppResourceUcSecurableUcSecurablePermission =
  | "READ_VOLUME"
  | "WRITE_VOLUME";

export type AppResourceUcSecurableUcSecurableType =
  | "VOLUME";

export type ApplicationState =
  | "DEPLOYING"
  | "RUNNING"
  | "CRASHED"
  | "UNAVAILABLE";

export interface ApplicationStatus extends Resource {
  message?: VariableOr<string>;
  state?: VariableOr<ApplicationState>;
}

export type ComputeState =
  | "ERROR"
  | "DELETING"
  | "STARTING"
  | "STOPPING"
  | "UPDATING"
  | "STOPPED"
  | "ACTIVE";

export interface ComputeStatus extends Resource {
  message?: VariableOr<string>;
  /**
   * State of the app compute.
   */
  state?: VariableOr<ComputeState>;
}

export interface AppPermission extends Resource {
  group_name?: VariableOr<string>;
  level: VariableOr<AppPermissionLevel>;
  service_principal_name?: VariableOr<string>;
  user_name?: VariableOr<string>;
}

export type AppPermissionLevel =
  | "CAN_MANAGE"
  | "CAN_USE";

export interface Lifecycle extends Resource {
  /**
   * Lifecycle setting to prevent the resource from being destroyed.
   */
  prevent_destroy?: VariableOr<boolean>;
}

/**
 * Helper function to create a App with type safety
 */
export function createApp(config: App): App {
  return config;
}
