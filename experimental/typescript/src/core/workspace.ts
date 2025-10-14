/**
 * Workspace variables for accessing Databricks workspace metadata.
 *
 * Provides type-safe access to workspace-level variables that are automatically
 * available in all bundles.
 */
import { Variable } from "./variable.js";

/**
 * Workspace provides access to workspace-level variables.
 *
 * These variables are automatically available and resolved by the Databricks CLI
 * during bundle deployment.
 *
 * @example
 * ```typescript
 * import { Workspace } from "@databricks/bundles/core";
 *
 * // Use workspace variables in resource configuration
 * const job = new Job("my_job", {
 *   name: `${Workspace.currentUser.shortName}-job`,
 *   // ...
 * });
 * ```
 */
export class Workspace {
  /**
   * The workspace host URL (e.g., "https://dbc-12345678-abcd.cloud.databricks.com")
   */
  static readonly host = new Variable<string>("workspace.host");

  /**
   * Current user information
   */
  static readonly currentUser = {
    /**
     * Domain-friendly username (e.g., "john_doe" from "john.doe@company.com")
     */
    domainFriendlyName: new Variable<string>("workspace.current_user.domain_friendly_name"),

    /**
     * Full username (e.g., "john.doe@company.com")
     */
    userName: new Variable<string>("workspace.current_user.user_name"),

    /**
     * Short username (e.g., "john_doe")
     */
    shortName: new Variable<string>("workspace.current_user.short_name"),
  };

  /**
   * Path to the current bundle file in the workspace
   */
  static readonly filePath = new Variable<string>("workspace.file_path");

  /**
   * Root path of the bundle in the workspace
   */
  static readonly rootPath = new Variable<string>("workspace.root_path");
}
