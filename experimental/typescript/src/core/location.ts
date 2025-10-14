/**
 * Location tracking for source code positions
 *
 * Used to track where resources and configurations are defined in source code
 * for better error messages and diagnostics.
 */

import path from "path";

/**
 * Represents a location in source code.
 *
 * Locations are used to provide context for diagnostics and errors,
 * showing where in the code a particular resource or configuration was defined.
 */
export class Location {
  /**
   * Path to the source file
   */
  readonly file: string;

  /**
   * Line number in the file (1-based, must be >= 1)
   */
  readonly line?: number;

  /**
   * Column number in the line (1-based, must be >= 1)
   */
  readonly column?: number;

  constructor(options: { file: string; line?: number; column?: number }) {
    if (options.line !== undefined && options.line < 1) {
      throw new Error(`Line number must be greater than 0, got ${options.line}`);
    }

    if (options.column !== undefined && options.column < 1) {
      throw new Error(`Column number must be greater than 0, got ${options.column}`);
    }

    this.file = options.file;
    this.line = options.line;
    this.column = options.column;
  }

  /**
   * Captures the location of a function.
   *
   * This is useful for creating diagnostics for decorated functions or callbacks.
   * Note: In JavaScript/TypeScript, we can get the function's location from stack traces
   * but it's less reliable than Python's inspect module.
   *
   * @param fn - The function to get location from
   * @returns Location if available, undefined otherwise
   */
  static fromFunction(_fn: Function): Location | undefined {
    // In JavaScript, we can try to parse the function's toString() or use stack traces
    // but it's not as reliable as Python's inspect.
    // For now, we'll return undefined and rely on explicit location tracking.
    // This can be enhanced with source map support in the future.
    return undefined;
  }

  /**
   * Captures the current location from the call stack.
   *
   * @param depth - How many stack frames to go up (0 = caller of this function)
   * @returns Location from the stack trace
   */
  static fromStack(depth = 0): Location | undefined {
    const error = new Error();
    const stack = error.stack;

    if (!stack) {
      return undefined;
    }

    const lines = stack.split("\n");
    // Skip "Error" line and the fromStack frame itself, then add depth
    const targetLine = lines[2 + depth];

    if (!targetLine) {
      return undefined;
    }

    // Parse stack trace line
    // Format is typically: "    at functionName (file:line:column)"
    const match =
      targetLine.match(/\((.+):(\d+):(\d+)\)/) || targetLine.match(/at (.+):(\d+):(\d+)/);

    if (!match) {
      return undefined;
    }

    const file = match[1] || "";
    const line = parseInt(match[2] || "0", 10);
    const column = parseInt(match[3] || "0", 10);

    return new Location({
      file: relativizePath(file),
      line: line > 0 ? line : undefined,
      column: column > 0 ? column : undefined,
    });
  }

  /**
   * Converts the location to a plain object suitable for JSON serialization.
   */
  toJSON(): Record<string, unknown> {
    const result: Record<string, unknown> = {
      file: this.file,
    };

    if (this.line !== undefined) {
      result.line = this.line;
    }

    if (this.column !== undefined) {
      result.column = this.column;
    }

    return result;
  }

  /**
   * Returns a string representation of the location.
   */
  toString(): string {
    let result = this.file;
    if (this.line !== undefined) {
      result += `:${this.line}`;
      if (this.column !== undefined) {
        result += `:${this.column}`;
      }
    }
    return result;
  }
}

/**
 * Makes a file path relative to the current working directory if possible.
 *
 * @param filePath - The absolute file path
 * @returns Relative path if under cwd, otherwise the original path
 */
function relativizePath(filePath: string): string {
  if (!path.isAbsolute(filePath)) {
    return filePath;
  }

  const cwd = process.cwd();
  if (filePath.startsWith(cwd)) {
    return path.relative(cwd, filePath);
  }

  return filePath;
}
