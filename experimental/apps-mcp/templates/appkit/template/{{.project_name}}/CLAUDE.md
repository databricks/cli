TypeScript full-stack template powered by **Databricks AppKit** with tRPC for additional custom API endpoints. Use this when building Databricks Apps with the following structure:

- server/: Node.js backend with App Kit and tRPC
- client/: React frontend with App Kit hooks and tRPC client
- config/queries/: SQL query files for analytics
- shared/: Shared TypeScript types

## NPM Scripts

### Development
- `npm run dev` - Start dev server with hot reload (use during active development)
- `npm start` - Start production server (requires `npm run build` first)

### Build
- `npm run build` - Full build (server + client) - use before deployment
- `npm run build:server` - Compile server TypeScript only
- `npm run build:client` - Compile and bundle React app only

### Code Quality
- `npm run typecheck` - Type-check without building (fast validation)
- `npm run lint` - Check for linting issues
- `npm run lint:fix` - Auto-fix linting issues
- `npm run format` - Check code formatting
- `npm run format:fix` - Auto-format code with Prettier

### Testing
- `npm test` - Run unit tests (vitest) + smoke test
- `npm run test:e2e` - Run all Playwright tests
- `npm run test:e2e:ui` - Run Playwright with interactive UI (for debugging)
- `npm run test:smoke` - Run smoke test only (quick validation)

### Utility
- `npm run clean` - Remove all build artifacts and node_modules (requires `npm install` after)

**Common workflows:**
- Development: `npm run dev` → make changes → `npm run typecheck` → `npm run lint:fix`
- Pre-commit: `npm run typecheck && npm run lint:fix && npm run format:fix && npm test`
- Pre-deploy: `npm run build && npm start` (test locally) → `npm test`

## App Naming Constraints

App names must not exceed 30 characters total (including target prefix).

Formula: `target-prefix` + `-` + `app-name` ≤ 30 chars

With default `dev-` target: max app name = **25 characters**

Examples:
- ✅ `dev-sales-dashboard` (19 chars)
- ❌ `dev-retail-inventory-sales-reconciliation` (42 chars - TOO LONG)

The init-template command validates this automatically.

## TypeScript Import Rules

This template uses strict TypeScript settings with `verbatimModuleSyntax: true`. **Always use `import type` for type-only imports**:

```typescript
// ✅ CORRECT - use import type for types
import type { MyInterface, MyType } from '../../shared/types';

// ❌ WRONG - will fail compilation
import { MyInterface, MyType } from '../../shared/types';
```

**Common error this prevents:**
- `TS1484: 'X' is a type and must be imported using a type-only import`

## Databricks App Kit SDK:

This template uses `@databricks/app-kit` which provides:

- **Server setup**: `createApp()` with `server()` and `analytics()` plugins
- **SQL queries**: Store SQL files in `config/queries/` directory
- **React hooks**: `useAnalyticsQuery<T>()` for executing SQL queries from frontend
- **Authentication**: Automatic Databricks workspace authentication

### Server Setup Pattern:

```typescript
import { createApp, server, analytics } from '@databricks/app-kit';

const app = await createApp({
  plugins: [
    server({
      watch: process.env.NODE_ENV === 'development',
      staticPath,
      autoStart: false,
    }),
    analytics({
      timeout: 10000,
    }),
  ],
});

// Extend with custom tRPC endpoints if needed
app.server.extend((express: Application) => {
  express.use('/trpc', [appRouterMiddleware()]);
});

await app.server.start();
```

### Frontend Query Pattern:

```typescript
import { useAnalyticsQuery } from '@databricks/app-kit/react';
import { Skeleton } from '@/components/ui/skeleton';

interface QueryResult { column_name: string; }

function MyComponent() {
  const { data, loading, error } = useAnalyticsQuery<QueryResult[]>('query_name', {});

  if (loading) return <Skeleton className="h-4 w-3/4" />;
  if (error) return <div>Error: {error}</div>;
  return <div>{data?.map(row => row.column_name)}</div>;
}
```

**useAnalyticsQuery API:**

```typescript
const { data, loading, error } = useAnalyticsQuery<T>(
  queryName: string,                        // SQL file name without .sql extension
  params: Record<string, string | number>   // Query parameters
);
// Returns: { data: T | null, loading: boolean, error: string | null }
```

**NOT supported options:**
- `enabled` - Query always executes on mount. Use conditional rendering instead: `{selectedId && <MyComponent id={selectedId} />}`
- `refetch` - Not available. Re-mount component to re-query.

### SQL Query Files:

**IMPORTANT**: ALWAYS use SQL files in `config/queries/` for data retrieval. NEVER use tRPC for SQL queries.

