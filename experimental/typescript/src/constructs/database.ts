import { Bundle, Workspace } from "@databricks/bundles/core";
import { DatabaseInstance as BaseDatabaseInstance, type DatabaseInstanceParams } from "../../generated/database_instances/index.js";
import { Variable } from "../core/variable.js";
import type { VariableOr } from "../core/variable.js";

export class Database extends BaseDatabaseInstance {
  readonly resourceName: VariableOr<string>;

  constructor(name: string, bundle: Bundle, params: Partial<DatabaseInstanceParams> = {}) {
    const defaultParams: Partial<DatabaseInstanceParams> = {
      capacity: "CU_1",
    };

    if (!params.name) {
      params.name = name;
    }
    
    if (bundle.mode === "development") {
      params.name = `dev-${Workspace.currentUser.domainFriendlyName}-${params.name}`;
    }

    super(name, {...defaultParams, ...params, name: params.name});
    this.resourceName = params.name;
  }

  get name() {
    return new Variable<string>(`resources.database_instances.${this.dabsName}.name`);
  }
}