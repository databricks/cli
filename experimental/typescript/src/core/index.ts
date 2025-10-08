/**
 * Core framework for Databricks Asset Bundles
 *
 * This module exports the core types and utilities for working with bundles:
 * - Bundle: Context and configuration
 * - Resources: Resource container
 * - Variables: Variable resolution
 * - Diagnostics: Error and warning handling
 * - Location: Source code location tracking
 * - Transform: JSON serialization utilities
 */

// Variable system
export {
  Variable,
  variable,
  variables,
  createVariables,
  isVariable,
  getVariablePath,
  type VariableOr,
  type VariableOrOptional,
  type VariableOrList,
  type VariableOrDict,
  type VariablesClass,
} from "./variable.js";

// Location tracking
export { Location } from "./location.js";

// Diagnostics
export { Diagnostics, Severity, type Diagnostic } from "./diagnostics.js";

// Bundle context
export { Bundle } from "./bundle.js";

// Resources
export type { Resource } from "./resource.js";
export { isResource } from "./resource.js";
export { Resources } from "./resources.js";
export type { Job, Pipeline, Schema, Volume } from "./resources.js";

// Transform utilities
export { transformToJSON, deepClone } from "./transform.js";