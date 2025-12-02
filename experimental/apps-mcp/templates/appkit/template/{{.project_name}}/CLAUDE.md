TypeScript full-stack template powered by **Databricks AppKit** with tRPC for additional custom API endpoints. Use this when building Databricks Apps with the following structure:

- server/: Node.js backend with App Kit and tRPC
- client/: React frontend with App Kit hooks and tRPC client
- config/queries/: SQL query files for analytics
- shared/: Shared TypeScript types

## NPM Scripts

### Development
- `npm run dev` - Start dev server with hot reload (**ALWAYS use this during development**)
- `npm start` - Start production server (**NEVER use during development - production only!**)

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
- Pre-deploy: Validate with `databricks experimental apps-mcp tools validate .`

**CRITICAL - npm run dev vs npm start:**
```
┌─────────────────────────────────────────────────────┐
│ DURING DEVELOPMENT: ALWAYS use `npm run dev`       │
│ - Hot reload enabled                                │
│ - Fast iteration                                    │
│ - Runs on localhost:8000                           │
│                                                     │
│ NEVER use `npm start` during development!          │
│ - npm start is for production deployment only      │
│ - No hot reload                                     │
│ - Requires build step first                        │
└─────────────────────────────────────────────────────┘
```

## Deployment Procedures

**CRITICAL**: Deployment is a significant action that affects live environments. NEVER deploy without explicit user confirmation.

### Before Deploying

**ALWAYS** confirm with the user first by asking: "Ready to deploy to [environment]?"

Pre-deployment checklist:
1. ✅ App validated successfully (`databricks experimental apps-mcp tools validate .`)
2. ✅ Local testing completed (`npm run dev` and manual verification)
3. ✅ User explicitly approved deployment
4. ✅ Code committed to git (recommended)

### Deployment Commands

```bash
databricks bundle deploy              # Deploy app infrastructure
databricks bundle run app            # Start the app
databricks bundle summary            # Get deployed app URL
```

Or use the all-in-one command:
```bash
databricks experimental apps-mcp tools deploy  # Validates, deploys, and runs
```

**DO NOT** run deployment commands unless the user explicitly requests it or confirms when asked.

## Local Development vs Deployed Apps

**CRITICAL**: Understand the difference between local development and deployed environments.

### During Development (Before Deployment)

When the user asks to "open the app", "see the app", "view the app", or "open a dev copy":

- ✅ **Use localhost**: `http://localhost:8000`
- ✅ **Requires**: `npm run dev` must be running
- ✅ **Purpose**: Hot reload, fast iteration, debugging

**DO NOT** suggest deployed URLs (staging/production) during active development!

### After Deployment

Only after running `databricks bundle deploy && databricks bundle run app`:

- ✅ **Use deployed URL**: Get from `databricks bundle summary`
- ✅ **Purpose**: Production/staging environment, sharing with others

### Decision Tree

```
User asks to "open the app" or "see the app"
│
├─ Has the app been deployed in this session?
│  ├─ No  → Use http://localhost:8000 (start with `npm run dev` if needed)
│  └─ Yes → Ask user which environment they want:
│           • localhost (development)
│           • deployed URL (production/staging)
│
└─ User explicitly says "localhost" or "local"
   → Always use http://localhost:8000
```

## Deployment Procedures

**CRITICAL**: Deployment is a significant action that affects live environments. NEVER deploy without explicit user confirmation.

### Before Deploying

**ALWAYS** confirm with the user first by asking: "Ready to deploy to [environment]?"

Pre-deployment checklist:
1. ✅ App validated successfully (`databricks experimental apps-mcp tools validate .`)
2. ✅ Local testing completed (`npm run dev` and manual verification)
3. ✅ User explicitly approved deployment
4. ✅ Code committed to git (recommended)

### Deployment Commands

```bash
databricks bundle deploy              # Deploy app infrastructure
databricks bundle run app            # Start the app
databricks bundle summary            # Get deployed app URL
```

Or use the all-in-one command:
```bash
databricks experimental apps-mcp tools deploy  # Validates, deploys, and runs
```

**DO NOT** run deployment commands unless the user explicitly requests it or confirms when asked.

## Local Development vs Deployed Apps

**CRITICAL**: Understand the difference between local development and deployed environments.

### During Development (Before Deployment)

When the user asks to "open the app", "see the app", "view the app", or "open a dev copy":

- ✅ **Use localhost**: `http://localhost:8000`
- ✅ **Requires**: `npm run dev` must be running
- ✅ **Purpose**: Hot reload, fast iteration, debugging

**DO NOT** suggest deployed URLs (staging/production) during active development!

### After Deployment

