/**
 * Resource mutator decorators and types
 *
 * Resource mutators allow modifying resources before deployment.
 * Mutators are applied to both YAML-defined and TypeScript-defined resources.
 */

import type { Bundle } from "../core/bundle.js";
import type { Resource } from "../core/resource.js";
import type { Job } from "../../generated/jobs/index.js";
import type { Pipeline } from "../../generated/pipelines/index.js";
import type { Schema } from "../../generated/schemas/index.js";
import type { Volume } from "../../generated/volumes/index.js";

/**
 * Resource mutator wrapper
 *
 * Stores the resource type and the mutator function.
 */
export interface ResourceMutator<T extends Resource<any> = Resource<any>> {
  resourceType: new () => T;
  function: ((bundle: Bundle, resource: T) => T) | ((resource: T) => T);
}

/**
 * Job mutator decorator
 *
 * Creates a mutator that applies only to Job resources.
 *
 * @example
 * ```typescript
 * import { jobMutator } from "@databricks/bundles/build";
 *
 * export const myJobMutator = jobMutator((bundle, job) => ({
 *   ...job,
 *   name: `${bundle.target}_${job.name}`,
 * }));
 * ```
 */
export function jobMutator(
  func: ((bundle: Bundle, job: Job) => Job) | ((job: Job) => Job)
): ResourceMutator<Job> {
  return {
    resourceType: Object as unknown as new () => Job,
    function: func as ((bundle: Bundle, resource: Job) => Job) | ((resource: Job) => Job),
  };
}

/**
 * Pipeline mutator decorator
 *
 * Creates a mutator that applies only to Pipeline resources.
 *
 * @example
 * ```typescript
 * import { pipelineMutator } from "@databricks/bundles/build";
 *
 * export const myPipelineMutator = pipelineMutator((bundle, pipeline) => ({
 *   ...pipeline,
 *   development: bundle.target !== "production",
 * }));
 * ```
 */
export function pipelineMutator(
  func: ((bundle: Bundle, pipeline: Pipeline) => Pipeline) | ((pipeline: Pipeline) => Pipeline)
): ResourceMutator<Pipeline> {
  return {
    resourceType: Object as unknown as new () => Pipeline,
    function: func as
      | ((bundle: Bundle, resource: Pipeline) => Pipeline)
      | ((resource: Pipeline) => Pipeline),
  };
}

/**
 * Schema mutator decorator
 *
 * Creates a mutator that applies only to Schema resources.
 *
 * @example
 * ```typescript
 * import { schemaMutator } from "@databricks/bundles/build";
 *
 * export const mySchemaMutator = schemaMutator((bundle, schema) => ({
 *   ...schema,
 *   name: `${bundle.target}_${schema.name}`,
 * }));
 * ```
 */
export function schemaMutator(
  func: ((bundle: Bundle, schema: Schema) => Schema) | ((schema: Schema) => Schema)
): ResourceMutator<Schema> {
  return {
    resourceType: Object as unknown as new () => Schema,
    function: func as
      | ((bundle: Bundle, resource: Schema) => Schema)
      | ((resource: Schema) => Schema),
  };
}

/**
 * Volume mutator decorator
 *
 * Creates a mutator that applies only to Volume resources.
 *
 * @example
 * ```typescript
 * import { volumeMutator } from "@databricks/bundles/build";
 *
 * export const myVolumeMutator = volumeMutator((bundle, volume) => ({
 *   ...volume,
 *   name: `${bundle.target}_${volume.name}`,
 * }));
 * ```
 */
export function volumeMutator(
  func: ((bundle: Bundle, volume: Volume) => Volume) | ((volume: Volume) => Volume)
): ResourceMutator<Volume> {
  return {
    resourceType: Object as unknown as new () => Volume,
    function: func as
      | ((bundle: Bundle, resource: Volume) => Volume)
      | ((resource: Volume) => Volume),
  };
}
