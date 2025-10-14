/**
 * Base class for all Databricks resources.
 *
 * All resource types (Job, Pipeline, Schema, Volume, etc.) should extend this class.
 */
import { transformToJSON } from "./transform.js";
import { type ResourceType } from "./resources.js";

/**
 * Base class for all Databricks resources.
 *
 * This class provides common functionality for all resource types including
 * type tracking, name tracking, and JSON serialization.
 *
 * @template T - The type of the resource parameters
 *
 * @example
 * ```typescript
 * class Job extends Resource<JobParams> {
 *   constructor(name: string, params: JobParams) {
 *     super(name, params, "jobs");
 *   }
 * }
 * ```
 */
export class Resource<T> {
  /**
   * The resource type namespace (e.g., "jobs", "pipelines")
   */
  public readonly type: ResourceType;

  /**
   * The name of this resource in the bundle
   */
  public readonly dabsName: string;

  /**
   * Creates a new Resource instance.
   *
   * @param dabsName - The name of the resource in the bundle
   * @param data - The resource parameters/configuration
   * @param type - The resource type namespace
   */
  constructor(dabsName: string, public readonly data: T, type: ResourceType) {
    this.type = type;
    this.dabsName = dabsName;
  }

  /**
   * Converts the resource to JSON format for serialization.
   *
   * @returns JSON-serializable representation of the resource data
   */
  toJSON() {
    return transformToJSON(this.data);
  }
}