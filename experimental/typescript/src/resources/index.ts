/**
 * Resource definitions for Databricks Asset Bundles
 *
 * This module exports helper functions for defining resources.
 * Generated resource types are imported from the generated/ directory.
 */

import type { Bundle } from "../core/bundle.js";
import { Resources } from "../core/resources.js";

// Re-export resource types
export type { Job } from "../../generated/jobs/index.js";
export type { Pipeline } from "../../generated/pipelines/index.js";
export type { Schema } from "../../generated/schemas/index.js";
export type { Volume } from "../../generated/volumes/index.js";

// Re-export helper functions
export { createJob } from "../../generated/jobs/index.js";
export { createPipeline } from "../../generated/pipelines/index.js";
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
