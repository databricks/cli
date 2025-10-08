import { type Bundle, Resources, Variable, variables } from "@databricks/bundles";
import { App as BaseApp, type AppParams } from "@databricks/bundles/apps";
import { Volume as BaseVolume, type VolumeParams } from "@databricks/bundles/volumes";
import { stringify } from 'yaml'

import path from "path";
import fs from "fs";

const vars = variables<{
  warehouse_id: Variable;
  catalog: Variable;
  schema: Variable;
}>();

export function loadResources(bundle: Bundle): Resources {
  const resources = new Resources();

  // Create variables for configuration
  const warehouseId = vars.warehouse_id;
  const catalogName = vars.catalog;
  const schemaName = vars.schema;

  const volume = new Volume({
    name: `${bundle.target}-landing_zone`,
    comment: "Landing zone for the data",
    catalog_name: catalogName,
    schema_name: schemaName,
    volume_type: "MANAGED",
  });
  resources.addResource("landing_zone", volume);

  // Add a Databricks App
  resources.addResource("my_data_app", createServer(bundle, {
    name: `${bundle.target}-data-explorer`,
    description: "Interactive data exploration app",

    source_code_path: "./backend",
    env: {
      FOO: "BAR2",
    },
    command: ["npm", "start"],

    // App resources (permissions to access other Databricks resources)
    resources: [
      {
        name: "warehouse",
        description: `SQL warehouse for querying data ${bundle.target}`,
        sql_warehouse: {
          id: warehouseId,
          permission: "CAN_USE",
        },
      },
      {
        name: "volume",
        description: `Volume for the data ${bundle.target}`,
        uc_securable: {
          securable_full_name: volume.fullName,
          securable_type: "VOLUME",
          permission: "READ_VOLUME",
        },
      }
    ],
  }));

  return resources;
}

function createServer(_bundle: Bundle, config: AppParams & {
  env?: { [key: string]: string },
  command?: string[],
}): BaseApp {
  
  const appYmlConfig: { env?: Array<{ name: string; value: string }>, command?: string[] } = {};
  if (config.env || config.command) {
    appYmlConfig.env = Object.entries(config.env || {}).map(([name, value]) => ({ name, value: value as string }));
    appYmlConfig.command = config.command;
  }
  
  if (typeof config.source_code_path === "string") {
    fs.writeFileSync(path.join(config.source_code_path, "app.yml"), stringify(appYmlConfig));
  }

  delete(config.env);
  delete(config.command);

  const app = new BaseApp(config);

  return app;
}

export class Volume extends BaseVolume {
  constructor(params: VolumeParams) {
    super(params);
  }

  get fullName() {
    return `${this.data.catalog_name}.${this.data.schema_name}.${this.data.name}`;
  }
}