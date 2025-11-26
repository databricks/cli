TypeScript full-stack template powered by **Databricks AppKit** with tRPC for additional custom API endpoints. Use this when building Databricks Apps with the following structure:
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
  // Second parameter is for query parameters (see Query Parameterization section)
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

Pass parameters as the second argument to `useAnalyticsQuery`:

```typescript
import { useState } from 'react';
import { useAnalyticsQuery } from '@databricks/app-kit/react';

function FilteredComponent() {
  const [minValue, setMinValue] = useState('0');
  const [maxValue, setMaxValue] = useState('100');
  const [category, setCategory] = useState('electronics');
  const [optionalFilter, setOptionalFilter] = useState('');

  // Build parameters object
  const queryParams = {
    min_value: minValue,
    max_value: maxValue,
    category: category,
    optional_filter: optionalFilter || '', // Default for optional params
  };

  // Pass parameters to query
  const { data, loading, error } = useAnalyticsQuery<ResultType[]>(
    'filtered_data',
    queryParams
  );

  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error: {error}</div>;
  return <div>{/* render data */}</div>;
}
```

#### Date Parameters:

**IMPORTANT**: For date parameters, use `YYYY-MM-DD` format with `DATE()` function in SQL.

**Frontend - Date Formatting:**
```typescript
const getDateParam = (daysAgo: number): string => {
  const date = new Date();
  date.setDate(date.getDate() - daysAgo);
  // Format as YYYY-MM-DD
  return date.toISOString().split("T")[0];
};

const queryParams = {
  start_date: getDateParam(7), // 7 days ago
};
```

**SQL - Date Comparison:**
```sql
-- Use DATE() function for timestamp comparisons
WHERE DATE(timestamp_column) >= :start_date
```

#### Complete Example:

**SQL Query** (`config/queries/trip_statistics.sql`):
```sql
SELECT
  COUNT(*) as total_trips,
  ROUND(AVG(trip_distance), 2) as avg_distance,
  ROUND(AVG(fare_amount), 2) as avg_fare
FROM samples.nyctaxi.trips
WHERE fare_amount > 0
  AND DATE(tpep_pickup_datetime) >= :start_date
  AND fare_amount >= :min_fare
  AND fare_amount <= :max_fare
  AND (:zip_code = '' OR pickup_zip = :zip_code)
```

**TypeScript Component:**
```typescript
import { useState } from 'react';
import { useAnalyticsQuery } from '@databricks/app-kit/react';

interface TripStatistics {
  total_trips: number;
  avg_distance: number;
  avg_fare: number;
}

function TripDashboard() {
  const [dateFilter, setDateFilter] = useState<'week' | 'month' | 'all'>('all');
  const [minFare, setMinFare] = useState('');
  const [maxFare, setMaxFare] = useState('');
  const [zipCode, setZipCode] = useState('');

  const getStartDate = (): string => {
    const now = new Date();
    const startDate = dateFilter === 'week'
      ? new Date(now.setDate(now.getDate() - 7))
      : dateFilter === 'month'
      ? new Date(now.setMonth(now.getMonth() - 1))
      : new Date('2000-01-01');
    return startDate.toISOString().split("T")[0];
  };

  const queryParams = {
    start_date: getStartDate(),
    min_fare: minFare || '0',
    max_fare: maxFare || '999999',
    zip_code: zipCode || '',
  };

  const { data, loading, error } = useAnalyticsQuery<TripStatistics[]>(
    'trip_statistics',
    queryParams
  );

  return (
    <div>
      {/* Filter UI */}
      <select value={dateFilter} onChange={(e) => setDateFilter(e.target.value as any)}>
        <option value="week">Last Week</option>
        <option value="month">Last Month</option>
        <option value="all">All Time</option>
      </select>
      {/* Display results */}
      {data && data[0] && (
        <div>
          <p>Total Trips: {data[0].total_trips}</p>
          <p>Avg Distance: {data[0].avg_distance} mi</p>
          <p>Avg Fare: ${data[0].avg_fare}</p>
        </div>
      )}
    </div>
  );
}
```

**Parameter Types Reference:**
- **Strings**: `status: 'active'` → Use directly in SQL: `:status`
- **Numbers**: `min_value: 100` → Use directly: `:min_value`
- **Dates**: Format as `YYYY-MM-DD` → Use with `DATE()` in SQL
- **Optional**: Provide empty string default → Check with `(:param = '' OR ...)`

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

### Unit Tests (Vitest)

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

❌ **Do not write unit tests for query files:**
- writing unit tests for serving sql files under `config/queries` has little value
- do not write unit tests to types associated with queries

### Smoke Test (Playwright)

**CRITICAL**: Keep the smoke test simple - it verifies the app loads and displays data correctly.

The template includes a smoke test at `tests/smoke.spec.ts` that:
- Opens the app
- Waits for data to load (SQL query results and health check)
- Captures screenshots and console logs to `.smoke-test/` directory
- Always captures artifacts, even on test failure (using try-finally)

**When to update the smoke test:**
- When you change what data is displayed on the initial page load
- When you modify the main App component's loading behavior
- When you add/remove data sources that should be validated on startup

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
