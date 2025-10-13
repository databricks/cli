import { Bundle, Workspace } from "@databricks/bundles/core";
import { DatabaseCatalog as BaseDatabaseCatalog, type DatabaseCatalogParams } from "../../generated/database_catalogs/index.js";
import { Variable, type VariableOr } from "../core/variable.js";
import type { Database } from "./database.js";

export class DatabaseCatalog extends BaseDatabaseCatalog {
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

  get database_name() {
    return new Variable<string>(`resources.database_catalogs.${this.dabsName}.database_name`);
  }

  get database_instance_name() {
    return new Variable<string>(`resources.database_catalogs.${this.dabsName}.database_instance_name`);
  }
}