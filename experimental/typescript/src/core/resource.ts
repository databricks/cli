/**
 * Base class for all Databricks resources.
 *
 * All resource types (Job, Pipeline, Schema, Volume, etc.) should extend this class.
 */
export abstract class Resource {
  /**
   * Internal marker to identify Resource instances at runtime
   * @internal
   */
  private readonly __isResource = true;

  /**
   * Create a new Resource instance
   * @param data - Resource configuration data
   */
  constructor(data?: Record<string, unknown>) {
    if (data) {
      Object.assign(this, data);
    }
  }
}

/**
 * Type guard to check if an object is a Resource instance.
 */
export function isResource(value: unknown): value is Resource {
  return typeof value === "object" && value !== null && "__isResource" in value;
}
