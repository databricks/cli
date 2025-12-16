TypeScript full-stack template powered by **Databricks AppKit** with tRPC for additional custom API endpoints.

- server/: Node.js backend with App Kit and tRPC
- client/: React frontend with App Kit hooks and tRPC client
- config/queries/: SQL query files for analytics
- shared/: Shared TypeScript types
- docs/: Detailed documentation on using App Kit features

## Quick Start: Your First Query & Chart

Follow these 3 steps to add data visualization to your app:

**Step 1: Create a SQL query file**

```sql
-- config/queries/my_data.sql
SELECT category, COUNT(*) as count, AVG(value) as avg_value
FROM my_table
GROUP BY category
```

**Step 2: Define the schema**

```typescript
// config/queries/schema.ts
export const querySchemas = {
  my_data: z.array(
    z.object({
      category: z.string(),
      count: z.number(),
      avg_value: z.number(),
    })
  ),
};
```

**Step 3: Add visualization to your app**

```typescript
// client/src/App.tsx
import { BarChart } from '@databricks/app-kit-ui/react';

<BarChart queryKey="my_data" parameters={{}} />
```

**That's it!** The component handles data fetching, loading states, and rendering automatically.

**To refresh TypeScript types after adding queries:**
- Run `npm run typegen` OR run `npm run dev` - both auto-generate type definitions in `client/src/appKitTypes.d.ts`
- DO NOT manually edit `appKitTypes.d.ts`

## Installation

**IMPORTANT**: When running `npm install`, always use `required_permissions: ['all']` to avoid sandbox permission errors.

## NPM Scripts

### Development
- `npm run dev` - Start dev server with hot reload (**ALWAYS use during development**)

### Testing and Code Quality
See the databricks experimental apps-mcp tools validate instead of running these individually.

### Utility
- `npm run clean` - Remove all build artifacts and node_modules

**Common workflows:**
- Development: `npm run dev` → make changes → `npm run typecheck` → `npm run lint:fix`
- Pre-deploy: Validate with `databricks experimental apps-mcp tools validate .`

## Documentation

**IMPORTANT**: Read the relevant docs below before implementing features. They contain critical information about common pitfalls (e.g., SQL numeric type handling, schema definitions, Radix UI constraints).

- [SQL Queries](docs/sql-queries.md) - query files, schemas, type handling, parameterization
- [App Kit SDK](docs/app-kit-sdk.md) - TypeScript imports, server setup, useAnalyticsQuery hook
- [Frontend](docs/frontend.md) - visualization components, styling, layout, Radix constraints
- [tRPC](docs/trpc.md) - custom endpoints for non-SQL operations (mutations, Databricks APIs)
- [Testing](docs/testing.md) - vitest unit tests, Playwright smoke/E2E tests