Only after running `databricks bundle deploy && databricks bundle run app`:

- ✅ **Use deployed URL**: Get from `databricks bundle summary`
- ✅ **Purpose**: Production/staging environment, sharing with others

### Decision Tree

```
User asks to "open the app" or "see the app"
│
├─ Has the app been deployed in this session?
│  ├─ No  → Use http://localhost:8000 (start with `npm run dev` if needed)
│  └─ Yes → Ask user which environment they want:
│           • localhost (development)
│           • deployed URL (production/staging)
│
└─ User explicitly says "localhost" or "local"
   → Always use http://localhost:8000
```

## App Naming Constraints

App names must not exceed 30 characters total (including target prefix).

Formula: `target-prefix` + `-` + `app-name` ≤ 30 chars

With default `dev-` target: max app name = **25 characters**

Examples:
- ✅ `dev-sales-dashboard` (19 chars)
- ❌ `dev-retail-inventory-sales-reconciliation` (42 chars - TOO LONG)

The init-template command validates this automatically.

## TypeScript Import Rules

This template uses strict TypeScript settings with `verbatimModuleSyntax: true`. **Always use `import type` for type-only imports**.

Template enforces `noUnusedLocals` - remove unused imports immediately or build fails.

**Type-only imports**:

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
- **Visualization Components**: AreaChart, BarChart, LineChart, PieChart, RadarChart, DataTable
- **Authentication**: Automatic Databricks workspace authentication

### Server Setup Pattern:

```typescript
import { createApp, server, analytics } from '@databricks/app-kit';

const app = await createApp({
  plugins: [
    server({
      autoStart: false,
    }),
    analytics(),
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

### Frontend Visualization Pattern:

```typescript
import { AreaChart } from '@databricks/app-kit/react';

function MyComponent() {
  return (
    <div>
      <AreaChart queryKey="query_name" parameters={{}} />
    </div>
  );
}
```

### SQL Query Files:

**IMPORTANT**: ALWAYS use SQL files in `config/queries/` for data retrieval. NEVER use tRPC for SQL queries.

- Store ALL SQL queries in `config/queries/` directory
- Name files descriptively: `trip_statistics.sql`, `user_metrics.sql`, `sales_by_region.sql`
- Reference by filename (without extension) in `useAnalyticsQuery` or directly in a visualization component passing it as `queryKey`
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

### Query Types

Once the schema and the result of a query has been discovered, create its corresponding type in `config/queries/schema.ts` using a zod schema.

Example

```typescript
import { z } from 'zod';

export const querySchemas = {
  mocked_sales: z.array(
    z.object({
      max_month_num: z.number().min(1).max(12),
    })
  ),

  hello_world: z.array(
    z.object({
      value: z.string(),
    })
  ),
};

```

**Key Points:**

- Parameters use colon prefix: `:parameter_name`
- Databricks infers types from values automatically
- For optional string parameters, use pattern: `(:param = '' OR column = :param)`
- **For optional date parameters, use sentinel dates** (`'1900-01-01'` and `'9999-12-31'`) instead of empty strings (see Date Parameters section below)

#### Frontend Parameter Passing:

```typescript
const { data } = useAnalyticsQuery('filtered_data', {
  min_value: minValue,
  max_value: maxValue,
  category: category,
  optional_filter: optionalFilter || '',  // empty string for optional params
});
```

#### Date Parameters:

For dates, use `YYYY-MM-DD` format in frontend, `CAST()` function in SQL:

```typescript
// Date helper for query params
const daysAgo = (n: number) => new Date(Date.now() - n * 86400000).toISOString().split('T')[0];

const startDate = daysAgo(7);  // 7 days ago
```

```sql
-- SQL
WHERE timestamp_column >= CAST(:start_date AS DATE)
```

**⚠️ IMPORTANT: Optional Date Parameters - Use Sentinel Dates**

Databricks App Kit validates parameter types before query execution. **DO NOT use empty strings (`''`) for optional date parameters** as this causes validation errors.

**✅ CORRECT - Use Sentinel Dates:**

```typescript
// Frontend: Use sentinel dates for "no filter" instead of empty strings
const revenueParams = {
  group_by: 'month',
  start_date: '1900-01-01',  // Sentinel: effectively no lower bound
  end_date: '9999-12-31',    // Sentinel: effectively no upper bound
  country: country || '',
  property_type: propertyType || '',
};
```

```sql
-- SQL: Simple comparison since sentinel dates are always valid
WHERE b.check_in >= CAST(:start_date AS DATE)
  AND b.check_in <= CAST(:end_date AS DATE)
