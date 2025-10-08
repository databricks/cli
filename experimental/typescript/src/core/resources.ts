/**
 * Resources container for managing bundle resources
 *
 * Provides methods to add and manage jobs, pipelines, schemas, and volumes.
 */

import { Diagnostics } from "./diagnostics.js";
import { Location } from "./location.js";
import type { Resource } from "./resource.js";

// Import actual resource types from generated code
import type { Job } from "../../generated/jobs/index.js";
import type { Pipeline } from "../../generated/pipelines/index.js";
import type { Schema } from "../../generated/schemas/index.js";
import type { Volume } from "../../generated/volumes/index.js";

// Re-export resource types
export type { Job, Pipeline, Schema, Volume };

/**
 * Resources is a collection of resources in a bundle.
 *
 * This class is returned by 'loadResources' functions specified in databricks.yml.
 * Each element in 'python.resources' list is a fully qualified function name that
 * returns an instance of Resources class.
 *
 * @example
 * ```yaml
 * python:
 *   resources:
 *     - "resources:loadResources"
 * ```
 *
 * @example
 * ```typescript
 * function loadResources(bundle: Bundle): Resources {
 *   const resources = new Resources();
 *
 *   for (const [name, config] of getConfigs()) {
 *     const job = createJob(config);
 *     resources.addJob(name, job);
 *   }
 *
 *   return resources;
 * }
 * ```
 */
export class Resources {
  private _jobs: Map<string, Job> = new Map();
  private _pipelines: Map<string, Pipeline> = new Map();
  private _schemas: Map<string, Schema> = new Map();
  private _volumes: Map<string, Volume> = new Map();
  private _locationsMap: Map<string, Location> = new Map();
  private _diagnostics: Diagnostics = new Diagnostics();

  /**
   * Returns all jobs
   */
  get jobs(): ReadonlyMap<string, Job> {
    return this._jobs;
  }

  /**
   * Returns all pipelines
   */
  get pipelines(): ReadonlyMap<string, Pipeline> {
    return this._pipelines;
  }

  /**
   * Returns all schemas
   */
  get schemas(): ReadonlyMap<string, Schema> {
    return this._schemas;
  }

  /**
   * Returns all volumes
   */
  get volumes(): ReadonlyMap<string, Volume> {
    return this._volumes;
  }

  /**
   * Returns diagnostics. If there are any diagnostic errors, bundle validation fails.
   */
  get diagnostics(): Diagnostics {
    return this._diagnostics;
  }

  /**
   * Internal: Returns location map for build system
   * @internal
   */
  get _locations(): Map<string, Location> {
    return this._locationsMap;
  }

  /**
   * Adds a resource to the collection of resources.
   *
   * Resource type is detected automatically. Resource name must be unique
   * across all resources of the same type.
   *
   * @param resourceName - Unique identifier for the resource
   * @param resource - The resource to add
   * @param location - Optional location of the resource in the source code
   */
  addResource(_resourceName: string, _resource: Resource, location?: Location): void {
    location = location || Location.fromStack(1);

    // In TypeScript, we can't pattern match as easily as Python
    // For now, we'll require explicit type-specific methods
    // This can be enhanced with type guards in the future
    throw new Error(
      "addResource requires type information. " +
        "Use addJob, addPipeline, addSchema, or addVolume instead."
    );
  }

  /**
   * Adds a job to the collection of resources.
   *
   * Resource name must be unique across all jobs.
   *
   * @param resourceName - Unique identifier for the job
   * @param job - The job to add
   * @param location - Optional location of the job in the source code
   */
  addJob(resourceName: string, job: Job, location?: Location): void {
    const path = ["resources", "jobs", resourceName];
    location = location || Location.fromStack(1);

    if (this._jobs.has(resourceName)) {
      this.addDiagnosticError(
        `Duplicate resource name '${resourceName}' for a job. Resource names must be unique.`,
        { location, path }
      );
    } else {
      if (location) {
        this.addLocation(path, location);
      }
      this._jobs.set(resourceName, job);
    }
  }

