/**
 * High-level App construct with enhanced functionality.
 *
 * Extends the generated App resource with:
 * - Automatic app.yml file generation
 * - Development mode name prefixing
 * - Simplified resource permission management
 */
import {
  App as BaseApp,
  type AppParams,
  type AppResource,
  type AppResourceDatabase,
  type AppResourceSqlWarehouse,
  type AppResourceUcSecurable,
} from "../../generated/apps/index.js";
import path from "path";
import { stringify } from "yaml";
import fs from "fs";
import { Bundle } from "../core/bundle.js";
import { Workspace } from "../core/workspace.js";
import { SqlWarehouse } from "./warehouse.js";
import { Volume } from "./volume.js";
import { DatabaseCatalog } from "./catalog.js";

/**
 * Additional parameters for App construct beyond the base AppParams.
 */
interface AppExtraParams {
  /**
   * Environment variables to include in app.yml
   */
  env?: { [key: string]: string };

  /**
   * Command array to include in app.yml
   */
  command?: string[];
}

/**
 * Enhanced App construct for Databricks Apps.
 *
 * Features:
 * - Automatically generates app.yml file in the source code path
 * - Adds dev-{user}- prefix to app name in development mode
 * - Provides type-safe resource permission management
 *
 * @example
 * ```typescript
 * const app = new App("my_app", bundle, {
 *   name: "bi",
 *   description: "BI application",
 *   source_code_path: "./backend",
 *   env: { LOG_LEVEL: "info" },
 *   command: ["python", "app.py"],
 * });
 *
 * // Grant app access to resources
 * app.addResource(warehouse, "CAN_USE");
 * app.addResource(catalog, "CAN_CONNECT_AND_CREATE");
 * ```
 */
export class App extends BaseApp {
  constructor(name: string, bundle: Bundle, params: AppParams & AppExtraParams) {
    const appYmlConfig: { env?: Array<{ name: string; value: string }>; command?: string[] } = {};

    const env: Record<string, string> = params.env || {};
    if (bundle.mode) {
      env["DEPLOY_MODE"] = bundle.mode;
    }

    const command = params.command || [];
    appYmlConfig.env = Object.entries(env).map(([name, value]) => ({
      name,
      value,
    }));
    appYmlConfig.command = command;

    if (typeof params.source_code_path === "string") {
      fs.writeFileSync(path.join(params.source_code_path, "app.yml"), stringify(appYmlConfig));
    }

    delete params.env;
    delete params.command;

    if (bundle.isDevelopment) {
      params.name = `dev-${Workspace.currentUser.domainFriendlyName.toString()}-${params.name.toString()}`;
    }
    super(name, params);
  }

  /**
   * Grants the app access to a SQL Warehouse.
   *
   * @param resource - The SQL Warehouse to grant access to
   * @param permission - The permission level (e.g., "CAN_USE")
   */
  addResource(resource: SqlWarehouse, permission: AppResourceSqlWarehouse["permission"]): void;

  /**
   * Grants the app access to a Database Catalog.
   *
   * @param resource - The Database Catalog to grant access to
   * @param permission - The permission level (e.g., "CAN_CONNECT_AND_CREATE")
   */
  addResource(resource: DatabaseCatalog, permission: AppResourceDatabase["permission"]): void;

  /**
   * Grants the app access to a Volume.
   *
   * @param resource - The Volume to grant access to
   * @param permission - The permission level (e.g., "READ_VOLUME", "WRITE_VOLUME")
   */
  addResource(resource: Volume, permission: AppResourceUcSecurable["permission"]): void;

  addResource(
    resource: DatabaseCatalog | SqlWarehouse | Volume,
    permission:
      | AppResourceDatabase["permission"]
      | AppResourceSqlWarehouse["permission"]
      | AppResourceUcSecurable["permission"]
  ): void {
    if (!this.data.resources) {
      this.data.resources = [];
    }

    const resources = this.data.resources as AppResource[];

    if (resource instanceof SqlWarehouse) {
      resources.push({
        name: resource.dabsName,
        sql_warehouse: {
          id: resource.id,
          permission: permission as AppResourceSqlWarehouse["permission"],
        },
      });
    } else if (resource instanceof DatabaseCatalog) {
      resources.push({
        name: resource.dabsName,
        database: {
          database_name: resource.database_name,
          instance_name: resource.database_instance_name,
          permission: permission as AppResourceDatabase["permission"],
        },
      });
    } else if (resource instanceof Volume) {
      resources.push({
        name: resource.dabsName,
        uc_securable: {
          securable_full_name: resource.fullName,
          securable_type: "VOLUME",
          permission: permission as AppResourceUcSecurable["permission"],
        },
      });
    }
  }
}
