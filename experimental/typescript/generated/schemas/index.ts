/**
 * Schema resource types for Databricks Asset Bundles
 *
 * Auto-generated from JSON Schema. Do not edit manually.
 */

import type { Resource } from "../../src/core/resource.js";
import type { VariableOr } from "../../src/core/variable.js";

export interface Schema extends Resource {
  /**
   * Name of parent catalog.
   */
  catalog_name: VariableOr<string>;
  /**
   * User-provided free-form text description.
   */
  comment?: VariableOr<string>;
  grants?: VariableOr<SchemaGrant[]>;
  /**
   * Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
   */
  lifecycle?: VariableOr<Lifecycle>;
  /**
   * Name of schema, relative to parent catalog.
   */
  name: VariableOr<string>;
  properties?: VariableOr<Record<string, string>>;
  /**
   * Storage root URL for managed tables within schema.
   */
  storage_root?: VariableOr<string>;
}

export interface Lifecycle extends Resource {
  /**
   * Lifecycle setting to prevent the resource from being destroyed.
   */
  prevent_destroy?: VariableOr<boolean>;
}

export interface SchemaGrant extends Resource {
  principal: VariableOr<string>;
  privileges: VariableOr<SchemaGrantPrivilege[]>;
}

export type SchemaGrantPrivilege =
  | "ALL_PRIVILEGES"
  | "APPLY_TAG"
  | "CREATE_FUNCTION"
  | "CREATE_TABLE"
  | "CREATE_VOLUME"
  | "MANAGE"
  | "USE_SCHEMA"
  | "EXECUTE"
  | "MODIFY"
  | "REFRESH"
  | "SELECT"
  | "READ_VOLUME"
  | "WRITE_VOLUME";

/**
 * Helper function to create a Schema with type safety
 */
export function createSchema(config: Schema): Schema {
  return config;
}
