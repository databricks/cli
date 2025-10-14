/**
 * Bundle context and configuration
 *
 * Provides access to bundle metadata and variable resolution during
 * resource loading and mutation.
 */

import { isVariable, type VariableOr, type VariableOrList } from "./variable.js";

/**
 * Bundle contains information about a bundle accessible in functions
 * loading and mutating resources.
 */
export class Bundle {
  /**
   * Selected target where the bundle is being loaded.
   * E.g.: 'development', 'staging', or 'production'.
   */
  readonly target: string;

  /**
   * Values of bundle variables resolved for selected target.
   * Bundle variables are defined in databricks.yml.
   *
   * For accessing variables as structured data, use resolveVariable().
   *
   * @example
   * ```yaml
   * variables:
   *   default_dbr_version:
   *     description: Default version of Databricks Runtime
   *     default: "14.3.x-scala2.12"
   * ```
   */
  readonly variables: Readonly<Record<string, unknown>>;

  /**
   * Bundle mode (e.g., "development", "production")
   * Set via the `mode` field in databricks.yml target configuration.
   */
  readonly mode?: string;

  /**
   * Bundle name from databricks.yml
   */
  readonly name: string;

  constructor(options: {
    target: string;
    variables?: Record<string, unknown>;
    mode?: string;
    name: string;
  }) {
    this.target = options.target;
    this.variables = Object.freeze({ ...options.variables });
    this.mode = options.mode;
    this.name = options.name;
  }

  /**
   * Returns true if the bundle is in development mode.
   *
   * @returns true if mode is "development", false otherwise
   *
   * @example
   * ```typescript
   * if (bundle.isDevelopment) {
   *   // Add dev-specific configuration
   * }
   * ```
   */
  get isDevelopment(): boolean {
    return this.mode === "development";
  }

  /**
   * Returns true if the bundle is in production mode.
   *
   * @returns true if mode is "production", false otherwise
   *
   * @example
   * ```typescript
   * if (bundle.isProduction) {
   *   // Add production-specific configuration
   * }
   * ```
   */
  get isProduction(): boolean {
    return this.mode === "production";
  }

  /**
   * Resolve a variable to its value.
   *
   * If the value is a variable, it will be resolved and returned.
   * Otherwise, the value will be returned as is.
   *
   * @throws Error if the variable is not found or cannot be resolved
   */
  resolveVariable<T>(variable: VariableOr<T>): T {
    if (!isVariable(variable)) {
      return variable;
    }

    const VAR_PREFIX = "var";

    if (!variable.path.startsWith(`${VAR_PREFIX}.`)) {
      throw new Error(`You can only get values of variables starting with '${VAR_PREFIX}.*'`);
    }

    const variableName = variable.path.slice(VAR_PREFIX.length + 1);

    if (!(variableName in this.variables)) {
      throw new Error(
        `Can't find '${variableName}' variable. Did you define it in databricks.yml?`
      );
    }

    const value = this.variables[variableName];

    // Check if the value is a nested variable reference
    const nestedPath = unwrapVariablePath(value);
    if (nestedPath) {
      // The value itself is a variable reference like "${var.other}"
      // Check if the variable type allows nested variables
      throw new Error(
        `Failed to resolve '${variableName}' because it refers to another variable '${nestedPath}'. ` +
          `Consider using Variable<VariableOr<T>> for nested variable support.`
      );
    }

    return value as T;
  }

  /**
   * Resolve a list variable to its value.
   *
   * If the value is a variable, or the list items are variables,
   * they will be resolved and returned.
   * Otherwise, the value will be returned as is.
   */
  resolveVariableList<T>(variable: VariableOrList<T>): T[] {
    const resolved = this.resolveVariable(variable);

    if (!Array.isArray(resolved)) {
      throw new Error("Expected a list value");
    }

    return resolved.map((item) => this.resolveVariable(item));
  }
}

/**
 * Checks if a value is a string that looks like a variable reference.
 * Returns the variable path if found, undefined otherwise.
 *
 * @param value - Value to check
 * @returns Variable path (e.g., "var.foo") or undefined
 */
function unwrapVariablePath(value: unknown): string | undefined {
  if (typeof value !== "string") {
    return undefined;
  }

  // Check if it matches ${var.something}
  const match = value.match(/^\$\{(.+)\}$/);
  return match ? match[1] : undefined;
}
