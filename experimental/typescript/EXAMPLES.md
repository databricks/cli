# Databricks Bundles TypeScript Examples

This document provides examples of how to use the TypeScript API for Databricks Asset Bundles.

## Using Constructs (High-Level Components)

Constructs provide reusable, high-level components with enhanced functionality:

```typescript
import { Bundle, Resources } from "@databricks/bundles";
import { App } from "@databricks/bundles/constructs/app";
import { SqlWarehouse } from "@databricks/bundles/constructs/warehouse";
import { Database } from "@databricks/bundles/constructs/database";
import { DatabaseCatalog } from "@databricks/bundles/constructs/catalog";
import { Volume } from "@databricks/bundles/constructs/volume";

export function loadResources(bundle: Bundle): Resources {
  const resources = new Resources();

  // Create a SQL warehouse
  const warehouse = new SqlWarehouse("warehouse", bundle);
  resources.addResource(warehouse);

  // Create a database instance
  const postgres = new Database("postgres", bundle);
  resources.addResource(postgres);

  // Create a catalog connected to the database
  const catalog = new DatabaseCatalog("catalog", bundle, {
    database: postgres,
    database_name: "postgres-catalog",
    create_database_if_not_exists: true,
  });
  resources.addResource(catalog);

  // Create a volume
  const volume = new Volume("landing-zone", bundle, {
    name: "landing-zone",
    catalog_name: "main",
    schema_name: "default",
    volume_type: "MANAGED",
  });
  resources.addResource(volume);

  // Create an app with resource permissions
  const app = new App("bi_app", bundle, {
    name: "bi",
    description: "BI example",
    source_code_path: "./backend",
  });

  // Grant app access to resources
  app.addResource(warehouse, "CAN_USE");
  app.addResource(catalog, "CAN_CONNECT_AND_CREATE");
  app.addResource(volume, "READ_VOLUME");

  resources.addResource(app);

  return resources;
}
```

## Using Variables

```typescript
import { Bundle, Resources, Variable, variables } from "@databricks/bundles";
import { Volume } from "@databricks/bundles/constructs/volume";

const vars = variables<{
  catalog: Variable<string>;
  schema: Variable<string>;
  warehouse_id: Variable<string>;
}>();

export function loadResources(bundle: Bundle): Resources {
  const resources = new Resources();

  const volume = new Volume("data-volume", bundle, {
    name: "data-volume",
    catalog_name: vars.catalog.value,
    schema_name: vars.schema.value,
    volume_type: "MANAGED",
  });
  resources.addResource(volume);

  return resources;
}
```

In your `databricks.yml`:

```yaml
variables:
  catalog:
    description: Catalog name
    default: main
  schema:
    description: Schema name
    default: default
  warehouse_id:
    description: SQL warehouse ID
```

## Using Low-Level Resource Classes

You can also use the generated resource classes directly:

```typescript
import { Bundle, Resources } from "@databricks/bundles";
import { Job } from "@databricks/bundles/jobs";
import { Pipeline } from "@databricks/bundles/pipelines";
import { Schema } from "@databricks/bundles/schemas";
import { Volume } from "@databricks/bundles/volumes";

export function loadResources(bundle: Bundle): Resources {
  const resources = new Resources();

  // Create a schema
  const schema = new Schema("my_schema", {
    name: `${bundle.target}_schema`,
    catalog_name: "main",
    comment: "Schema for data processing",
  });
  resources.addResource(schema);

  // Create a volume
  const volume = new Volume("my_volume", {
    name: `${bundle.target}_volume`,
    catalog_name: "main",
    schema_name: `${bundle.target}_schema`,
    volume_type: "MANAGED",
    comment: "Volume for file storage",
  });
  resources.addResource(volume);

  // Create a job
  const job = new Job("my_job", {
    name: "My TypeScript Job",
    tasks: [
      {
        task_key: "main",
        notebook_task: {
          notebook_path: "/path/to/notebook",
          base_parameters: {
            env: bundle.target,
          },
        },
        new_cluster: {
          spark_version: "13.3.x-scala2.12",
          node_type_id: "i3.xlarge",
          num_workers: 2,
        },
      },
    ],
    max_concurrent_runs: 1,
    timeout_seconds: 3600,
  });
  resources.addResource(job);

  // Create a pipeline
  const pipeline = new Pipeline("bronze_to_silver", {
    name: "Bronze to Silver Pipeline",
    catalog: "main",
    target: "silver",
    libraries: [
      { notebook: { path: "/pipelines/bronze_to_silver" } },
    ],
    clusters: [{
      label: "default",
      num_workers: 4,
      node_type_id: "i3.xlarge",
    }],
    continuous: false,
    development: bundle.target === "development",
    photon: true,
  });
  resources.addResource(pipeline);

  return resources;
}
```

## Dynamic Resource Creation

