/**
 * Resource definitions for Databricks Asset Bundles
 *
 * This module exports helper functions for defining resources.
 * Generated resource types are imported from the generated/ directory.
 */

import type { Bundle } from "../core/bundle.js";
import { Resources } from "../core/resources.js";

// Re-export resource classes
export { App } from "../../generated/apps/index.js";
export { Cluster } from "../../generated/clusters/index.js";
export { Dashboard } from "../../generated/dashboards/index.js";
export { Job } from "../../generated/jobs/index.js";
export { MlflowExperiment } from "../../generated/mlflow_experiments/index.js";
export { MlflowModel } from "../../generated/mlflow_models/index.js";
export { ModelServingEndpoint } from "../../generated/model_serving_endpoints/index.js";
export { Pipeline } from "../../generated/pipelines/index.js";
export { QualityMonitor } from "../../generated/quality_monitors/index.js";
export { RegisteredModel } from "../../generated/registered_models/index.js";
export { Schema } from "../../generated/schemas/index.js";
export { Volume } from "../../generated/volumes/index.js";

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
