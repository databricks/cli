/**
 * Resource definitions for Databricks Asset Bundles
 *
 * This module exports helper functions for defining resources.
 * Generated resource types are imported from the generated/ directory.
 */

import type { Bundle } from "../core/bundle.js";
import { Resources } from "../core/resources.js";

// Re-export resource types
export type { App } from "../../generated/apps/index.js";
export type { Cluster } from "../../generated/clusters/index.js";
export type { Dashboard } from "../../generated/dashboards/index.js";
export type { Job } from "../../generated/jobs/index.js";
export type { MlflowExperiment } from "../../generated/mlflow_experiments/index.js";
export type { MlflowModel } from "../../generated/mlflow_models/index.js";
export type { ModelServingEndpoint } from "../../generated/model_serving_endpoints/index.js";
export type { Pipeline } from "../../generated/pipelines/index.js";
export type { QualityMonitor } from "../../generated/quality_monitors/index.js";
export type { RegisteredModel } from "../../generated/registered_models/index.js";
export type { Schema } from "../../generated/schemas/index.js";
export type { Volume } from "../../generated/volumes/index.js";

// Re-export helper functions
export { createApp } from "../../generated/apps/index.js";
export { createCluster } from "../../generated/clusters/index.js";
export { createDashboard } from "../../generated/dashboards/index.js";
export { createJob } from "../../generated/jobs/index.js";
export { createMlflowExperiment } from "../../generated/mlflow_experiments/index.js";
export { createMlflowModel } from "../../generated/mlflow_models/index.js";
export { createModelServingEndpoint } from "../../generated/model_serving_endpoints/index.js";
export { createPipeline } from "../../generated/pipelines/index.js";
export { createQualityMonitor } from "../../generated/quality_monitors/index.js";
export { createRegisteredModel } from "../../generated/registered_models/index.js";
export { createSchema } from "../../generated/schemas/index.js";
export { createVolume } from "../../generated/volumes/index.js";

/**
 * Helper function to create a resources definition
 *
 * @example
 * ```typescript
 * export const loadResources = defineResources((bundle) => {
 *   const resources = new Resources();
 *   // Add resources here
 *   return resources;
 * });
 * ```
 */
export function defineResources(fn: (bundle: Bundle) => Resources): (bundle: Bundle) => Resources {
  return fn;
}
