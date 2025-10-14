/**
 * Transform system for converting values to JSON
 *
 * Provides utilities for serializing resources and other objects to JSON format
 * that can be consumed by the Databricks CLI.
 */

import { Variable } from "./variable.js";

/**
 * Transforms a value to a JSON-serializable format.
 *
 * Handles special cases like Variables, nested objects, arrays, and primitives.
 * Implements "omitempty" semantics by skipping null, empty arrays, and empty objects.
 *
 * @param value - The value to transform
 * @returns JSON-serializable value
 */
export function transformToJSON(
  value: unknown
): string | boolean | number | null | Record<string, unknown> | unknown[] {
  // Handle null and undefined
  if (value === null || value === undefined) {
    return null;
  }

  // Handle Variable instances
  if (value instanceof Variable) {
    return value.value;
  }

  // Handle primitives
  if (typeof value === "string" || typeof value === "boolean" || typeof value === "number") {
    return value;
  }

  // Handle arrays
  if (Array.isArray(value)) {
    return value.map((item) => transformToJSON(item));
  }

  // Handle plain objects and class instances
  if (typeof value === "object") {
    return transformObjectToJSON(value as Record<string, unknown>);
  }

  throw new Error(`Cannot serialize value: ${String(value)}`);
}

/**
 * Transforms an object to a JSON-serializable format.
 *
 * Implements "omitempty" semantics:
 * - Skips properties with null or undefined values
 * - Skips properties with empty arrays
 * - Skips properties with empty objects
 *
 * @param obj - The object to transform
 * @returns JSON-serializable object
 */
function transformObjectToJSON(obj: Record<string, unknown>): Record<string, unknown> {
  const result: Record<string, unknown> = {};

  // Get all enumerable properties
  const keys = Object.keys(obj).sort(); // Sort for stable output

  for (const key of keys) {
    const value = obj[key];
    const transformed = transformToJSON(value);

    // Implement omitempty semantics
    if (shouldIncludeValue(transformed)) {
      result[key] = transformed;
    }
  }

  return result;
}

/**
 * Determines if a value should be included in JSON output.
 *
 * Implements "omitempty" semantics:
 * - Include non-null primitives
 * - Include non-empty arrays
 * - Include non-empty objects
 * - Skip null, undefined, empty arrays, and empty objects
 */
function shouldIncludeValue(value: unknown): boolean {
  if (value === null || value === undefined) {
    return false;
  }

  if (Array.isArray(value) && value.length === 0) {
    return false;
  }

  if (typeof value === "object" && Object.keys(value).length === 0) {
    return false;
  }

  return true;
}
