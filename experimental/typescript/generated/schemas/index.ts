/**
 * Schema resource types for Databricks Asset Bundles
 *
 * These are placeholder types that will be replaced by auto-generated types
 * from OpenAPI specifications in the future.
 */

import type { Resource } from "../../src/core/resource.js";
import type { VariableOr } from "../../src/core/variable.js";

/**
 * Represents a Databricks Unity Catalog Schema resource
 */
export interface Schema extends Resource {
  /**
   * Name of the schema
   */
  name: VariableOr<string>;

  /**
   * Name of the parent catalog
   */
  catalog_name: VariableOr<string>;

  /**
   * User-provided comment/description
   */
  comment?: VariableOr<string>;

  /**
   * Properties for the schema
   */
  properties?: VariableOr<Record<string, string>>;

  /**
   * Storage root for managed tables
   */
  storage_root?: VariableOr<string>;

  /**
   * Owner of the schema
   */
  owner?: VariableOr<string>;
}

/**
 * Helper function to create a Schema with type safety
 */
export function createSchema(config: Schema): Schema {
  return config;
}