```

**❌ WRONG - Empty Strings Cause Validation Errors:**

```typescript
// ❌ DON'T DO THIS - causes "Invalid date format" error
const params = {
  start_date: '',  // Empty string triggers parameter validation error
  end_date: '',
};
```

```sql
-- ❌ DON'T DO THIS - even with conditional logic, validation happens first
WHERE (:start_date = '' OR b.check_in >= CAST(:start_date AS DATE))
```

**Why Sentinel Dates Work:**
- `1900-01-01` is before any real data (effectively no lower bound filter)
- `9999-12-31` is after any real data (effectively no upper bound filter)
- Always valid DATE types, so no parameter validation errors
- All real dates fall within this range, so no filtering occurs

**When to Use Real Dates vs Sentinel Dates:**
- **Sentinel dates**: When you want "no filter" by default (e.g., show all time periods)
- **Real dates**: When you have actual date pickers or filters that users can set

**Parameter Types:**
- **Strings/Numbers**: Use directly in SQL with `:param_name`
- **Dates**: Format as `YYYY-MM-DD`, use with `CAST(:param AS DATE)` in SQL
- **Optional Strings**: Use empty string default, check with `(:param = '' OR column = :param)`
- **Optional Dates**: Use sentinel dates (`'1900-01-01'` and `'9999-12-31'`) instead of empty strings

## SQL Type Handling

Numeric fields from Databricks SQL (especially `ROUND()`, `AVG()`, `SUM()`) are returned as strings in JSON. Convert before using numeric methods:

```typescript
// ❌ WRONG - fails at runtime
<span>{row.total_amount.toFixed(2)}</span>

// ✅ CORRECT
<span>{Number(row.total_amount).toFixed(2)}</span>
```

Use helpers from `shared/types.ts`: `toNumber()`, `formatCurrency()`, `formatPercent()`.

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

1. **Is it a SQL query?** → Use `config/queries/*.sql` + a Visualization component or `useAnalyticsQuery` if no component is available
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

### Radix UI Constraints

- `SelectItem` cannot have `value=""`. Use sentinel value like `"all"` for "show all" options.

### Map Libraries (react-leaflet)

For maps with React 19, use react-leaflet v5:

```bash
npm install react-leaflet@^5.0.0 leaflet @types/leaflet
```

Import CSS in your component:
```typescript
import 'leaflet/dist/leaflet.css';
```

Note: react-leaflet v4.x requires React 18. Use v5 for React 19 compatibility.

### Best Practices:

- Use shadcn/radix components (Button, Input, Card, etc.) for consistent UI
- **Use skeleton loaders**: Always use `<Skeleton>` components instead of plain "Loading..." text
- Define result types in `shared/types.ts` for reuse between frontend and backend
- Handle nullable fields: `value={field || ''}` for inputs
- Type callbacks explicitly: `onChange={(e: React.ChangeEvent<HTMLInputElement>) => ...}`
- Forms should have loading states: `disabled={isLoading}`
- Show empty states with helpful text when no data exists

## Data Visualization with App Kit UI

App Kit UI provides an abstraction over Recharts.

It exports a list of components where each component also exports its own props, so for example to use the `LineChart` it would be used as follows:

```typescript
import { LineChart } from '@databricks/app-kit-ui/react';

function MyComponent() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>My Data</CardTitle>
      </CardHeader>
      <CardContent>
        <LineChart queryKey="my_data" parameters={{}} />
      </CardContent>
    </Card>
  );
}
```
Each component exports their props, so to know the props from `LineChart`, `LineChartProps` can be imported too.

The Visualization components provided by the App Kit UI library can also be used in full control mode combined with Recharts, which is included in the template.
Use Databricks brand colors: `['#40d1f5', '#4462c9', '#EB1600', '#0B2026', '#4A4A4A', '#353a4a']` (via `stroke` or `fill` props).

```tsx
import { LineChart } from '@databricks/app-kit-ui/react';
import { Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

<Card>
  <CardHeader><CardTitle>My Metrics</CardTitle></CardHeader>
  <CardContent>
    <LineChart queryKey="query_name" parameters={salesParameters}>
      <Line type="monotone" dataKey="revenue" stroke="#40d1f5" />
      <Line type="monotone" dataKey="expenses" stroke="#4462c9" />
      <Line type="monotone" dataKey="customers" stroke="#EB1600" />
      <XAxis dataKey="month" />
      <YAxis />
      <Tooltip />
    </LineChart>
  </CardContent>
</Card>
```

Every component handles loading, errors and data fetching internally, so the only thing needed is the `queryKey` and `parameters`.
When rendering fully custom mode, it also needs the recharts components for that specific visualization component.