  /**
   * Adds a pipeline to the collection of resources.
   *
   * Resource name must be unique across all pipelines.
   *
   * @param resourceName - Unique identifier for the pipeline
   * @param pipeline - The pipeline to add
   * @param location - Optional location of the pipeline in the source code
   */
  addPipeline(resourceName: string, pipeline: Pipeline, location?: Location): void {
    const path = ["resources", "pipelines", resourceName];
    location = location || Location.fromStack(1);

    if (this._pipelines.has(resourceName)) {
      this.addDiagnosticError(
        `Duplicate resource name '${resourceName}' for a pipeline. Resource names must be unique.`,
        { location, path }
      );
    } else {
      if (location) {
        this.addLocation(path, location);
      }
      this._pipelines.set(resourceName, pipeline);
    }
  }

  /**
   * Adds a schema to the collection of resources.
   *
   * Resource name must be unique across all schemas.
   *
   * @param resourceName - Unique identifier for the schema
   * @param schema - The schema to add
   * @param location - Optional location of the schema in the source code
   */
  addSchema(resourceName: string, schema: Schema, location?: Location): void {
    const path = ["resources", "schemas", resourceName];
    location = location || Location.fromStack(1);

    if (this._schemas.has(resourceName)) {
      this.addDiagnosticError(
        `Duplicate resource name '${resourceName}' for a schema. Resource names must be unique.`,
        { location, path }
      );
    } else {
      if (location) {
        this.addLocation(path, location);
      }
      this._schemas.set(resourceName, schema);
    }
  }

  /**
   * Adds a volume to the collection of resources.
   *
   * Resource name must be unique across all volumes.
   *
   * @param resourceName - Unique identifier for the volume
   * @param volume - The volume to add
   * @param location - Optional location of the volume in the source code
   */
  addVolume(resourceName: string, volume: Volume, location?: Location): void {
    const path = ["resources", "volumes", resourceName];
    location = location || Location.fromStack(1);

    if (this._volumes.has(resourceName)) {
      this.addDiagnosticError(
        `Duplicate resource name '${resourceName}' for a volume. Resource names must be unique.`,
        { location, path }
      );
    } else {
      if (location) {
        this.addLocation(path, location);
      }
      this._volumes.set(resourceName, volume);
    }
  }

  /**
   * Associate source code location with a path in the bundle configuration.
   */
  addLocation(path: readonly string[], location: Location): void {
    const key = path.join(".");
    this._locationsMap.set(key, location);
  }

  /**
   * Add diagnostics from another Diagnostics object.
   */
  addDiagnostics(other: Diagnostics): void {
    this._diagnostics = this._diagnostics.extend(other);
  }

  /**
   * Report a diagnostic error.
   *
   * If there are any diagnostic errors, bundle validation fails.
   */
  addDiagnosticError(
    message: string,
    options?: {
      detail?: string;
      path?: readonly string[];
      location?: Location;
    }
  ): void {
    this.addDiagnostics(Diagnostics.createError(message, options));
  }

  /**
   * Report a diagnostic warning.
   *
   * Warnings are informational and do not cause bundle validation to fail.
   */
  addDiagnosticWarning(
    message: string,
    options?: {
      detail?: string;
      path?: readonly string[];
      location?: Location;
    }
  ): void {
    this.addDiagnostics(Diagnostics.createWarning(message, options));
  }

  /**
   * Add resources from another Resources object.
   *
   * Adds error to diagnostics if there are duplicate resource names.
   */
  addResources(other: Resources): void {
    for (const [name, job] of other.jobs) {
      this.addJob(name, job);
    }

    for (const [name, pipeline] of other.pipelines) {
      this.addPipeline(name, pipeline);
    }

    for (const [name, schema] of other.schemas) {
      this.addSchema(name, schema);
    }

    for (const [name, volume] of other.volumes) {
      this.addVolume(name, volume);
    }

    for (const [key, location] of other._locations) {
      this._locationsMap.set(key, location);
    }

    this._diagnostics = this._diagnostics.extend(other._diagnostics);
  }

  /**
   * Converts resources to a JSON-serializable format for CLI integration.
   */
  toJSON(): Record<string, Record<string, unknown>> {
    const result: Record<string, Record<string, unknown>> = {};

    if (this._jobs.size > 0) {
      result.jobs = Object.fromEntries(this._jobs);
    }

    if (this._pipelines.size > 0) {
      result.pipelines = Object.fromEntries(this._pipelines);
    }

    if (this._schemas.size > 0) {
      result.schemas = Object.fromEntries(this._schemas);
    }

    if (this._volumes.size > 0) {
      result.volumes = Object.fromEntries(this._volumes);
    }

    return result;
  }
}
