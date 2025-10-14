/**
 * High-level DatabaseCatalog construct with enhanced functionality.
 *
 * Extends the generated DatabaseCatalog resource with:
 * - Development mode name prefixing
 * - Variable references for catalog properties
 * - Simplified interface for database connection
 */
import { Bundle, Workspace } from "@databricks/bundles/core";
import { DatabaseCatalog as BaseDatabaseCatalog } from "../../generated/database_catalogs/index.js";
import { Variable, type VariableOr } from "../core/variable.js";
import type { Database } from "./database.js";

/**
 * Enhanced DatabaseCatalog construct.
 *
 * Features:
 * - Connects to a Database instance
 * - Adds dev-{user}- prefix to catalog name in development mode
 * - Provides variable references for database_name and database_instance_name
 *
 * @example
 * ```typescript
 * const postgres = new Database("postgres", bundle);
 * const catalog = new DatabaseCatalog("catalog", bundle, {
 *   database: postgres,
 *   database_name: "my_database",
 *   create_database_if_not_exists: true,
 * });
 *
 * // Use the catalog in other resources (e.g., Apps)
 * app.addResource(catalog, "CAN_CONNECT_AND_CREATE");
 * ```
 */
export class DatabaseCatalog extends BaseDatabaseCatalog {
  /**
   * Creates a new DatabaseCatalog construct.
   *
   * @param name - The name of the catalog in the bundle
   * @param bundle - The bundle context
   * @param params - Catalog parameters
   * @param params.database - The Database instance to connect to
   * @param params.database_name - The name of the database in the instance
   * @param params.create_database_if_not_exists - Whether to create the database if it doesn't exist
   * @param params.name - Optional catalog name (defaults to bundle resource name)
   * @param params.uid - Optional unique identifier
   */
  constructor(name: string, bundle: Bundle, params: {
    database: Database;
    name?: VariableOr<string>;
    database_name: VariableOr<string>;
    create_database_if_not_exists: boolean;
    uid?: string;
  }) {

    if (!params.name) {
      params.name = name;
    }

    if (bundle.mode === "development") {
      params.name = `dev-${Workspace.currentUser.domainFriendlyName}-${params.name}`;
    }

    super(name, {
      database_instance_name: params.database.name,
      database_name: params.database_name,
      create_database_if_not_exists: params.create_database_if_not_exists,
      name: params.name,
      uid: params.uid,
    });
  }

  /**
   * Returns a variable reference to the database name.
   *
   * This can be used to reference the database in other resources.
   */
  get database_name() {
    return new Variable<string>(`resources.database_catalogs.${this.dabsName}.database_name`);
  }

  /**
   * Returns a variable reference to the database instance name.
   *
   * This can be used to reference the database instance in other resources.
   */
  get database_instance_name() {
    return new Variable<string>(`resources.database_catalogs.${this.dabsName}.database_instance_name`);
  }
}
