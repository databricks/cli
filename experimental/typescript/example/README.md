# TypeScript Bundle Example

This example demonstrates how to create Databricks resources using TypeScript with the Databricks Asset Bundles framework.

## What's Included

This example creates two resources:

1. **Databricks App** (`my_data_app`) - An interactive data exploration app
2. **Unity Catalog Volume** (`landing_zone`) - A managed volume for file storage

## Project Structure

```
example/
├── databricks.yml      # Bundle configuration
├── resources.ts        # TypeScript resource definitions
└── README.md          # This file
```

## Usage

### Prerequisites

- Databricks CLI installed
- Node.js and npm installed
- Access to a Databricks workspace

### Setup

1. **Install dependencies**:
   ```bash
   cd example
   npm install
   ```

2. **Build TypeScript to JavaScript**:
   ```bash
   npm run build
   ```

   This compiles the TypeScript files in the current directory to JavaScript in the `dist/` folder.

### Deployment

1. **Set required variables**:
   ```bash
   export DATABRICKS_WAREHOUSE_ID="your-warehouse-id"
   ```

2. **Deploy to dev environment**:
   ```bash
   databricks bundle deploy --target dev
   ```

3. **Deploy to production**:
   ```bash
   databricks bundle deploy --target prod
   ```

### Development Workflow

- **Watch mode**: Auto-compile on file changes
  ```bash
  npm run watch
  ```

- **Clean build**: Remove dist and rebuild
  ```bash
  npm run rebuild
  ```

## How It Works

### Resource Definition (`resources.ts`)

The `loadResources` function is called by the Databricks CLI during bundle processing:

```typescript
export function loadResources(bundle: Bundle): Resources {
  const resources = new Resources();

  // Create variables
  const warehouseId = variable<string>("var.warehouse_id");

  // Add resources
  resources.addApp("my_data_app", createApp({
    name: `${bundle.target}_data_explorer`,
    source_code_path: "/Workspace/apps/data_explorer",
    // ...
  }));

  return resources;
}
```

### Configuration (`databricks.yml`)

The bundle configuration specifies:
- Variables (catalog, schema, warehouse)
- TypeScript resource loader
- Target-specific overrides

```yaml
python:
  resources:
    - "example/resources:loadResources"
```

## Type Safety

TypeScript provides full type safety for all resource definitions:

- **IntelliSense**: Auto-completion for all properties
- **Type Checking**: Compile-time validation
- **Documentation**: Inline JSDoc comments from the schema

## Variables

Variables can be referenced using the `variable()` function:

```typescript
const warehouseId = variable<string>("var.warehouse_id");

resources.addApp("app", createApp({
  resources: [{
    sql_warehouse: {
      id: warehouseId,  // Type-safe variable reference
      permission: "CAN_USE",
    }
  }]
}));
```

At deployment time, these resolve to their configured values.

## Customization

To add more resources:

1. Import the creator function (e.g., `createJob`, `createPipeline`)
2. Call `resources.add{ResourceType}(name, config)`
3. Deploy with `databricks bundle deploy`

Example adding a job:

```typescript
import { createJob } from "../generated/jobs/index.js";

resources.addJob("data_processing", createJob({
  name: "Daily ETL",
  tasks: [{
    task_key: "process",
    notebook_task: {
      notebook_path: "/notebooks/etl",
    },
  }],
}));
```

## Learn More

- [Databricks Asset Bundles Documentation](https://docs.databricks.com/dev-tools/bundles/)
- [TypeScript Bundle API](../README.md)
- [Generated Resource Types](../generated/)
