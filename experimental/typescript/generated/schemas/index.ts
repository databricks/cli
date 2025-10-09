/**
 * Schema resource types for Databricks Asset Bundles
 *
 * Auto-generated from JSON Schema. Do not edit manually.
 */

import { Resource } from "../../src/core/resource.js";
import type { VariableOr } from "../../src/core/variable.js";

export interface SchemaParams {
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

export class Schema extends Resource<SchemaParams> {
  constructor(name: string, params: SchemaParams) {
    super(name, params, "schemas");
  }
}

export interface Lifecycle {
  /**
   * Lifecycle setting to prevent the resource from being destroyed.
   */
  prevent_destroy?: VariableOr<boolean>;
}

export interface SchemaGrant {
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