- Store ALL SQL queries in `config/queries/` directory
- Name files descriptively: `trip_statistics.sql`, `user_metrics.sql`, `sales_by_region.sql`
- Reference by filename (without extension) in `useAnalyticsQuery`
- App Kit automatically executes queries against configured Databricks warehouse
- Benefits: Built-in caching, proper connection pooling, better performance

### Query Parameterization:

SQL queries can accept parameters to make them dynamic and reusable. Parameters allow filtering, sorting, and customizing queries based on user input or application state.

#### SQL Parameter Syntax:

Use the `:parameter_name` syntax in SQL files:

```sql
-- config/queries/filtered_data.sql
SELECT *
FROM my_table
WHERE column_value >= :min_value
  AND column_value <= :max_value
  AND category = :category
  AND (:optional_filter = '' OR status = :optional_filter)
```

**Key Points:**

- Parameters use colon prefix: `:parameter_name`
- Databricks infers types from values automatically
- For optional parameters, use pattern: `(:param = '' OR column = :param)`

#### Frontend Parameter Passing:

```typescript
const { data } = useAnalyticsQuery<ResultType[]>('filtered_data', {
  min_value: minValue,
  max_value: maxValue,
  category: category,
  optional_filter: optionalFilter || '',  // empty string for optional params
});
```

#### Date Parameters:

For dates, use `YYYY-MM-DD` format in frontend, `DATE()` function in SQL:

```typescript
// Date helper for query params
const daysAgo = (n: number) => new Date(Date.now() - n * 86400000).toISOString().split('T')[0];

const startDate = daysAgo(7);  // 7 days ago
```

```sql
-- SQL
WHERE DATE(timestamp_column) >= :start_date
```

**Parameter Types:**
- **Strings/Numbers**: Use directly in SQL with `:param_name`
- **Dates**: Format as `YYYY-MM-DD`, use with `DATE()` in SQL
- **Optional**: Use empty string default, check with `(:param = '' OR column = :param)`

## tRPC for Custom Endpoints:

**CRITICAL**: Do NOT use tRPC for SQL queries or data retrieval. Use `config/queries/` + `useAnalyticsQuery` instead.

Use tRPC ONLY for:

- **Mutations**: Creating, updating, or deleting data (INSERT, UPDATE, DELETE)
- **External APIs**: Calling Databricks APIs (serving endpoints, jobs, MLflow, etc.)
- **Complex business logic**: Multi-step operations that cannot be expressed in SQL
- **File operations**: File uploads, processing, transformations
- **Custom computations**: Operations requiring TypeScript/Node.js logic

### Server-side tRPC Pattern:

```typescript
// server/trpc.ts
import { initTRPC } from '@trpc/server';
import { getRequestContext } from '@databricks/app-kit';
import { z } from 'zod';

const t = initTRPC.create({ transformer: superjson });
const publicProcedure = t.procedure;

export const appRouter = t.router({
  // Example: Query a serving endpoint
  queryModel: publicProcedure.input(z.object({ prompt: z.string() })).query(async ({ input: { prompt } }) => {
    const { serviceDatabricksClient: client } = getRequestContext();
    const response = await client.servingEndpoints.query({
      name: 'your-endpoint-name',
      messages: [{ role: 'user', content: prompt }],
    });
    return response;
  }),

  // Example: Mutation
  createRecord: publicProcedure.input(z.object({ name: z.string() })).mutation(async ({ input }) => {
    // Custom logic here
    return { success: true, id: 123 };
  }),
});
```

### Client-side tRPC Pattern:

```typescript
// client/src/components/MyComponent.tsx
import { trpc } from '@/lib/trpc';
import { useState, useEffect } from 'react';

function MyComponent() {
  const [result, setResult] = useState(null);

  useEffect(() => {
    trpc.queryModel
      .query({ prompt: "Hello" })
      .then(setResult)
      .catch(console.error);
  }, []);

  const handleCreate = async () => {
    await trpc.createRecord.mutate({ name: "test" });
  };

  return <div>{/* component JSX */}</div>;
}
```

### When to Use What:

**Decision Tree for Data Operations:**

1. **Is it a SQL query?** → Use `config/queries/*.sql` + `useAnalyticsQuery`
   - SELECT statements
   - Aggregations, JOINs, GROUP BY
   - Analytics and reporting queries
   - Data visualization queries

2. **Is it calling a Databricks API?** → Use tRPC
   - Serving endpoints (model inference)
   - MLflow operations
   - Jobs API
   - Workspace API

3. **Is it modifying data?** → Use tRPC mutations
   - INSERT, UPDATE, DELETE operations
   - Multi-step transactions
   - Business logic with side effects

