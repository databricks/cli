import { Bundle, Resources, Variable, variables } from "@databricks/bundles";
import { Volume } from "./components/volume.js";
import { App } from "./components/app.js";


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
    name: "explorer",
    description: "Interactive data exploration app",
    source_code_path: "./backend",
    env: {
      FOO: "BAR2",
    },
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