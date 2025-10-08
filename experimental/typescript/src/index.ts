/**
 * @databricks/bundles - TypeScript support for Databricks Asset Bundles
 *
 * This package provides type-safe TypeScript APIs for defining Databricks resources
 * such as jobs, pipelines, schemas, and volumes as code.
 */

// Core framework
export * from "./core/index.js";

// Resource types
export type { App } from "../generated/apps/index.js";
export type { Cluster } from "../generated/clusters/index.js";
export type { Dashboard } from "../generated/dashboards/index.js";
export type { Job } from "../generated/jobs/index.js";
export type { MlflowExperiment } from "../generated/mlflow_experiments/index.js";
export type { MlflowModel } from "../generated/mlflow_models/index.js";
export type { ModelServingEndpoint } from "../generated/model_serving_endpoints/index.js";
export type { Pipeline } from "../generated/pipelines/index.js";
export type { QualityMonitor } from "../generated/quality_monitors/index.js";
export type { RegisteredModel } from "../generated/registered_models/index.js";
export type { Schema } from "../generated/schemas/index.js";
export type { Volume } from "../generated/volumes/index.js";

// Helper functions for creating resources
export { createApp } from "../generated/apps/index.js";
export { createCluster } from "../generated/clusters/index.js";
export { createDashboard } from "../generated/dashboards/index.js";
export { createJob } from "../generated/jobs/index.js";
export { createMlflowExperiment } from "../generated/mlflow_experiments/index.js";
export { createMlflowModel } from "../generated/mlflow_models/index.js";
export { createModelServingEndpoint } from "../generated/model_serving_endpoints/index.js";
export { createPipeline } from "../generated/pipelines/index.js";
export { createQualityMonitor } from "../generated/quality_monitors/index.js";
export { createRegisteredModel } from "../generated/registered_models/index.js";
export { createSchema } from "../generated/schemas/index.js";
export { createVolume } from "../generated/volumes/index.js";
