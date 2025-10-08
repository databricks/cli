import { type Bundle, Resources, createApp, Variable, variables } from "@databricks/bundles";

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

  // Add a Databricks App
  resources.addApp("my_data_app", createApp({
    name: `${bundle.target}-data-explorer`,
    description: "Interactive data exploration app",

    // Source code location in workspace
    source_code_path: "/Workspace/Users/fabian.jakobs@databricks.com/dev-data-explorer",

    // App resources (permissions to access other Databricks resources)
    resources: [
      {
        name: "warehouse",
        description: `SQL warehouse for querying data ${bundle.target} ${catalogName} ${schemaName}`,
        sql_warehouse: {
          id: `${warehouseId}`,
          permission: "CAN_USE",
        },
      }
    ],
  }));

  return resources;
}