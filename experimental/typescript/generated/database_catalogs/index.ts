/**
 * DatabaseCatalog resource types for Databricks Asset Bundles
 *
 * Auto-generated from JSON Schema. Do not edit manually.
 */

import { Resource } from "../../src/core/resource.js";
import type { VariableOr } from "../../src/core/variable.js";

export interface DatabaseCatalogParams {
  create_database_if_not_exists?: VariableOr<boolean>;
  /**
   * The name of the DatabaseInstance housing the database.
   */
  database_instance_name: VariableOr<string>;
  /**
   * The name of the database (in a instance) associated with the catalog.
   */
  database_name: VariableOr<string>;
  /**
   * Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
   */
  lifecycle?: VariableOr<Lifecycle>;
  /**
   * The name of the catalog in UC.
   */
  name: VariableOr<string>;
  uid?: VariableOr<string>;
}

export class DatabaseCatalog extends Resource<DatabaseCatalogParams> {
  constructor(name: string, params: DatabaseCatalogParams) {
    super(name, params, "database_catalogs");
  }
}

export interface Lifecycle {
  /**
   * Lifecycle setting to prevent the resource from being destroyed.
   */
  prevent_destroy?: VariableOr<boolean>;
}
