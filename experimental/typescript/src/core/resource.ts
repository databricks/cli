/**
 * Base class for all resources
 */

/**
 * Base interface for all Databricks resources.
 *
 * All resource types (Job, Pipeline, Schema, Volume, etc.) should extend this interface.
 */
export interface Resource {
  // Marker interface - specific resource types will add their own properties
}

/**
 * Type guard to check if an object is a Resource.
 *
 * In TypeScript, we can't do runtime type checking as easily as Python,
 * so this is a basic implementation that checks for an object.
 */
export function isResource(value: unknown): value is Resource {
  return typeof value === "object" && value !== null;
}
