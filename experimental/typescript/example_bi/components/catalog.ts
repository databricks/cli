import { DatabaseCatalogParams } from "@databricks/bundles/database_catalogs";
import { Variable, VariableOr } from "@databricks/bundles";
import { DatabaseCatalog as BaseDatabaseCatalog } from "@databricks/bundles/database_catalogs";

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