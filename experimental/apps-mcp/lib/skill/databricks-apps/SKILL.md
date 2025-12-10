---
name: databricks-apps
description: Guide for building and deploying Databricks Apps using AppKit. Use when creating data visualization apps, dashboards and similar data-oriented apps. Covers scaffolding, development, validation, and deployment.
---

# Databricks Apps Development Guide

## Overview

Build full-stack TypeScript apps that query Databricks SQL warehouses and deploy to Databricks Apps. Uses AppKit SDK for data fetching and visualization.

## CLI Wrapper

All Databricks CLI commands use the bundled wrapper script which logs commands to stderr:

```bash
scripts/db <command>
```

---

# Workflow

## Phase 1: Environment Setup

### Verify Authentication

```bash
scripts/db auth profiles
```

If auth issues, see [Authentication Reference](./reference/authentication.md).

### Check Available Warehouses

```bash
scripts/db sql warehouses list
```

Note a warehouse ID for the app. Can also set `DATABRICKS_WAREHOUSE_ID` env var.

---

## Phase 2: Data Exploration

Before scaffolding, explore available data to understand what to visualize.

### Browse Catalogs/Schemas/Tables

```bash
scripts/db catalogs list
scripts/db schemas list <catalog>
scripts/db tables list <catalog> <schema>
```

### Discover Table Schema

```bash
scripts/db experimental apps-mcp tools discover-schema CATALOG.SCHEMA.TABLE
```

Returns: columns, types, sample data (5 rows), null counts.

### Test Queries

```bash
scripts/db experimental apps-mcp tools query "SELECT * FROM catalog.schema.table LIMIT 10"
```

**Note:** Use separate arguments for `catalogs/schemas/tables` commands. Dot notation only works in `discover-schema` and `query`.

---

## Phase 3: Project Scaffolding

### Create New App

```bash
scripts/db experimental apps-mcp tools init-template --name my-app --description "My app description"
```

**Constraints:**
- App name must be ≤26 characters (dev- prefix adds 4 chars, max total 30)
- Use lowercase letters, numbers, and hyphens only

### Project Structure

After scaffolding:
- `server/` - Node.js backend with App Kit and tRPC
- `client/` - React frontend with App Kit hooks
- `config/queries/` - SQL query files for analytics
- `shared/` - Shared TypeScript types
- `docs/` - Detailed documentation

---

## Phase 4: Development

### Start Dev Server

```bash
cd <project-name>
npm install
npm run dev
```

Dev server runs on localhost (port shown in output).

### Adding Data Visualizations

**Step 1: Create SQL query file**
```sql
-- config/queries/my_data.sql
SELECT category, COUNT(*) as count
FROM my_table
GROUP BY category
```

**Step 2: Define schema in `config/queries/schema.ts`**
```typescript
export const querySchemas = {
  my_data: z.array(
    z.object({
      category: z.string(),
      count: z.number(),
    })
  ),
};
```

**Step 3: Add visualization**
```typescript
import { BarChart } from '@databricks/app-kit-ui/react';

<BarChart queryKey="my_data" parameters={{}} />
```

### Key Documentation

Load these references when implementing features:
- [SQL Queries](./reference/sql-queries.md) - query files, schemas, type handling, parameterization
- [App Kit SDK](./reference/app-kit-sdk.md) - imports, server setup, useAnalyticsQuery hook
- [Frontend](./reference/frontend.md) - visualization components, styling, Radix constraints
- [tRPC](./reference/trpc.md) - custom endpoints for mutations, Databricks APIs
- [Testing](./reference/testing.md) - vitest unit tests, Playwright smoke tests
- [Authentication](./reference/authentication.md) - profiles, OAuth, troubleshooting

---

## Phase 5: Validation

**Always validate before deploying:**

```bash
scripts/db experimental apps-mcp tools validate ./
```

This catches common issues: build errors, type errors, lint issues, test failures.
After the validation passes, double check by looking at the screenshot and browser console logs in `.smoke-test/` of the app dir. 


---

## Phase 6: Deployment

**Requires explicit user permission.**

```bash
scripts/db experimental apps-mcp tools deploy
```

### View Deployed App

```bash
scripts/db bundle summary
```

---

## CLI Reference

### Exploring Data

```bash
# List catalogs
scripts/db catalogs list

# List schemas in catalog
scripts/db schemas list <catalog>

# List tables
scripts/db tables list <catalog> <schema>

# Discover table schemas (columns, types, sample data)
scripts/db experimental apps-mcp tools discover-schema CATALOG.SCHEMA.TABLE1 CATALOG.SCHEMA.TABLE2

# Run SQL query
scripts/db experimental apps-mcp tools query "SELECT * FROM catalog.schema.table LIMIT 10"
```

**Note:** Use separate arguments for catalog/schema in most commands: `tables list samples tpcds_sf1` (not dot notation). Dot notation only works in `discover-schema` and `query`.

### Jobs and Clusters

```bash
scripts/db jobs list
scripts/db jobs get <job_id>
scripts/db clusters list
```

### Bundle Management

```bash
scripts/db bundle summary
scripts/db bundle validate
scripts/db bundle deploy
```

---

## Local vs Deployed

**During development:**
- Use `npm run dev`
- Access via localhost URL shown in terminal

**After deployment:**
- Get URL from `scripts/db bundle summary`

**Decision tree:**
- "open the app" + not deployed → localhost
- "open the app" + deployed → ask which environment
- "localhost"/"local" → always localhost

---

## Best Practices

- Always validate before deploying
- For operations affecting live environments, ask for confirmation
- Use SQL files in `config/queries/` for data retrieval (never tRPC)
- Run `npm run dev` to auto-regenerate TypeScript types after schema changes
- Check docs for numeric type handling (all SQL numbers return as strings in JSON)
