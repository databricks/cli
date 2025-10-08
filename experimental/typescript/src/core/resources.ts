/**
 * Resources container for managing bundle resources
 *
 * Provides methods to add and manage all Databricks resource types.
 */

import { Diagnostics } from "./diagnostics.js";
import { Location } from "./location.js";
import type { Resource } from "./resource.js";
import { transformToJSON } from "./transform.js";

// Import actual resource types from generated code
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

// Re-export resource types
export type { App, Cluster, Dashboard, Job, MlflowExperiment, MlflowModel, ModelServingEndpoint, Pipeline, QualityMonitor, RegisteredModel, Schema, Volume };

/**
 * Enum of all supported resource types for type-safe resource management
 */
export enum ResourceType {
  APPS = "apps",
  CLUSTERS = "clusters",
  DASHBOARDS = "dashboards",
  JOBS = "jobs",
  MLFLOW_EXPERIMENTS = "mlflow_experiments",
  MLFLOW_MODELS = "mlflow_models",
  MODEL_SERVING_ENDPOINTS = "model_serving_endpoints",
  PIPELINES = "pipelines",
  QUALITY_MONITORS = "quality_monitors",
  REGISTERED_MODELS = "registered_models",
  SCHEMAS = "schemas",
  VOLUMES = "volumes",
}

/**
 * Type mapping from ResourceType enum to actual resource types
 */
export type ResourceTypeMap = {
  [ResourceType.APPS]: App;
  [ResourceType.CLUSTERS]: Cluster;
  [ResourceType.DASHBOARDS]: Dashboard;
  [ResourceType.JOBS]: Job;
  [ResourceType.MLFLOW_EXPERIMENTS]: MlflowExperiment;
  [ResourceType.MLFLOW_MODELS]: MlflowModel;
  [ResourceType.MODEL_SERVING_ENDPOINTS]: ModelServingEndpoint;
  [ResourceType.PIPELINES]: Pipeline;
  [ResourceType.QUALITY_MONITORS]: QualityMonitor;
  [ResourceType.REGISTERED_MODELS]: RegisteredModel;
  [ResourceType.SCHEMAS]: Schema;
  [ResourceType.VOLUMES]: Volume;
};

/**
 * Metadata for a resource type
 */
