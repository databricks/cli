/**
 * MlflowExperiment resource types for Databricks Asset Bundles
 *
 * Auto-generated from JSON Schema. Do not edit manually.
 */

import type { Resource } from "../../src/core/resource.js";
import type { VariableOr } from "../../src/core/variable.js";

/**
 * An experiment and its metadata.
 */
export interface MlflowExperiment extends Resource {
  /**
   * Location where artifacts for the experiment are stored.
   */
  artifact_location?: VariableOr<string>;
  /**
   * Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
   */
  lifecycle?: VariableOr<Lifecycle>;
  /**
   * Human readable name that identifies the experiment.
   */
  name: VariableOr<string>;
  permissions?: VariableOr<MlflowExperimentPermission[]>;
  /**
   * Tags: Additional metadata key-value pairs.
   */
  tags?: VariableOr<ExperimentTag[]>;
}

/**
 * A tag for an experiment.
 */
export interface ExperimentTag extends Resource {
  /**
   * The tag key.
   */
  key?: VariableOr<string>;
  /**
   * The tag value.
   */
  value?: VariableOr<string>;
}

export interface Lifecycle extends Resource {
  /**
   * Lifecycle setting to prevent the resource from being destroyed.
   */
  prevent_destroy?: VariableOr<boolean>;
}

export interface MlflowExperimentPermission extends Resource {
  group_name?: VariableOr<string>;
  level: VariableOr<MlflowExperimentPermissionLevel>;
  service_principal_name?: VariableOr<string>;
  user_name?: VariableOr<string>;
}

export type MlflowExperimentPermissionLevel =
  | "CAN_MANAGE"
  | "CAN_EDIT"
  | "CAN_READ";

/**
 * Helper function to create a MlflowExperiment with type safety
 */
export function createMlflowExperiment(config: MlflowExperiment): MlflowExperiment {
  return config;
}
