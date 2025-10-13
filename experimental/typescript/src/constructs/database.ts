import { DatabaseInstance as BaseDatabaseInstance, type DatabaseInstanceParams } from "../../generated/database_instances/index.js";
import { Variable } from "../core/variable.js";
import type { VariableOr } from "../core/variable.js";

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