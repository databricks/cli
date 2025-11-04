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
  type AppResourceSecret,
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
import { DatabaseCatalog } from "./database_catalog.js";
import { SecretScope } from "./secret_scope.js";
import { Resource } from "../core/resource.js";
import { DatabaseInstance } from "./database_instance.js";

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
  private appYmlConfig: {
    env?: Array<{ name: string; value?: string; valueFrom?: string }>;
    command?: string[];
  } = {};

  constructor(name: string, bundle: Bundle, params: AppParams & AppExtraParams) {
    if (bundle.isDevelopment) {
      params.name = `dev-${Workspace.currentUser.domainFriendlyName.toString()}-${params.name.toString()}`;
    }
    super(name, params);

    const env = params.env || {};
    if (bundle.mode) {
      env["DEPLOY_MODE"] = bundle.mode;
    }
    this.appYmlConfig.env = Object.entries(env).map(([name, value]) => ({
      name,
      value: value.toString(),
    }));
    this.appYmlConfig.command = params.command || [];
  }

  override async onBeforeCreate(): Promise<void> {
    if (typeof this.data.source_code_path === "string") {
      await fs.promises.writeFile(
        path.join(this.data.source_code_path, "app.yml"),
        stringify(this.appYmlConfig)
      );
    }
  }

  /**
   * Sets an environment variable for the app.
   *
   * @param name - The name of the environment variable
   * @param value - The value of the environment variable. Can be a string or a Resource.
   */
  setEnvironment(name: string, value: string | Resource<unknown>): void {
    if (!this.appYmlConfig.env) {
      this.appYmlConfig.env = [];
    }
    if (value instanceof Resource) {
      this.appYmlConfig.env.push({
        name,
        valueFrom: value.dabsName,
      });
    } else {
      this.appYmlConfig.env.push({
        name,
        value: value,
      });
    }
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
   * Grants the app access to a Database Instance.
   *
   * @param resource - The Database Instance to grant access to
   * @param permission - The permission level (e.g., "CAN_CONNECT_AND_CREATE")
   */
  addResource(
    resource: DatabaseInstance,
    permission: AppResourceDatabase["permission"],
    databaseName?: string
  ): void;

  /**
   * Grants the app access to a Volume.
   *
   * @param resource - The Volume to grant access to
   * @param permission - The permission level (e.g., "READ_VOLUME", "WRITE_VOLUME")
   */
  addResource(resource: Volume, permission: AppResourceUcSecurable["permission"]): void;

  /**
   * Grants the app access to a Secret Scope.
   *
   * @param resource - The Secret Scope to grant access to
   * @param permission - The permission level (e.g., "READ", "WRITE", "MANAGE")
   */
  addResource(
    resource: SecretScope,
    permission: AppResourceSecret["permission"],
    key: string
  ): void;

  addResource(
    resource: DatabaseCatalog | DatabaseInstance | SqlWarehouse | Volume | SecretScope,
    permission:
      | AppResourceDatabase["permission"]
      | AppResourceSqlWarehouse["permission"]
      | AppResourceUcSecurable["permission"]
      | AppResourceSecret["permission"],
    arg?: string
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
    } else if (resource instanceof DatabaseInstance && arg) {
      const databaseName = arg;
      resources.push({
        name: resource.dabsName,
        database: {
          database_name: databaseName,
          instance_name: resource.name,
          permission: permission as AppResourceDatabase["permission"],
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
    } else if (resource instanceof SecretScope && arg) {
      const key = arg;
      resources.push({
        name: resource.dabsName,
        secret: {
          permission: permission as AppResourceSecret["permission"],
          key,
          scope: resource.name,
        },
      });
    }
  }
}
