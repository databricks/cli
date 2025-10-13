import { DatabaseCatalog as BaseDatabaseCatalog, type DatabaseCatalogParams } from "../../generated/database_catalogs/index.js";
import { Variable } from "../core/variable.js";

export class DatabaseCatalog extends BaseDatabaseCatalog {
  constructor(name: string, params: DatabaseCatalogParams) {
    super(name, params);
  }

  get database_name() {
    return new Variable<string>(`resources.database_catalogs.${this.dabsName}.database_name`);
  }

  get database_instance_name() {
    return new Variable<string>(`resources.database_catalogs.${this.dabsName}.database_instance_name`);
  }
}