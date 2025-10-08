/**
 * Resources container for managing bundle resources
 *
 * Provides methods to add and manage all Databricks resource types.
 */

import { Diagnostics } from "./diagnostics.js";
import { Location } from "./location.js";

// Import actual resource types from generated code (type-only to avoid circular dependency)
import type { App } from "../../generated/apps/index.js";
import type { Cluster } from "../../generated/clusters/index.js";
import type { Dashboard } from "../../generated/dashboards/index.js";
import type { Job } from "../../generated/jobs/index.js";
import type { MlflowExperiment } from "../../generated/mlflow_experiments/index.js";
import type { MlflowModel } from "../../generated/mlflow_models/index.js";
import type { ModelServingEndpoint } from "../../generated/model_serving_endpoints/index.js";
import type { Pipeline } from "../../generated/pipelines/index.js";
import type { QualityMonitor } from "../../generated/quality_monitors/index.js";
import type { RegisteredModel } from "../../generated/registered_models/index.js";
import type { Schema } from "../../generated/schemas/index.js";
import type { Volume } from "../../generated/volumes/index.js";
import type { Resource } from "./resource.js";

/**
 * Enum of all supported resource types for type-safe resource management
 */
export type ResourceType = "apps" | "clusters" | "dashboards" | "jobs" | "mlflow_experiments" | "mlflow_models" | "model_serving_endpoints" | "pipelines" | "quality_monitors" | "registered_models" | "schemas" | "volumes";

/**
 * Type mapping from ResourceType enum to actual resource types
 */
export type ResourceTypeMap = {
  apps: App;
  clusters: Cluster;
  dashboards: Dashboard;
  jobs: Job;
  mlflow_experiments: MlflowExperiment;
  mlflow_models: MlflowModel;
  model_serving_endpoints: ModelServingEndpoint;
  pipelines: Pipeline;
  quality_monitors: QualityMonitor;
  registered_models: RegisteredModel;
  schemas: Schema;
  volumes: Volume;
};

/**
 * Metadata for a resource type
 */
interface ResourceTypeMetadata {
  readonly type: ResourceType;
  readonly pluralName: string;
  readonly map: Map<string, Resource<unknown>>;
}

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
  private readonly registry: Map<ResourceType, ResourceTypeMetadata> = new Map();
  private _locationsMap: Map<string, Location> = new Map();
  private _diagnostics: Diagnostics = new Diagnostics();

  constructor() {
    // Initialize registry with all resource types
    this.registerResourceType<App>("apps");
    this.registerResourceType<Cluster>("clusters");
    this.registerResourceType<Dashboard>("dashboards");
    this.registerResourceType<Job>("jobs");
    this.registerResourceType<MlflowExperiment>("mlflow_experiments");
    this.registerResourceType<MlflowModel>("mlflow_models");
    this.registerResourceType<ModelServingEndpoint>("model_serving_endpoints");
    this.registerResourceType<Pipeline>("pipelines");
    this.registerResourceType<QualityMonitor>("quality_monitors");
    this.registerResourceType<RegisteredModel>("registered_models");
    this.registerResourceType<Schema>("schemas");
    this.registerResourceType<Volume>("volumes");
  }

  /**
   * Register a resource type in the registry
   */
  private registerResourceType<T>(type: ResourceType): void {
    this.registry.set(type, {
      type,
      pluralName: type,
      map: new Map<string, Resource<T>>(),
    });
  }

  /**
   * Get metadata for a resource type
   */
  private getMetadata<K extends ResourceType>(type: K): ResourceTypeMetadata {
    const metadata = this.registry.get(type);
    if (!metadata) {
      throw new Error(`Resource type not registered: ${type}`);
    }
    return metadata;
  }

  /**
   * Generic method to add a resource
   */
  public addResource(
    name: string,
    resource: ResourceTypeMap[ResourceType],
    location?: Location
  ): void {
    const type = resource.type;
    const metadata = this.getMetadata(type);

    if (metadata.map.has(name)) {
      this.addDiagnosticWarning(`Duplicate ${type} resource: ${name}`, {
        path: ["resources", type, name],
        location,
      });
    }

    metadata.map.set(name, resource);

    if (location) {
      this.addLocation(["resources", type, name], location);
    }
  }

  /**
   * Generic method to get resources of a type
   */
  public getResources(type: ResourceType) {
    return this.getMetadata(type).map;
  }

  /**
   * Returns diagnostics accumulated during resource processing
   */
  get diagnostics(): Diagnostics {
    return this._diagnostics;
  }

  /**
   * Returns locations map for internal use by build system
   * @internal
   */
  get _locations(): Map<string, Location> {
    return this._locationsMap;
  }

  /**
   * Add resources from another Resources instance
   */
  addResources(other: Resources): void {
    // Iterate through all resource types in the registry
    for (const type of this.registry.keys()) {
      const otherMetadata = other.getMetadata(type);
      for (const [name, resource] of otherMetadata.map.entries()) {
        this.addResource(name, resource as ResourceTypeMap[typeof type]);
      }
    }

    // Extend diagnostics
    this._diagnostics = this._diagnostics.extend(other.diagnostics);
  }

  /**
   * Add a diagnostic error
   */
  addDiagnosticError(
    summary: string,
    options?: {
      detail?: string;
      location?: Location;
      path?: readonly string[];
    }
  ): void {
    this._diagnostics = this._diagnostics.extend(
      Diagnostics.createError(summary, options)
    );
  }

  /**
   * Add a diagnostic warning
   */
  addDiagnosticWarning(
    summary: string,
    options?: {
      detail?: string;
      location?: Location;
      path?: readonly string[];
    }
  ): void {
    this._diagnostics = this._diagnostics.extend(
      Diagnostics.createWarning(summary, options)
    );
  }

  /**
   * Add a location for a resource path
   */
  addLocation(path: readonly string[], location: Location): void {
    this._locationsMap.set(path.join("."), location);
  }

  /**
   * Convert to JSON representation
   */
  toJSON(): Record<string, Record<string, unknown>> {
    const resources: Record<string, Record<string, unknown>> = {};

    // Iterate through all resource types in the registry
    for (const [type, metadata] of this.registry.entries()) {
      if (metadata.map.size > 0) {
        resources[type] = Object.fromEntries(
          Array.from(metadata.map.entries()).map(([key, value]) => [key, value.toJSON()])
        );
      }
    }

    return resources;
  }
}
