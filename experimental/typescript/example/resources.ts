import { Bundle, Resources, Variable, variables, Workspace } from "@databricks/bundles";
import { App as BaseApp, type AppParams } from "@databricks/bundles/apps";
import { Volume as BaseVolume, type VolumeParams } from "@databricks/bundles/volumes";
import { stringify } from 'yaml'

import path from "path";
import fs from "fs";

const vars = variables<{
  warehouse_id: Variable<string>;
  catalog: Variable<string>;
  schema: Variable<string>;
}>();

export function loadResources(bundle: Bundle): Resources {
  const resources = new Resources();

  const volume = new Volume(bundle, {
    name: `landing-zone`,
    comment: "Landing zone for the data",
    catalog_name: vars.catalog.value,
    schema_name: vars.schema.value,
    volume_type: "MANAGED",
  });
  resources.addResource("landing_zone", volume);

  const app = new App(bundle, {
    name: `explorer`,
    description: "Interactive data exploration app",
    source_code_path: "./backend",
    env: {
      FOO: "BAR2",
    },
    command: ["npm", "start"],
    resources: [
      {
        name: "warehouse",
        description: `SQL warehouse for querying data ${bundle.target}`,
        sql_warehouse: {
          id: vars.warehouse_id.value,
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
  });
  resources.addResource("my_data_app", app);

  return resources;
}

interface AppExtraParams {
  env?: { [key: string]: string },
  command?: string[],
}

class App extends BaseApp {
  constructor(bundle: Bundle, params: AppParams & AppExtraParams) {
    const appYmlConfig: { env?: Array<{ name: string; value: string }>, command?: string[] } = {};
    if (params.env || params.command) {
      appYmlConfig.env = Object.entries(params.env || {}).map(([name, value]) => ({ name, value: value as string }));
      appYmlConfig.command = params.command;
    }
    
    if (typeof params.source_code_path === "string") {
      fs.writeFileSync(path.join(params.source_code_path, "app.yml"), stringify(appYmlConfig));
    }
  
    delete(params.env);
    delete(params.command);

    if (bundle.mode === "development") {
      params.name = `dev-${Workspace.currentUser.domainFriendlyName}-${params.name}`;
    }
    super(params);
  }
}

export class Volume extends BaseVolume {
  constructor(bundle: Bundle, params: VolumeParams) {
    if (bundle.mode === "development") {
      params.name = `dev-${Workspace.currentUser.domainFriendlyName}-${params.name}`;
    }
    super(params);
  }

  get fullName() {
    return `${this.data.catalog_name}.${this.data.schema_name}.${this.data.name}`;
  }
}