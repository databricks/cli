/**
 * Diagnostics system for errors and warnings
 *
 * Provides a structured way to collect and report errors and warnings
 * with location and path information.
 */

import { Location } from "./location.js";

/**
 * Severity level for diagnostic messages
 */
export enum Severity {
  WARNING = "warning",
  ERROR = "error",
}

/**
 * A single diagnostic item (error or warning)
 */
export interface Diagnostic {
  /**
   * Severity of the diagnostic
   */
  severity: Severity;

  /**
   * Short summary of the error or warning
   */
  summary: string;

  /**
   * Detailed explanation of the error or warning
   */
  detail?: string;

  /**
   * Path in databricks.yml where the error or warning occurred
   */
  path?: readonly string[];

  /**
   * Source code location where the error or warning occurred
   */
  location?: Location;
}

/**
 * Collection of diagnostic items (errors and warnings)
 *
 * Diagnostics are immutable and can be accumulated using the extend methods.
 *
 * @example
 * ```typescript
 * let diagnostics = new Diagnostics();
 * diagnostics = diagnostics.extend(someDiagnostics);
 * diagnostics = diagnostics.extend(moreDiagnostics);
 * ```
 */
export class Diagnostics {
  readonly items: readonly Diagnostic[];

  constructor(items: readonly Diagnostic[] = []) {
    this.items = items;
  }

  /**
   * Extend items with another diagnostics collection.
   *
   * This pattern allows accumulating errors and warnings.
   *
   * @example
   * ```typescript
   * let diagnostics = new Diagnostics();
   * diagnostics = diagnostics.extend(foo());
   * diagnostics = diagnostics.extend(bar());
   * ```
   */
  extend(other: Diagnostics): Diagnostics {
    return new Diagnostics([...this.items, ...other.items]);
  }

  /**
   * Extend items with diagnostics from a tuple result.
   *
   * This variant is useful when methods return a pair of value and diagnostics.
   *
   * @example
   * ```typescript
   * let diagnostics = new Diagnostics();
   * const [value, newDiagnostics] = diagnostics.extendTuple(foo());
   * diagnostics = newDiagnostics;
   * ```
   */
  extendTuple<T>(pair: [T, Diagnostics]): [T, Diagnostics] {
    const [value, otherDiagnostics] = pair;
    return [value, this.extend(otherDiagnostics)];
  }

  /**
   * Returns true if there is at least one error in diagnostics.
   */
  hasError(): boolean {
    return this.items.some((item) => item.severity === Severity.ERROR);
  }

  /**
   * Returns true if there is at least one warning in diagnostics.
   */
  hasWarning(): boolean {
    return this.items.some((item) => item.severity === Severity.WARNING);
  }

  /**
   * Returns all errors
   */
  getErrors(): readonly Diagnostic[] {
    return this.items.filter((item) => item.severity === Severity.ERROR);
  }

  /**
   * Returns all warnings
   */
  getWarnings(): readonly Diagnostic[] {
    return this.items.filter((item) => item.severity === Severity.WARNING);
  }

  /**
   * Creates an error diagnostic.
   */
  static createError(
    summary: string,
    options: {
      detail?: string;
      location?: Location;
      path?: readonly string[];
    } = {}
  ): Diagnostics {
    return new Diagnostics([
      {
        severity: Severity.ERROR,
        summary,
        detail: options.detail,
        location: options.location,
        path: options.path,
      },
    ]);
  }

  /**
   * Creates a warning diagnostic.
   */
  static createWarning(
    summary: string,
    options: {
      detail?: string;
      location?: Location;
      path?: readonly string[];
    } = {}
  ): Diagnostics {
    return new Diagnostics([
      {
        severity: Severity.WARNING,
        summary,
        detail: options.detail,
        location: options.location,
        path: options.path,
      },
    ]);
  }

  /**
   * Creates diagnostics from an exception.
   *
   * @param error - The error to create diagnostics from
   * @param summary - Short summary of the error
   * @param options - Additional options
   * @returns Diagnostics containing the error
   */
  static fromException(
    error: Error,
    summary: string,
    options: {
      location?: Location;
      path?: readonly string[];
      explanation?: string;
    } = {}
  ): Diagnostics {
    let detail = error.stack || error.message;

    if (options.explanation) {
      detail = `${detail}\n\n\x1b[0;36mExplanation:\x1b[0m ${options.explanation}`;
    }

    return new Diagnostics([
      {
        severity: Severity.ERROR,
        summary,
        detail,
        location: options.location,
        path: options.path,
      },
    ]);
  }

  /**
   * Converts diagnostics to a JSON-serializable format.
   */
  toJSON(): unknown[] {
    return this.items.map((item) => diagnosticToJSON(item));
  }
}

/**
 * Converts a single diagnostic to a JSON-serializable object.
 */
function diagnosticToJSON(diagnostic: Diagnostic): Record<string, unknown> {
  const result: Record<string, unknown> = {
    severity: diagnostic.severity,
    summary: diagnostic.summary,
  };

  if (diagnostic.detail !== undefined) {
    result.detail = diagnostic.detail;
  }

  if (diagnostic.path !== undefined) {
    result.path = diagnostic.path;
  }

  if (diagnostic.location !== undefined) {
    result.location = diagnostic.location.toJSON();
  }

  return result;
}
