# AppKit Backend (@databricks/appkit)

## Server Plugin (`server()`)

What it does:
- Starts an Express server (default `host=0.0.0.0`, `port=8000`)
- Mounts plugin routes under `/api/<pluginName>/...`
- Adds `/health` (returns `{ status: "ok" }`)
- Serves frontend:
  - **Development** (`NODE_ENV=development`): runs a Vite dev server in middleware mode
  - **Production**: auto-detects static frontend directory

```ts
import { createApp, server } from "@databricks/appkit";

await createApp({
  plugins: [
    server({
      port: 8000,          // default: Number(process.env.DATABRICKS_APP_PORT) || 8000
      host: "0.0.0.0",     // default: process.env.FLASK_RUN_HOST || "0.0.0.0"
      autoStart: true,     // default: true
      staticPath: "dist",  // optional: force a specific static directory
    }),
  ],
});
```

### Manual Server Start

When you need to extend Express:

```ts
import { createApp, server } from "@databricks/appkit";

const appkit = await createApp({
  plugins: [server({ autoStart: false })],
});

appkit.server.extend((app) => {
  app.get("/custom", (_req, res) => res.json({ ok: true }));
});

await appkit.server.start();
```

## Analytics Plugin (`analytics()`)

Add SQL query execution backed by Databricks SQL Warehouses:

```ts
import { analytics, createApp, server } from "@databricks/appkit";

await createApp({
  plugins: [server(), analytics({})],
});
```

### SQL Queries

Put `.sql` files in `config/queries/`. Query key is filename without `.sql`:

```sql
-- config/queries/spend_summary.sql
-- @param startDate DATE
-- @param endDate DATE
-- @param limit NUMERIC
SELECT *
FROM usage
WHERE usage_date BETWEEN :startDate AND :endDate
LIMIT :limit
```

**Supported `-- @param` types:** STRING, NUMERIC, BOOLEAN, DATE, TIMESTAMP, BINARY

**Server-injected params:**
- `:workspaceId` is injected automatically and must NOT be annotated

### HTTP Endpoints

Mounted under `/api/analytics`:

- `POST /api/analytics/query/:query_key`
- `POST /api/analytics/users/me/query/:query_key`
- `GET /api/analytics/arrow-result/:jobId`

**Formats:**
- `format: "JSON"` (default) returns JSON rows
- `format: "ARROW"` returns Arrow statement_id over SSE

## Execution Context

### `asUser(req)` for User-Scoped Operations

```ts
// Execute as the user (uses their Databricks permissions)
router.post("/users/me/data", async (req, res) => {
  const result = await this.asUser(req).query("SELECT ...");
  res.json(result);
});

// Service principal execution (default)
router.post("/system/data", async (req, res) => {
  const result = await this.query("SELECT ...");
  res.json(result);
});
```

### Context Helper Functions

- `getExecutionContext()`: Returns current context (user or service)
- `getCurrentUserId()`: Returns user ID in user context
- `getWorkspaceClient()`: Returns appropriate WorkspaceClient
- `getWarehouseId()`: `Promise<string>`
- `getWorkspaceId()`: `Promise<string>`
- `isInUserContext()`: Returns `true` if in user context

## Custom Plugins

```ts
import { Plugin, toPlugin } from "@databricks/appkit";
import type express from "express";

class MyPlugin extends Plugin {
  name = "my-plugin";
  envVars = [];

  injectRoutes(router: express.Router) {
    this.route(router, {
      name: "hello",
      method: "get",
      path: "/hello",
      handler: async (_req, res) => {
        res.json({ ok: true });
      },
    });
  }
}

export const myPlugin = toPlugin<typeof MyPlugin, Record<string, never>, "my-plugin">(
  MyPlugin,
  "my-plugin",
);
```

## Caching

### Global Cache

```ts
await createApp({
  plugins: [server(), analytics({})],
  cache: {
    enabled: true,
    ttl: 3600,              // seconds
    strictPersistence: false,
  },
});
```

### Plugin-Level Cache

```ts
// inside a Plugin subclass:
const value = await this.cache.getOrExecute(
  ["my-plugin", "data", userId],
  async () => expensiveWork(),
  userKey,
  { ttl: 300 },
);
```

## Environment Variables

### Required for Databricks Apps

| Variable | Description |
|----------|-------------|
| `DATABRICKS_HOST` | Workspace URL |
| `DATABRICKS_APP_PORT` | Port to bind (default: 8000) |
| `DATABRICKS_APP_NAME` | App name in Databricks |

### Required for SQL Queries

| Variable | How to Set |
|----------|------------|
| `DATABRICKS_WAREHOUSE_ID` | In `app.yaml`: `valueFrom: sql-warehouse` |

### Local Development Auth

**Option 1: Databricks CLI Auth (recommended)**

```bash
databricks auth login --host [host] --profile [profile-name]
DATABRICKS_CONFIG_PROFILE=my-profile npm run dev
```

**Option 2: Environment variables**

```bash
export DATABRICKS_HOST="https://xxx.cloud.databricks.com"
export DATABRICKS_TOKEN="dapi..."
export DATABRICKS_WAREHOUSE_ID="abc123..."
npm run dev
```

**Option 3: `.env` file**

```bash
# .env (add to .gitignore!)
DATABRICKS_HOST=https://xxx.cloud.databricks.com
DATABRICKS_TOKEN=dapi...
DATABRICKS_WAREHOUSE_ID=abc123...
```
