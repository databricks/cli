/**
 * High-level SqlWarehouse construct with enhanced functionality.
 *
 * Extends the generated SqlWarehouse resource with:
 * - Sensible defaults for serverless warehouses
 * - Development mode name prefixing
 * - Variable reference for warehouse ID
 */
import { Variable } from "../core/variable.js";
import {
  SqlWarehouse as BaseSqlWarehouse,
  type SqlWarehouseParams,
} from "../../generated/sql_warehouses/index.js";
import { Bundle } from "../core/bundle.js";
import { Workspace } from "../core/workspace.js";

/**
 * Enhanced SQL Warehouse construct.
 *
 * Features:
 * - Defaults to serverless PRO warehouse with 2X-Small cluster size
 * - Adds dev-{user}- prefix to warehouse name in development mode
 * - Provides `id` property as a variable reference
 *
 * @example
 * ```typescript
 * const warehouse = new SqlWarehouse("my_warehouse", bundle, {
 *   cluster_size: "Medium",
 * });
 *
 * // Use the warehouse ID in other resources
 * const job = new Job("my_job", {
 *   tasks: [{
 *     sql_task: {
 *       warehouse_id: warehouse.id,
 *       // ...
 *     }
 *   }]
 * });
 * ```
 */
export class SqlWarehouse extends BaseSqlWarehouse {
  /**
   * Creates a new SqlWarehouse construct.
   *
   * @param name - The name of the warehouse in the bundle
   * @param bundle - The bundle context
   * @param params - SQL warehouse parameters (optional, uses defaults)
   */
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

    if (bundle.isDevelopment) {
      params.name = `dev-${Workspace.currentUser.domainFriendlyName}-${params.name}`;
    }

    super(name, { ...defaultParams, ...params });
  }

  /**
   * Returns a variable reference to the warehouse ID.
   *
   * This can be used to reference the warehouse in other resources.
   */
  get id() {
    return new Variable<string>(`resources.sql_warehouses.${this.dabsName}.id`);
  }
}
