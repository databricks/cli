/**
 * Variable system for Databricks Asset Bundles
 *
 * Provides type-safe references to bundle variables that can be resolved at runtime.
 */

/**
 * Reference to a bundle variable.
 *
 * Variables are references to values defined in databricks.yml that can be
 * resolved during bundle deployment.
 *
 * @example
 * ```typescript
 * const warehouseId = new Variable<string>("var.warehouse_id");
 * // Serializes to: "${var.warehouse_id}"
 * ```
 */
export class Variable<T = unknown> {
  // Type parameter T is used for type safety at compile time
  // but not used at runtime
  private _typeMarker?: T;

  /**
   * Path to the variable, e.g. "var.my_variable"
   */
  readonly path: string;

  /**
   * Creates a new Variable reference.
   *
   * @param path - The variable path (e.g., "var.warehouse_id")
   */
  constructor(path: string) {
    this.path = path;
  }

  /**
   * Returns the variable path in the format "${path}"
   */
  get value(): string {
    return `\${${this.path}}`;
  }

  toString(): string {
    return this.value;
  }

  toJSON(): string {
    return this.value;
  }
}

/**
 * Helper function to create a Variable with type inference.
 *
 * @example
 * ```typescript
 * const name = variable<string>("var.job_name");
 * ```
 */
export function variable<T>(path: string): Variable<T> {
  return new Variable<T>(path);
}

/**
 * Type that can be either a Variable reference or a concrete value.
 *
 * @example
 * ```typescript
 * type JobName = VariableOr<string>;
 * const name1: JobName = "my-job";              // concrete value
 * const name2: JobName = variable("var.name");  // variable reference
 * ```
 */
export type VariableOr<T> = Variable<T> | T;

/**
 * Type that can be a Variable, a concrete value, or undefined.
 */
export type VariableOrOptional<T> = Variable<T> | T | undefined;

/**
 * Type for lists that can be:
 * 1. A variable: ${var.my_list}
 * 2. A regular list: [item1, item2, ...]
 * 3. A list with variables: [${var.item1}, item2, ...]
 * 4. A mix of variables and concrete values
 */
export type VariableOrList<T> = VariableOr<Array<VariableOr<T>>>;

/**
 * Type for dictionaries that can be:
 * 1. A variable: ${var.my_dict}
 * 2. A regular dict: {key: value, ...}
 * 3. A dict with variable values: {key: ${var.value}, ...}
 * 4. A mix of variables and concrete values
 */
export type VariableOrDict<T> = VariableOr<Record<string, VariableOr<T>>>;

/**
 * Interface for classes that can be decorated with @variables
 */
export interface VariablesClass {
  [key: string]: Variable<unknown>;
}

/**
 * Creates a proxy object where property access returns Variable instances.
 *
 * This is a TypeScript alternative to Python's @variables decorator.
 *
 * @example
 * ```typescript
 * interface MyVars {
 *   warehouse_id: Variable<string>;
 *   job_name: Variable<string>;
 * }
 *
 * const vars = variables<MyVars>("var");
 * console.log(vars.warehouse_id.value); // "${var.warehouse_id}"
 * ```
 *
 * @param prefix - The variable prefix (typically "var")
 * @returns A proxy that creates Variables on property access
 */
export function variables<T extends Record<string, Variable<unknown>>>(
  prefix = "var"
): T {
  return new Proxy({} as T, {
    get(_target, prop: string | symbol): Variable<unknown> {
      if (typeof prop === "symbol") {
        throw new Error("Symbol properties are not supported for variables");
      }
      return new Variable(`${prefix}.${prop}`);
    },
  });
}

/**
 * Helper to create a set of variable references from a list of names.
 *
 * @example
 * ```typescript
 * const vars = createVariables(["warehouse_id", "job_name"]);
 * console.log(vars.warehouse_id.value); // "${var.warehouse_id}"
 * ```
 */
export function createVariables<K extends string>(
  names: readonly K[],
  prefix = "var"
): Record<K, Variable<unknown>> {
  const result = {} as Record<K, Variable<unknown>>;
  for (const name of names) {
    result[name] = new Variable(`${prefix}.${name}`);
  }
  return result;
}

/**
 * Type guard to check if a value is a Variable.
 */
export function isVariable<T>(value: VariableOr<T>): value is Variable<T> {
  return value instanceof Variable;
}

/**
 * Extracts the variable path if the value is a Variable, otherwise returns undefined.
 */
export function getVariablePath<T>(value: VariableOr<T>): string | undefined {
  return isVariable(value) ? value.path : undefined;
}
