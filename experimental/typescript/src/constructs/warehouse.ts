import { Variable } from "../core/variable.js";
import { SqlWarehouse as BaseSqlWarehouse, type SqlWarehouseParams } from "../../generated/sql_warehouses/index.js";

export class SqlWarehouse extends BaseSqlWarehouse {

  constructor(name: string, params: SqlWarehouseParams) {
    super(name, params);
  }

  get id() {
    return new Variable<string>(`resources.sql_warehouses.${this.dabsName}.id`);
  }
}