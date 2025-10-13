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

interface AppExtraParams {
  env?: { [key: string]: string };
  command?: string[];
}

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
      value: value as string,
    }));
    appYmlConfig.command = command;

    if (typeof params.source_code_path === "string") {
      fs.writeFileSync(path.join(params.source_code_path, "app.yml"), stringify(appYmlConfig));
    }

    delete params.env;
    delete params.command;

    if (bundle.mode === "development") {
      params.name = `dev-${Workspace.currentUser.domainFriendlyName}-${params.name}`;
    }
    super(name, params);
  }

  addResource(resource: SqlWarehouse, permission: AppResourceSqlWarehouse["permission"]): void;
  addResource(resource: DatabaseCatalog, permission: AppResourceDatabase["permission"]): void;
  addResource(resource: Volume, permission: AppResourceUcSecurable["permission"]): void;

  addResource(
    resource: DatabaseCatalog | SqlWarehouse | Volume,
    permission: AppResourceDatabase["permission"] | AppResourceSqlWarehouse["permission"] | AppResourceUcSecurable["permission"]
  ) {
    if (!this.data.resources) {
      this.data.resources = [];
    }

    const resoues = this.data.resources as AppResource[];

    if (resource instanceof SqlWarehouse) {
      resoues.push({
        name: resource.dabsName,
        sql_warehouse: {
          id: resource.id,
          permission: permission as AppResourceSqlWarehouse["permission"],
        },
      });
    } else if (resource instanceof DatabaseCatalog) {
      resoues.push({
        name: resource.dabsName,
        database: {
          database_name: resource.database_name,
          instance_name: resource.database_instance_name,
          permission: permission as AppResourceDatabase["permission"],
        },
      });
    } else if (resource instanceof Volume) {
      resoues.push({
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
