# Databricks App Kit SDK

## TypeScript Import Rules

This template uses strict TypeScript settings with `verbatimModuleSyntax: true`. **Always use `import type` for type-only imports**.

Template enforces `noUnusedLocals` - remove unused imports immediately or build fails.

```typescript
// ✅ CORRECT - use import type for types
import type { MyInterface, MyType } from '../../shared/types';

// ❌ WRONG - will fail compilation
import { MyInterface, MyType } from '../../shared/types';
```

## Server Setup

```typescript
import { createApp, server, analytics } from '@databricks/app-kit';

const app = await createApp({
  plugins: [
    server({ autoStart: false }),
    analytics(),
  ],
});

// Extend with custom tRPC endpoints if needed
app.server.extend((express: Application) => {
  express.use('/trpc', [appRouterMiddleware()]);
});

await app.server.start();
```

## useAnalyticsQuery Hook

**ONLY use when displaying data in a custom way that isn't a chart or table.**

Use cases:
- Custom HTML layouts (cards, lists, grids)
- Summary statistics and KPIs
- Conditional rendering based on data values
- Data that needs transformation before display

**⚠️ CRITICAL: Always cast data to the correct type**

The hook's return type doesn't automatically infer from QueryRegistry. You MUST cast the data manually:

```typescript
import { useAnalyticsQuery, Skeleton } from '@databricks/appkit-ui/react';

// Define your result type based on your schema
interface QueryResult { column_name: string; value: number; }

function CustomDisplay() {
  const { data, loading, error } = useAnalyticsQuery('query_name', {
    start_date: sql.date(Date.now()),
    category: sql.string("tools")
  });

  // ✅ REQUIRED: Cast data to your type
  const typedData = data as QueryResult[] | undefined;

  if (loading) return <Skeleton className="h-4 w-3/4" />;
  if (error) return <div className="text-destructive">Error: {error}</div>;

  return (
    <div className="grid gap-4">
      {typedData?.map(row => (
        <div key={row.column_name} className="p-4 border rounded">
          <h3>{row.column_name}</h3>
          <p>{row.value}</p>
        </div>
      ))}
    </div>
  );
}
```

**❌ WRONG - Don't use data directly without casting:**
```typescript
// This will cause TypeScript errors like "Property 'map' does not exist on type '{}'"
const { data } = useAnalyticsQuery('my_query', {});
data?.map(...)  // TypeScript error!
```

**✅ CORRECT - Always cast first:**
```typescript
const { data } = useAnalyticsQuery('my_query', {});
const typedData = data as MyType[] | undefined;
typedData?.map(...)  // Works!
```

**API:**

```typescript
const { data, loading, error } = useAnalyticsQuery<T>(
  queryName: string,                        // SQL file name without .sql extension
  params: Record<string, SQLTypeMarker>     // Query parameters
);
// Returns: { data: T | null, loading: boolean, error: string | null }
```

**NOT supported:**
- `enabled` - Query always executes on mount. Use conditional rendering: `{selectedId && <MyComponent id={selectedId} />}`
- `refetch` - Not available. Re-mount component to re-query.
