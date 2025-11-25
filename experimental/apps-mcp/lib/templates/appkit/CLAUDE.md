TypeScript full-stack template powered by the **Databricks Apps SDK** with tRPC for additional custom API endpoints. Use this when building Databricks Apps with the following structure:
- server/: Node.js backend with App Kit and tRPC
- client/: React frontend with App Kit hooks and tRPC client
- config/queries/: SQL query files for analytics
- shared/: Shared TypeScript types

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

// Define result type matching your SQL query structure
interface QueryResult {
  column_name: string;
  // ... other columns
}

function MyComponent() {
  // Reference SQL file by name (without .sql extension)
  const { data, loading, error } = useAnalyticsQuery<QueryResult[]>('query_name', {});

  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error: {error}</div>;

  return <div>{data?.map(row => row.column_name)}</div>;
}
```

### SQL Query Files:

**IMPORTANT**: ALWAYS use SQL files in `config/queries/` for data retrieval. NEVER use tRPC for SQL queries.

- Store ALL SQL queries in `config/queries/` directory
- Name files descriptively: `trip_statistics.sql`, `user_metrics.sql`, `sales_by_region.sql`
- Reference by filename (without extension) in `useAnalyticsQuery`
- App Kit automatically executes queries against configured Databricks warehouse
- Benefits: Built-in caching, proper connection pooling, better performance

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
  queryModel: publicProcedure
    .input(z.object({ prompt: z.string() }))
    .query(async ({ input: { prompt } }) => {
      const { serviceDatabricksClient: client } = getRequestContext();
      const response = await client.servingEndpoints.query({
        name: "your-endpoint-name",
        messages: [{ role: "user", content: prompt }],
      });
      return response;
    }),

  // Example: Mutation
  createRecord: publicProcedure
    .input(z.object({ name: z.string() }))
    .mutation(async ({ input }) => {
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

**CRITICAL**: Use vitest for all tests. Put tests next to the code (e.g. src/*.test.ts)

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

## Frontend Styling Guidelines:

### Component Structure Pattern:
- Use container with proper spacing: `<div className="container mx-auto p-4">`
- Page titles: `<h1 className="text-2xl font-bold mb-4">Title</h1>`
- Forms: Use `space-y-4` for vertical spacing between inputs
- Cards: Use shadcn Card components or `border p-4 rounded-md` for item display
- Grids: Use `grid gap-4` for list layouts

### Example App Structure:
```tsx
<div className="container mx-auto p-4">
  <h1 className="text-2xl font-bold mb-4">Page Title</h1>
  <form className="space-y-4 mb-8">{/* form inputs */}</form>
  <div className="grid gap-4">{/* list items */}</div>
</div>
```

### Tailwind Usage:
- Use Tailwind classes directly in JSX
- Avoid @apply unless creating reusable component styles
- When using @apply, only in @layer components (never @layer base)
- Template has CSS variables defined - use via Tailwind (bg-background, text-foreground, etc.)

### Typography & Spacing:
- Headings: text-2xl font-bold with mb-4
- Secondary text: text-foreground/70
- Card titles: text-xl font-semibold
- Form spacing: space-y-4 between inputs, mb-8 after forms
- Grid/list spacing: gap-4 for consistent item spacing

### Component Organization:
Create separate components when:
- Logic exceeds ~100 lines
- Component is reused in multiple places
- Component has distinct responsibility (e.g., ProductForm, ProductList)
File structure:
- Shared UI: client/src/components/ui/
- Feature components: client/src/components/FeatureName.tsx

### Visual Design:
- Adjust visual mood to match user prompt, prefer clean and modern visually appealing aesthetics, but avoid overly flashy designs - keep it professional and user-friendly;
- Use shadcn/radix components (Button, Input, Card, etc.) for consistent UI
- Forms should have loading states: `disabled={isLoading}`
- Show empty states with helpful text when no data exists

### Best Practices:
- **SQL queries**: ALWAYS use `config/queries/*.sql` + `useAnalyticsQuery()` (never use tRPC for data retrieval)
- **Data retrieval workflow**:
  1. Create SQL file in `config/queries/my_query.sql`
  2. Define result type in `shared/types.ts`
  3. Use `useAnalyticsQuery<MyType[]>('my_query', {})` in component
- **Custom APIs**: Use tRPC ONLY for non-SQL operations (mutations, external APIs, complex business logic)
- Handle nullable fields: `value={field || ''}` for inputs
- Type all callbacks explicitly: `onChange={(e: React.ChangeEvent<HTMLInputElement>) => ...}`
- Define result types in `shared/types.ts` for reuse between frontend and backend

## Data Visualization with Recharts

The template includes Recharts for data visualization. Use Databricks brand colors for chart elements: `['#40d1f5', '#4462c9', '#EB1600', '#0B2026', '#4A4A4A', '#353a4a']` (apply via `stroke` or `fill` props).

### Basic Chart Pattern:
```tsx
import { useAnalyticsQuery } from '@databricks/app-kit/react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

// Define type in shared/types.ts
interface MetricData {
  name: string;
  value: number;
}

function MyDashboard() {
  // Query references SQL file in config/queries/metrics.sql
  const { data, loading, error } = useAnalyticsQuery<MetricData[]>('metrics', {});

  return (
    <Card>
      <CardHeader>
        <CardTitle>My Metrics</CardTitle>
      </CardHeader>
      <CardContent>
        {loading && <div>Loading metrics...</div>}
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
  );
}
```

**Note**: Create `config/queries/metrics.sql` with your SQL query that returns columns matching the `MetricData` type.
