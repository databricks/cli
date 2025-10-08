/**
 * RegisteredModel resource types for Databricks Asset Bundles
 *
 * Auto-generated from JSON Schema. Do not edit manually.
 */

import type { Resource } from "../../src/core/resource.js";
import type { VariableOr } from "../../src/core/variable.js";

export interface RegisteredModel extends Resource {
  /**
   * The name of the catalog where the schema and the registered model reside
   */
  catalog_name: VariableOr<string>;
  /**
   * The comment attached to the registered model
   */
  comment?: VariableOr<string>;
  grants?: VariableOr<Grant[]>;
  /**
   * Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
   */
  lifecycle?: VariableOr<Lifecycle>;
  /**
   * The name of the registered model
   */
  name: VariableOr<string>;
  /**
   * The name of the schema where the registered model resides
   */
  schema_name: VariableOr<string>;
  /**
   * The storage location on the cloud under which model version data files are stored
   */
  storage_location?: VariableOr<string>;
}

export interface Grant extends Resource {
  /**
   * The name of the principal that will be granted privileges
   */
  principal: VariableOr<string>;
  /**
   * The privileges to grant to the specified entity
   */
  privileges: VariableOr<string[]>;
}

export interface Lifecycle extends Resource {
  /**
   * Lifecycle setting to prevent the resource from being destroyed.
   */
  prevent_destroy?: VariableOr<boolean>;
}

/**
 * Helper function to create a RegisteredModel with type safety
 */
export function createRegisteredModel(config: RegisteredModel): RegisteredModel {
  return config;
}
