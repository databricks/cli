/**
 * MlflowModel resource types for Databricks Asset Bundles
 *
 * Auto-generated from JSON Schema. Do not edit manually.
 */

import { Resource } from "../../src/core/resource.js";
import type { VariableOr } from "../../src/core/variable.js";

export interface MlflowModelParams {
  /**
   * Optional description for registered model.
   */
  description?: VariableOr<string>;
  /**
   * Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
   */
  lifecycle?: VariableOr<Lifecycle>;
  /**
   * Register models under this name
   */
  name: VariableOr<string>;
  permissions?: VariableOr<MlflowModelPermission[]>;
  /**
   * Additional metadata for registered model.
   */
  tags?: VariableOr<ModelTag[]>;
}

export class MlflowModel extends Resource<MlflowModelParams> {
  constructor(params: MlflowModelParams) {
    super(params, "mlflow_models");
  }
}

/**
 * Tag for a registered model
 */
export interface ModelTag {
  /**
   * The tag key.
   */
  key?: VariableOr<string>;
  /**
   * The tag value.
   */
  value?: VariableOr<string>;
}

export interface Lifecycle {
  /**
   * Lifecycle setting to prevent the resource from being destroyed.
   */
  prevent_destroy?: VariableOr<boolean>;
}

export interface MlflowModelPermission {
  group_name?: VariableOr<string>;
  level: VariableOr<MlflowModelPermissionLevel>;
  service_principal_name?: VariableOr<string>;
  user_name?: VariableOr<string>;
}

export type MlflowModelPermissionLevel =
  | "CAN_EDIT"
  | "CAN_MANAGE"
  | "CAN_MANAGE_STAGING_VERSIONS"
  | "CAN_MANAGE_PRODUCTION_VERSIONS"
  | "CAN_READ";
