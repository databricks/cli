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
  private _apps: Map<string, App> = new Map();
  private _clusters: Map<string, Cluster> = new Map();
  private _dashboards: Map<string, Dashboard> = new Map();
  private _jobs: Map<string, Job> = new Map();
  private _mlflow_experiments: Map<string, MlflowExperiment> = new Map();
  private _mlflow_models: Map<string, MlflowModel> = new Map();
  private _model_serving_endpoints: Map<string, ModelServingEndpoint> = new Map();
  private _pipelines: Map<string, Pipeline> = new Map();
  private _quality_monitors: Map<string, QualityMonitor> = new Map();
  private _registered_models: Map<string, RegisteredModel> = new Map();
  private _schemas: Map<string, Schema> = new Map();
  private _volumes: Map<string, Volume> = new Map();
  private _locationsMap: Map<string, Location> = new Map();
  private _diagnostics: Diagnostics = new Diagnostics();

  /**
   * Returns all apps
   */
  get apps(): ReadonlyMap<string, App> {
    return this._apps;
  }

  /**
   * Returns all clusters
   */
  get clusters(): ReadonlyMap<string, Cluster> {
    return this._clusters;
  }

  /**
   * Returns all dashboards
   */
  get dashboards(): ReadonlyMap<string, Dashboard> {
    return this._dashboards;
  }

  /**
   * Returns all jobs
   */
  get jobs(): ReadonlyMap<string, Job> {
    return this._jobs;
  }

  /**
   * Returns all mlflow experiments
   */
  get mlflow_experiments(): ReadonlyMap<string, MlflowExperiment> {
    return this._mlflow_experiments;
  }

  /**
   * Returns all mlflow models
   */
  get mlflow_models(): ReadonlyMap<string, MlflowModel> {
    return this._mlflow_models;
  }

  /**
   * Returns all model serving endpoints
   */
  get model_serving_endpoints(): ReadonlyMap<string, ModelServingEndpoint> {
    return this._model_serving_endpoints;
  }

  /**
   * Returns all pipelines
   */
  get pipelines(): ReadonlyMap<string, Pipeline> {
    return this._pipelines;
  }

  /**
   * Returns all quality monitors
   */
  get quality_monitors(): ReadonlyMap<string, QualityMonitor> {
    return this._quality_monitors;
  }

  /**
   * Returns all registered models
   */
  get registered_models(): ReadonlyMap<string, RegisteredModel> {
    return this._registered_models;
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
    if (this._apps.has(name)) {
      this.addDiagnosticWarning(`Duplicate apps resource: ${name}`, {
        path: ["resources", "apps", name],
        location,
      });
    }
    this._apps.set(name, resource);
    if (location) {
      this.addLocation(["resources", "apps", name], location);
    }
  }

  /**
   * Add a cluster to the bundle
   */
  addCluster(name: string, resource: Cluster, location?: Location): void {
    if (this._clusters.has(name)) {
      this.addDiagnosticWarning(`Duplicate clusters resource: ${name}`, {
        path: ["resources", "clusters", name],
        location,
      });
    }
    this._clusters.set(name, resource);
    if (location) {
      this.addLocation(["resources", "clusters", name], location);
    }
  }

  /**
   * Add a dashboard to the bundle
   */
  addDashboard(name: string, resource: Dashboard, location?: Location): void {
    if (this._dashboards.has(name)) {
      this.addDiagnosticWarning(`Duplicate dashboards resource: ${name}`, {
        path: ["resources", "dashboards", name],
        location,
      });
    }
    this._dashboards.set(name, resource);
    if (location) {
      this.addLocation(["resources", "dashboards", name], location);
    }
  }

  /**
   * Add a job to the bundle
   */
  addJob(name: string, resource: Job, location?: Location): void {
    if (this._jobs.has(name)) {
      this.addDiagnosticWarning(`Duplicate jobs resource: ${name}`, {
        path: ["resources", "jobs", name],
        location,
      });
    }
    this._jobs.set(name, resource);
    if (location) {
      this.addLocation(["resources", "jobs", name], location);
    }
  }

  /**
   * Add an MLflow experiment to the bundle
   */
  addMlflowExperiment(name: string, resource: MlflowExperiment, location?: Location): void {
    if (this._mlflow_experiments.has(name)) {
      this.addDiagnosticWarning(`Duplicate mlflow_experiments resource: ${name}`, {
        path: ["resources", "mlflow_experiments", name],
        location,
      });
    }
    this._mlflow_experiments.set(name, resource);
    if (location) {
      this.addLocation(["resources", "mlflow_experiments", name], location);
    }
  }

  /**
   * Add an MLflow model to the bundle
   */
  addMlflowModel(name: string, resource: MlflowModel, location?: Location): void {
    if (this._mlflow_models.has(name)) {
      this.addDiagnosticWarning(`Duplicate mlflow_models resource: ${name}`, {
        path: ["resources", "mlflow_models", name],
        location,
      });
    }
    this._mlflow_models.set(name, resource);
    if (location) {
      this.addLocation(["resources", "mlflow_models", name], location);
    }
  }

  /**
   * Add a model serving endpoint to the bundle
   */
  addModelServingEndpoint(name: string, resource: ModelServingEndpoint, location?: Location): void {
    if (this._model_serving_endpoints.has(name)) {
      this.addDiagnosticWarning(`Duplicate model_serving_endpoints resource: ${name}`, {
        path: ["resources", "model_serving_endpoints", name],
        location,
      });
    }
    this._model_serving_endpoints.set(name, resource);
    if (location) {
      this.addLocation(["resources", "model_serving_endpoints", name], location);
    }
  }

  /**
   * Add a pipeline to the bundle
   */
  addPipeline(name: string, resource: Pipeline, location?: Location): void {
    if (this._pipelines.has(name)) {
      this.addDiagnosticWarning(`Duplicate pipelines resource: ${name}`, {
        path: ["resources", "pipelines", name],
        location,
      });
    }
    this._pipelines.set(name, resource);
    if (location) {
      this.addLocation(["resources", "pipelines", name], location);
    }
  }

  /**
   * Add a quality monitor to the bundle
   */
  addQualityMonitor(name: string, resource: QualityMonitor, location?: Location): void {
    if (this._quality_monitors.has(name)) {
      this.addDiagnosticWarning(`Duplicate quality_monitors resource: ${name}`, {
        path: ["resources", "quality_monitors", name],
        location,
      });
    }
    this._quality_monitors.set(name, resource);
    if (location) {
      this.addLocation(["resources", "quality_monitors", name], location);
    }
  }

  /**
   * Add a registered model to the bundle
   */
  addRegisteredModel(name: string, resource: RegisteredModel, location?: Location): void {
    if (this._registered_models.has(name)) {
      this.addDiagnosticWarning(`Duplicate registered_models resource: ${name}`, {
        path: ["resources", "registered_models", name],
        location,
      });
    }
    this._registered_models.set(name, resource);
    if (location) {
      this.addLocation(["resources", "registered_models", name], location);
    }
  }

  /**
   * Add a schema to the bundle
   */
  addSchema(name: string, resource: Schema, location?: Location): void {
    if (this._schemas.has(name)) {
      this.addDiagnosticWarning(`Duplicate schemas resource: ${name}`, {
        path: ["resources", "schemas", name],
        location,
      });
    }
    this._schemas.set(name, resource);
    if (location) {
      this.addLocation(["resources", "schemas", name], location);
    }
  }

  /**
   * Add a volume to the bundle
   */
  addVolume(name: string, resource: Volume, location?: Location): void {
    if (this._volumes.has(name)) {
      this.addDiagnosticWarning(`Duplicate volumes resource: ${name}`, {
        path: ["resources", "volumes", name],
        location,
      });
    }
    this._volumes.set(name, resource);
    if (location) {
      this.addLocation(["resources", "volumes", name], location);
    }
  }

  /**
   * Add resources from another Resources instance
   */
  addResources(other: Resources): void {
    for (const [name, resource] of other.apps.entries()) {
      this.addApp(name, resource);
    }
    for (const [name, resource] of other.clusters.entries()) {
      this.addCluster(name, resource);
    }
    for (const [name, resource] of other.dashboards.entries()) {
      this.addDashboard(name, resource);
    }
    for (const [name, resource] of other.jobs.entries()) {
      this.addJob(name, resource);
    }
    for (const [name, resource] of other.mlflow_experiments.entries()) {
      this.addMlflowExperiment(name, resource);
    }
    for (const [name, resource] of other.mlflow_models.entries()) {
      this.addMlflowModel(name, resource);
    }
    for (const [name, resource] of other.model_serving_endpoints.entries()) {
      this.addModelServingEndpoint(name, resource);
    }
    for (const [name, resource] of other.pipelines.entries()) {
      this.addPipeline(name, resource);
    }
    for (const [name, resource] of other.quality_monitors.entries()) {
      this.addQualityMonitor(name, resource);
    }
    for (const [name, resource] of other.registered_models.entries()) {
      this.addRegisteredModel(name, resource);
    }
    for (const [name, resource] of other.schemas.entries()) {
      this.addSchema(name, resource);
    }
    for (const [name, resource] of other.volumes.entries()) {
      this.addVolume(name, resource);
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

    if (this._apps.size > 0) {
      resources.apps = Object.fromEntries(
        Array.from(this._apps.entries()).map(([key, value]) => [key, transformToJSON(value)])
      );
    }
    if (this._clusters.size > 0) {
      resources.clusters = Object.fromEntries(
        Array.from(this._clusters.entries()).map(([key, value]) => [key, transformToJSON(value)])
      );
    }
    if (this._dashboards.size > 0) {
      resources.dashboards = Object.fromEntries(
        Array.from(this._dashboards.entries()).map(([key, value]) => [key, transformToJSON(value)])
      );
    }
    if (this._jobs.size > 0) {
      resources.jobs = Object.fromEntries(
        Array.from(this._jobs.entries()).map(([key, value]) => [key, transformToJSON(value)])
      );
    }
    if (this._mlflow_experiments.size > 0) {
      resources.mlflow_experiments = Object.fromEntries(
        Array.from(this._mlflow_experiments.entries()).map(([key, value]) => [key, transformToJSON(value)])
      );
    }
    if (this._mlflow_models.size > 0) {
      resources.mlflow_models = Object.fromEntries(
        Array.from(this._mlflow_models.entries()).map(([key, value]) => [key, transformToJSON(value)])
      );
    }
    if (this._model_serving_endpoints.size > 0) {
      resources.model_serving_endpoints = Object.fromEntries(
        Array.from(this._model_serving_endpoints.entries()).map(([key, value]) => [key, transformToJSON(value)])
      );
    }
    if (this._pipelines.size > 0) {
      resources.pipelines = Object.fromEntries(
        Array.from(this._pipelines.entries()).map(([key, value]) => [key, transformToJSON(value)])
      );
    }
    if (this._quality_monitors.size > 0) {
      resources.quality_monitors = Object.fromEntries(
        Array.from(this._quality_monitors.entries()).map(([key, value]) => [key, transformToJSON(value)])
      );
    }
    if (this._registered_models.size > 0) {
      resources.registered_models = Object.fromEntries(
        Array.from(this._registered_models.entries()).map(([key, value]) => [key, transformToJSON(value)])
      );
    }
    if (this._schemas.size > 0) {
      resources.schemas = Object.fromEntries(
        Array.from(this._schemas.entries()).map(([key, value]) => [key, transformToJSON(value)])
      );
    }
    if (this._volumes.size > 0) {
      resources.volumes = Object.fromEntries(
        Array.from(this._volumes.entries()).map(([key, value]) => [key, transformToJSON(value)])
      );
    }

    return resources;
  }
}
