/**
 * @databricks/bundles - TypeScript support for Databricks Asset Bundles
 *
 * This package provides type-safe TypeScript APIs for defining Databricks resources
 * such as jobs, pipelines, schemas, and volumes as code.
 */

// Core framework
export * from "./core/index.js";

// Resource types
export type { Job } from "../generated/jobs/index.js";
export type { Pipeline } from "../generated/pipelines/index.js";
export type { Schema } from "../generated/schemas/index.js";
export type { Volume } from "../generated/volumes/index.js";

// Helper functions for creating resources
export { createJob } from "../generated/jobs/index.js";
export { createPipeline } from "../generated/pipelines/index.js";
export { createSchema } from "../generated/schemas/index.js";
export { createVolume } from "../generated/volumes/index.js";