interface ResourceTypeMetadata<T = unknown> {
  readonly type: ResourceType;
  readonly pluralName: string;
  readonly map: Map<string, T>;
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
    this.registerResourceType<App>(ResourceType.APPS);
    this.registerResourceType<Cluster>(ResourceType.CLUSTERS);
    this.registerResourceType<Dashboard>(ResourceType.DASHBOARDS);
    this.registerResourceType<Job>(ResourceType.JOBS);
    this.registerResourceType<MlflowExperiment>(ResourceType.MLFLOW_EXPERIMENTS);
    this.registerResourceType<MlflowModel>(ResourceType.MLFLOW_MODELS);
    this.registerResourceType<ModelServingEndpoint>(ResourceType.MODEL_SERVING_ENDPOINTS);
    this.registerResourceType<Pipeline>(ResourceType.PIPELINES);
    this.registerResourceType<QualityMonitor>(ResourceType.QUALITY_MONITORS);
    this.registerResourceType<RegisteredModel>(ResourceType.REGISTERED_MODELS);
    this.registerResourceType<Schema>(ResourceType.SCHEMAS);
    this.registerResourceType<Volume>(ResourceType.VOLUMES);
  }

  /**
   * Register a resource type in the registry
   */
  private registerResourceType<T>(type: ResourceType): void {
    this.registry.set(type, {
      type,
      pluralName: type,
      map: new Map<string, T>(),
    });
  }

  /**
   * Get metadata for a resource type
   */
  private getMetadata<K extends ResourceType>(type: K): ResourceTypeMetadata<ResourceTypeMap[K]> {
    const metadata = this.registry.get(type);
    if (!metadata) {
      throw new Error(`Resource type not registered: ${type}`);
    }
    return metadata as ResourceTypeMetadata<ResourceTypeMap[K]>;
  }

  /**
   * Generic method to add a resource
   */
  private addResource<K extends ResourceType>(
    type: K,
    name: string,
    resource: ResourceTypeMap[K],
    location?: Location
  ): void {
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
  private getResources<K extends ResourceType>(type: K): ReadonlyMap<string, ResourceTypeMap[K]> {
    return this.getMetadata(type).map as ReadonlyMap<string, ResourceTypeMap[K]>;
  }

  /**
   * Returns all apps
   */
  get apps(): ReadonlyMap<string, App> {
    return this.getResources(ResourceType.APPS);
  }

  /**
   * Returns all clusters
   */
  get clusters(): ReadonlyMap<string, Cluster> {
    return this.getResources(ResourceType.CLUSTERS);
  }

  /**
   * Returns all dashboards
   */
  get dashboards(): ReadonlyMap<string, Dashboard> {
    return this.getResources(ResourceType.DASHBOARDS);
  }

  /**
   * Returns all jobs
   */
  get jobs(): ReadonlyMap<string, Job> {
    return this.getResources(ResourceType.JOBS);
  }

  /**
   * Returns all mlflow experiments
   */
  get mlflow_experiments(): ReadonlyMap<string, MlflowExperiment> {
    return this.getResources(ResourceType.MLFLOW_EXPERIMENTS);
  }

  /**
   * Returns all mlflow models
   */
  get mlflow_models(): ReadonlyMap<string, MlflowModel> {
    return this.getResources(ResourceType.MLFLOW_MODELS);
  }

  /**
   * Returns all model serving endpoints
   */
  get model_serving_endpoints(): ReadonlyMap<string, ModelServingEndpoint> {
    return this.getResources(ResourceType.MODEL_SERVING_ENDPOINTS);
  }

  /**
   * Returns all pipelines
   */
  get pipelines(): ReadonlyMap<string, Pipeline> {
    return this.getResources(ResourceType.PIPELINES);
  }

  /**
   * Returns all quality monitors
   */
  get quality_monitors(): ReadonlyMap<string, QualityMonitor> {
    return this.getResources(ResourceType.QUALITY_MONITORS);
  }

  /**
   * Returns all registered models
   */
  get registered_models(): ReadonlyMap<string, RegisteredModel> {
    return this.getResources(ResourceType.REGISTERED_MODELS);
  }

  /**
   * Returns all schemas
   */
  get schemas(): ReadonlyMap<string, Schema> {
    return this.getResources(ResourceType.SCHEMAS);
  }

  /**
   * Returns all volumes
   */
  get volumes(): ReadonlyMap<string, Volume> {
    return this.getResources(ResourceType.VOLUMES);
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
   * Add an app to the bundle
   */
  addApp(name: string, resource: App, location?: Location): void {
    this.addResource(ResourceType.APPS, name, resource, location);
  }

  /**
   * Add a cluster to the bundle
   */
  addCluster(name: string, resource: Cluster, location?: Location): void {
    this.addResource(ResourceType.CLUSTERS, name, resource, location);
  }

  /**
   * Add a dashboard to the bundle
   */
  addDashboard(name: string, resource: Dashboard, location?: Location): void {
    this.addResource(ResourceType.DASHBOARDS, name, resource, location);
  }

  /**
   * Add a job to the bundle
   */
  addJob(name: string, resource: Job, location?: Location): void {
    this.addResource(ResourceType.JOBS, name, resource, location);
  }

  /**
   * Add an MLflow experiment to the bundle
   */
  addMlflowExperiment(name: string, resource: MlflowExperiment, location?: Location): void {
    this.addResource(ResourceType.MLFLOW_EXPERIMENTS, name, resource, location);
  }

  /**
   * Add an MLflow model to the bundle
   */
  addMlflowModel(name: string, resource: MlflowModel, location?: Location): void {
    this.addResource(ResourceType.MLFLOW_MODELS, name, resource, location);
  }

  /**
   * Add a model serving endpoint to the bundle
   */
  addModelServingEndpoint(name: string, resource: ModelServingEndpoint, location?: Location): void {
    this.addResource(ResourceType.MODEL_SERVING_ENDPOINTS, name, resource, location);
  }

  /**
   * Add a pipeline to the bundle
   */
  addPipeline(name: string, resource: Pipeline, location?: Location): void {
    this.addResource(ResourceType.PIPELINES, name, resource, location);
  }

  /**
   * Add a quality monitor to the bundle
   */
  addQualityMonitor(name: string, resource: QualityMonitor, location?: Location): void {
    this.addResource(ResourceType.QUALITY_MONITORS, name, resource, location);
  }

  /**
   * Add a registered model to the bundle
   */
  addRegisteredModel(name: string, resource: RegisteredModel, location?: Location): void {
    this.addResource(ResourceType.REGISTERED_MODELS, name, resource, location);
  }

  /**
   * Add a schema to the bundle
   */
  addSchema(name: string, resource: Schema, location?: Location): void {
    this.addResource(ResourceType.SCHEMAS, name, resource, location);
  }

  /**
   * Add a volume to the bundle
   */
  addVolume(name: string, resource: Volume, location?: Location): void {
    this.addResource(ResourceType.VOLUMES, name, resource, location);
  }

  /**
   * Add resources from another Resources instance
   */
  addResources(other: Resources): void {
    // Iterate through all resource types in the registry
    for (const type of this.registry.keys()) {
      const otherMetadata = other.getMetadata(type);
      for (const [name, resource] of otherMetadata.map.entries()) {
        this.addResource(type, name, resource as ResourceTypeMap[typeof type]);
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
          Array.from(metadata.map.entries()).map(([key, value]) => [key, transformToJSON(value)])
        );
      }
    }

    return resources;
  }
}
