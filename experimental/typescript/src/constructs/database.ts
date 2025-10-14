/**
 * High-level Database (DatabaseInstance) construct with enhanced functionality.
 *
 * Extends the generated DatabaseInstance resource with:
 * - Sensible defaults for database capacity
 * - Development mode name prefixing
 * - Variable reference for database name
 */
import { Bundle, Workspace } from "@databricks/bundles/core";
import {
  DatabaseInstance as BaseDatabaseInstance,
  type DatabaseInstanceParams,
} from "../../generated/database_instances/index.js";
import { Variable } from "../core/variable.js";
import type { VariableOr } from "../core/variable.js";

/**
 * Enhanced Database (DatabaseInstance) construct.
 *
 * Features:
 * - Defaults to CU_1 capacity
 * - Adds dev-{user}- prefix to database name in development mode
 * - Provides `name` property as a variable reference
 *
 * @example
 * ```typescript
 * const postgres = new Database("postgres", bundle, {
 *   capacity: "CU_2",
 * });
 *
 * // Use the database name in other resources
 * const catalog = new DatabaseCatalog("catalog", bundle, {
 *   database: postgres,
 *   database_name: "my_catalog",
 * });
 * ```
 */
export class Database extends BaseDatabaseInstance {
  /**
   * The resource name (possibly a variable reference)
   */
  readonly resourceName: VariableOr<string>;

  /**
   * Creates a new Database construct.
   *
   * @param name - The name of the database in the bundle
   * @param bundle - The bundle context
   * @param params - Database instance parameters (optional, uses defaults)
   */
  constructor(name: string, bundle: Bundle, params: Partial<DatabaseInstanceParams> = {}) {
    const defaultParams: Partial<DatabaseInstanceParams> = {
      capacity: "CU_1",
    };

    if (!params.name) {
      params.name = name;
    }

    if (bundle.isDevelopment) {
      params.name = `dev-${Workspace.currentUser.domainFriendlyName.toString()}-${params.name.toString()}`;
    }

    super(name, { ...defaultParams, ...params, name: params.name });
    this.resourceName = params.name;
  }

  /**
   * Returns a variable reference to the database instance name.
   *
   * This can be used to reference the database in other resources.
   */
  get name(): Variable<string> {
    return new Variable<string>(`resources.database_instances.${this.dabsName}.name`);
  }
}
