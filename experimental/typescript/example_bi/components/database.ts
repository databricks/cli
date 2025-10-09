import { DatabaseInstance as BaseDatabaseInstance, type DatabaseInstanceParams } from "@databricks/bundles/database_instances";
import { Variable, VariableOr } from "@databricks/bundles";

export class Database extends BaseDatabaseInstance {
  readonly resourceName: VariableOr<string>;

  constructor(name: string, params: DatabaseInstanceParams) {
    super(name, params);
    this.resourceName = params.name;
  }

  get name() {
    return new Variable<string>(`resources.database_instances.${this.dabsName}.name`);
  }
}