import { Variable } from "../core/variable.js";
import { SqlWarehouse as BaseSqlWarehouse, type SqlWarehouseParams } from "../../generated/sql_warehouses/index.js";
import { Bundle } from "../core/bundle.js";
import { Workspace } from "../core/workspace.js";

export class SqlWarehouse extends BaseSqlWarehouse {

  constructor(name: string, bundle: Bundle, params: SqlWarehouseParams = {}) {
    const defaultParams: SqlWarehouseParams = {
      warehouse_type: "PRO",
      enable_serverless_compute: true,
      cluster_size: "2X-Small",
      channel: {
        name: "CHANNEL_NAME_CURRENT",
      },
    };

    if (!params.name) {
      params.name = name;
    }
    
    if (bundle.mode === "development") {
      params.name = `dev-${Workspace.currentUser.domainFriendlyName}-${params.name}`;
    }

    super(name, {...defaultParams, ...params});
  }

  get id() {
    return new Variable<string>(`resources.sql_warehouses.${this.dabsName}.id`);
  }
}