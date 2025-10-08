/**
 * Volume resource types for Databricks Asset Bundles
 *
 * These are placeholder types that will be replaced by auto-generated types
 * from OpenAPI specifications in the future.
 */

import type { Resource } from "../../src/core/resource.js";
import type { VariableOr } from "../../src/core/variable.js";

/**
 * Represents a Databricks Unity Catalog Volume resource
 */
export interface Volume extends Resource {
  /**
   * Name of the volume
   */
  name: VariableOr<string>;

  /**
   * Name of the parent catalog
   */
  catalog_name: VariableOr<string>;

  /**
   * Name of the parent schema
   */
  schema_name: VariableOr<string>;

  /**
   * Volume type: MANAGED or EXTERNAL
   */
  volume_type: VariableOr<"MANAGED" | "EXTERNAL">;

  /**
   * Storage location for EXTERNAL volumes
   */
  storage_location?: VariableOr<string>;

  /**
   * User-provided comment/description
   */
  comment?: VariableOr<string>;

  /**
   * Owner of the volume
   */
  owner?: VariableOr<string>;
}

/**
 * Helper function to create a Volume with type safety
 */
export function createVolume(config: Volume): Volume {
  return config;
}
