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

```typescript
import { useAnalyticsQuery } from '@databricks/app-kit-ui/react';
import { Skeleton } from '@/components/ui/skeleton';

interface QueryResult { column_name: string; value: number; }

function CustomDisplay() {
  const { data, loading, error } = useAnalyticsQuery<QueryResult[]>('query_name', {});

  if (loading) return <Skeleton className="h-4 w-3/4" />;
  if (error) return <div className="text-destructive">Error: {error}</div>;

  return (
    <div className="grid gap-4">
      {data?.map(row => (
        <div key={row.column_name} className="p-4 border rounded">
          <h3>{row.column_name}</h3>
          <p>{row.value}</p>
        </div>
      ))}
    </div>
  );
}
```

**API:**

```typescript
const { data, loading, error } = useAnalyticsQuery<T>(
  queryName: string,                        // SQL file name without .sql extension
  params: Record<string, string | number>   // Query parameters
);
// Returns: { data: T | null, loading: boolean, error: string | null }
```

**NOT supported:**
- `enabled` - Query always executes on mount. Use conditional rendering: `{selectedId && <MyComponent id={selectedId} />}`
- `refetch` - Not available. Re-mount component to re-query.