4. **Is it non-SQL custom logic?** → Use tRPC
   - File processing
   - External API calls
   - Complex computations in TypeScript

❌ **NEVER use tRPC for:**

- Simple data retrieval that can be done with SQL
- Wrapping SQL queries in tRPC endpoints
- SELECT statements of any kind

## Testing Guidelines:

### Smoke Test (Playwright)

The template includes a smoke test at `tests/smoke.spec.ts` that verifies the app loads correctly.

**When customizing the app**, update `tests/smoke.spec.ts` to match your UI:
- Change heading selector to match your app title (replace 'Minimal Databricks App')
- Update data assertions to match your query results (replace 'hello world' check)
- Keep the test simple - just verify app loads and displays data

### Unit Tests (Vitest)

**CRITICAL**: Use vitest for all tests. Put tests next to the code (e.g. src/\*.test.ts)

```typescript
import { describe, it, expect } from 'vitest';

describe('Feature Name', () => {
  it('should do something', () => {
    expect(true).toBe(true);
  });

  it('should handle async operations', async () => {
    const result = await someAsyncFunction();
    expect(result).toBeDefined();
  });
});
```

- Use `describe` blocks to group related tests
- Use `it` for individual test cases
- Use `expect` for assertions
- Tests run with `npm test` (runs `vitest run`)

❌ **Do not write unit tests for query files:**

- writing unit tests for serving sql files under `config/queries` has little value
- do not write unit tests to types associated with queries

### Smoke Test (Playwright)

Keep the smoke test simple - it verifies the app loads and displays data correctly.

The template includes a smoke test at `tests/smoke.spec.ts` that:

- Opens the app
- Waits for data to load (SQL query results and health check)
- Captures screenshots and console logs to `.smoke-test/` directory
- Always captures artifacts, even on test failure (using try-finally)

**When to update the smoke test:**

When you customize the app, update `tests/smoke.spec.ts` to match your UI:
- Change heading selector to match your app title (replace 'Minimal Databricks App')
- Update data assertions to match your query results (replace 'hello world' check)
- Keep the test simple - just verify app loads and displays data
- The default test expects specific template content; update these expectations after customization

**Keep smoke tests simple:**

- Only verify that the app loads and displays initial data
- Wait for key elements to appear (page title, main content)
- Capture artifacts for debugging
- Run quickly (< 5 seconds)

**For extended E2E tests:**

- Create separate test files in `tests/` directory (e.g., `tests/user-flow.spec.ts`)
- Use `npm run test:e2e` to run all Playwright tests
- Keep complex user flows, interactions, and edge cases out of the smoke test

**Running tests:**

```bash
npm run test:smoke      # Run smoke test only
npm run test:e2e        # Run all E2E tests
npm run test:e2e:ui     # Run with Playwright UI
```

## Frontend Styling Guidelines:

### Layout Structure:

```tsx
<div className="container mx-auto p-4">
  <h1 className="text-2xl font-bold mb-4">Page Title</h1>
  <form className="space-y-4 mb-8">{/* form inputs */}</form>
  <div className="grid gap-4">{/* list items */}</div>
</div>
```

### Component Organization:

- Shared UI components: `client/src/components/ui/`
- Feature components: `client/src/components/FeatureName.tsx`
- Split components when logic exceeds ~100 lines or component is reused

### Best Practices:

- Use shadcn/radix components (Button, Input, Card, etc.) for consistent UI
- **Use skeleton loaders**: Always use `<Skeleton>` components instead of plain "Loading..." text
- Define result types in `shared/types.ts` for reuse between frontend and backend
- Handle nullable fields: `value={field || ''}` for inputs
- Type callbacks explicitly: `onChange={(e: React.ChangeEvent<HTMLInputElement>) => ...}`
- Forms should have loading states: `disabled={isLoading}`
- Show empty states with helpful text when no data exists

## Data Visualization with Recharts

The template includes Recharts. Use Databricks brand colors: `['#40d1f5', '#4462c9', '#EB1600', '#0B2026', '#4A4A4A', '#353a4a']` (via `stroke` or `fill` props).

```tsx
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

<Card>
  <CardHeader><CardTitle>My Metrics</CardTitle></CardHeader>
  <CardContent>
    {loading && <Skeleton className="h-[300px] w-full" />}
    {error && <div className="text-destructive">Error: {error}</div>}
    {data && (
      <ResponsiveContainer width="100%" height={300}>
        <LineChart data={data}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="name" />
          <YAxis />
          <Tooltip />
          <Line type="monotone" dataKey="value" stroke="#40d1f5" />
        </LineChart>
      </ResponsiveContainer>
    )}
  </CardContent>
</Card>
```
