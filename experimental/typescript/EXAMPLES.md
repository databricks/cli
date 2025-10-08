# Databricks Bundles TypeScript Examples

This document provides examples of how to use the TypeScript API for Databricks Asset Bundles.

## Basic Job Definition

```typescript
import { Bundle, Resources, createJob, createTask, variable } from "@databricks/bundles";

export function loadResources(bundle: Bundle): Resources {
  const resources = new Resources();

  resources.addJob("my_job", createJob({
    name: "My TypeScript Job",
    tasks: [
      createTask({
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
      }),
    ],
    max_concurrent_runs: 1,
    timeout_seconds: 3600,
  }));

  return resources;
}
```

## Job with Variables

```typescript
import { Bundle, Resources, createJob, variables, variable } from "@databricks/bundles";

// Define variables with type safety
interface MyVars {
  warehouse_id: Variable<string>;
  job_name: Variable<string>;
  num_workers: Variable<number>;
}

const vars = variables<MyVars>();

export function loadResources(bundle: Bundle): Resources {
  const resources = new Resources();

  // Resolve variables from databricks.yml
  const warehouseId = bundle.resolveVariable(vars.warehouse_id);
  const jobName = bundle.resolveVariable(vars.job_name);

  resources.addJob("sql_job", createJob({
    name: jobName,
    tasks: [{
      task_key: "run_query",
      sql_task: {
        warehouse_id: warehouseId,
        query: {
          query_id: "abc123",
        },
      },
    }],
  }));

  return resources;
}
```

## Dynamic Job Creation

```typescript
import { Bundle, Resources, createJob } from "@databricks/bundles";

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
    resources.addJob(config.name, createJob({
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
    }));
  }

  return resources;
}
```

## Pipeline Definition

```typescript
import { Bundle, Resources, createPipeline } from "@databricks/bundles";

export function loadResources(bundle: Bundle): Resources {
  const resources = new Resources();

  resources.addPipeline("bronze_to_silver", createPipeline({
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
  }));

  return resources;
}
```

## Schema and Volume Definition

```typescript
import { Bundle, Resources, createSchema, createVolume } from "@databricks/bundles";

export function loadResources(bundle: Bundle): Resources {
  const resources = new Resources();

  // Create a schema
  resources.addSchema("my_schema", createSchema({
    name: `${bundle.target}_schema`,
    catalog_name: "main",
    comment: "Schema for data processing",
  }));

  // Create a volume
  resources.addVolume("my_volume", createVolume({
    name: `${bundle.target}_volume`,
    catalog_name: "main",
    schema_name: `${bundle.target}_schema`,
    volume_type: "MANAGED",
    comment: "Volume for file storage",
  }));

  return resources;
}
```

## Multi-Task Job with Dependencies

```typescript
import { Bundle, Resources, createJob } from "@databricks/bundles";

export function loadResources(bundle: Bundle): Resources {
  const resources = new Resources();

  resources.addJob("data_pipeline", createJob({
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
  }));

  return resources;
}
```

## Using Variables Passed as References

```typescript
import { Bundle, Resources, createJob, variable } from "@databricks/bundles";

export function loadResources(bundle: Bundle): Resources {
  const resources = new Resources();

  // Variables can be passed directly without resolution
  // The CLI will resolve them during deployment
  resources.addJob("my_job", createJob({
    name: variable<string>("var.job_name"),
    tasks: [{
      task_key: "main",
      notebook_task: {
        notebook_path: variable<string>("var.notebook_path"),
      },
      existing_cluster_id: variable<string>("var.cluster_id"),
    }],
  }));

  return resources;
}
```

## Conditional Resource Creation

```typescript
import { Bundle, Resources, createJob } from "@databricks/bundles";

export function loadResources(bundle: Bundle): Resources {
  const resources = new Resources();

  // Create different jobs based on target
  if (bundle.target === "production") {
    resources.addJob("prod_job", createJob({
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
    }));
  } else {
    resources.addJob("dev_job", createJob({
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
    }));
  }

  return resources;
}
```

## Error Handling and Diagnostics

```typescript
import { Bundle, Resources, createJob, Diagnostics } from "@databricks/bundles";

export function loadResources(bundle: Bundle): Resources {
  const resources = new Resources();

  try {
    // Attempt to load configuration
    const config = loadConfig(bundle.target);

    resources.addJob("my_job", createJob(config));
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
  job_name:
    description: Name of the job
    default: "My Job"
  num_workers:
    description: Number of workers
    default: 2

python:
  resources:
    - "resources:loadResources"

targets:
  development:
    variables:
      num_workers: 1

  production:
    variables:
      num_workers: 10
```

Then compile your TypeScript code and reference it:

```bash
# Compile TypeScript
npm run build

# Deploy with Databricks CLI
databricks bundle deploy --target development
```