```typescript
import { Bundle, Resources } from "@databricks/bundles";
import { Job } from "@databricks/bundles/jobs";

// Configuration data
const configs = [
  { name: "etl_customers", table: "customers" },
  { name: "etl_orders", table: "orders" },
  { name: "etl_products", table: "products" },
];

export function loadResources(bundle: Bundle): Resources {
  const resources = new Resources();

  // Create a job for each configuration
  for (const config of configs) {
    const job = new Job(config.name, {
      name: `ETL ${config.table}`,
      tasks: [{
        task_key: "extract",
        notebook_task: {
          notebook_path: "/etl/extract",
          base_parameters: {
            table: config.table,
            target: bundle.target,
          },
        },
        job_cluster_key: "main",
      }],
      job_clusters: [{
        job_cluster_key: "main",
        new_cluster: {
          spark_version: "13.3.x-scala2.12",
          node_type_id: "i3.xlarge",
          num_workers: bundle.target === "production" ? 10 : 2,
        },
      }],
    });
    resources.addResource(job);
  }

  return resources;
}
```

## Multi-Task Job with Dependencies

```typescript
import { Bundle, Resources } from "@databricks/bundles";
import { Job } from "@databricks/bundles/jobs";

export function loadResources(bundle: Bundle): Resources {
  const resources = new Resources();

  const job = new Job("data_pipeline", {
    name: "Data Pipeline",
    tasks: [
      {
        task_key: "extract",
        notebook_task: {
          notebook_path: "/pipeline/extract",
        },
        new_cluster: {
          spark_version: "13.3.x-scala2.12",
          node_type_id: "i3.xlarge",
          num_workers: 2,
        },
      },
      {
        task_key: "transform",
        depends_on: [{ task_key: "extract" }],
        notebook_task: {
          notebook_path: "/pipeline/transform",
        },
        job_cluster_key: "transform_cluster",
      },
      {
        task_key: "load",
        depends_on: [{ task_key: "transform" }],
        sql_task: {
          warehouse_id: "abc123",
          file: {
            path: "/sql/load.sql",
          },
        },
      },
    ],
    job_clusters: [{
      job_cluster_key: "transform_cluster",
      new_cluster: {
        spark_version: "13.3.x-scala2.12",
        node_type_id: "i3.2xlarge",
        num_workers: 8,
      },
    }],
    email_notifications: {
      on_failure: ["team@company.com"],
      no_alert_for_skipped_runs: true,
    },
  });
  resources.addResource(job);

  return resources;
}
```

## Conditional Resource Creation

```typescript
import { Bundle, Resources } from "@databricks/bundles";
import { Job } from "@databricks/bundles/jobs";

export function loadResources(bundle: Bundle): Resources {
  const resources = new Resources();

  // Create different jobs based on target
  if (bundle.target === "production") {
    const job = new Job("prod_job", {
      name: "Production Job",
      tasks: [{
        task_key: "main",
        notebook_task: {
          notebook_path: "/prod/job",
        },
        new_cluster: {
          spark_version: "13.3.x-scala2.12",
          node_type_id: "i3.2xlarge",
          num_workers: 10,
        },
      }],
      schedule: {
        quartz_cron_expression: "0 0 * * * ?",
        timezone_id: "UTC",
      },
    });
    resources.addResource(job);
  } else {
    const job = new Job("dev_job", {
      name: "Development Job",
      tasks: [{
        task_key: "main",
        notebook_task: {
          notebook_path: "/dev/job",
        },
        new_cluster: {
          spark_version: "13.3.x-scala2.12",
          node_type_id: "i3.xlarge",
          num_workers: 1,
        },
      }],
    });
    resources.addResource(job);
  }

  return resources;
}
```

## Error Handling and Diagnostics

```typescript
import { Bundle, Resources } from "@databricks/bundles";
import { Job } from "@databricks/bundles/jobs";

export function loadResources(bundle: Bundle): Resources {
  const resources = new Resources();

  try {
    // Attempt to load configuration
    const config = loadConfig(bundle.target);

    const job = new Job("my_job", config);
    resources.addResource(job);
  } catch (error) {
    // Report errors as diagnostics
    resources.addDiagnosticError(
      "Failed to load job configuration",
      {
        detail: error instanceof Error ? error.message : String(error),
        path: ["resources", "jobs", "my_job"],
      }
    );
  }

  return resources;
}

function loadConfig(target: string) {
  // Load configuration based on target
  return {
    name: `Job for ${target}`,
    tasks: [],
  };
}
```

## Integration with databricks.yml

In your `databricks.yml`:

```yaml
bundle:
  name: my-bundle

variables:
  warehouse_id:
    description: SQL warehouse ID
    default: "abc123"
  catalog:
    description: Catalog name
    default: "main"
  schema:
    description: Schema name
    default: "default"

experimental:
  javascript:
    resources:
      - "resources:loadResources"

targets:
  development:
    mode: development

  production:
    mode: production
```

Then compile your TypeScript code and deploy:

```bash
# Compile TypeScript
npm run build

# Deploy with Databricks CLI
databricks bundle deploy --target development
```
