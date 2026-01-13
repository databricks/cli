---
name: appkit
description: Build full-stack TypeScript Databricks Apps using @databricks/appkit (backend) and @databricks/appkit-ui (frontend). Use for analytics dashboards, data apps, SQL query visualization, SSE streaming, and Arrow data processing.
---

# AppKit - Full-Stack TypeScript Databricks Apps

Build **full-stack TypeScript apps** on Databricks using:

- **Backend**: `@databricks/appkit`
- **Frontend**: `@databricks/appkit-ui`
- **Analytics**: SQL files in `config/queries/*.sql` executed via the AppKit analytics plugin

## Hard Rules (LLM Guardrails)

- **Do not invent APIs**. Stick to patterns shown here and documented exports only.
- **`createApp()` is async**. Use **top-level `await createApp(...)`**.
- **Always memoize query parameters** passed to `useAnalyticsQuery` / charts to avoid refetch loops.
- **Always handle loading/error/empty states** in UI (use `Skeleton`, error text, empty state).
- **Always use `sql.*` helpers** for query parameters (do not pass raw strings/numbers).
- **Never construct SQL strings dynamically**. Use parameterized queries with `:paramName`.
- **Never use `require()`**. Use ESM `import/export`.
- **Charts do NOT accept children**. Use props (`xKey`, `yKey`, `colors`) NOT Recharts children.

## TypeScript Import Rules

With `verbatimModuleSyntax: true`, **always use `import type` for type-only imports**:

```ts
import type { ReactNode } from "react";
import { useMemo } from "react";
```

## Detailed Documentation

- **Backend**: [appkit-backend.md](appkit-backend.md)
- **Frontend**: [appkit-frontend.md](appkit-frontend.md)
- **Project Setup**: [appkit-scaffolding.md](appkit-scaffolding.md)

## Quick Start

### Minimal Server

```ts
// server/index.ts
import { createApp, server, analytics } from "@databricks/appkit";

await createApp({
  plugins: [server(), analytics({})],
});
```

### Minimal Frontend

```tsx
// client/src/App.tsx
import { BarChart, Card, CardHeader, CardTitle, CardContent } from "@databricks/appkit-ui/react";
import { useMemo } from "react";

export default function App() {
  const params = useMemo(() => ({}), []);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Sales by Region</CardTitle>
      </CardHeader>
      <CardContent>
        <BarChart queryKey="sales_by_region" parameters={params} />
      </CardContent>
    </Card>
  );
}
```

### SQL Query

```sql
-- config/queries/sales_by_region.sql
SELECT region, SUM(revenue) as revenue
FROM sales
GROUP BY region
```

## Common Patterns

### useAnalyticsQuery (Custom UI Only)

Use only when you need custom UI (cards/KPIs/conditional rendering):

```tsx
import { useMemo } from "react";
import { useAnalyticsQuery, Skeleton } from "@databricks/appkit-ui/react";
import { sql } from "@databricks/appkit-ui/js";

export function Users() {
  const params = useMemo(
    () => ({
      status: sql.string("active"),
      limit: sql.number(50),
    }),
    [],
  );

  const { data, loading, error } = useAnalyticsQuery("users_list", params);

  if (loading) return <Skeleton className="h-24 w-full" />;
  if (error) return <div className="text-destructive">Error: {error}</div>;
  if (!data || data.length === 0) return <div>No results</div>;

  return <pre>{JSON.stringify(data, null, 2)}</pre>;
}
```

**Limitations:**
- No `enabled` option. Use conditional rendering to mount/unmount.
- No `refetch()`. Change `parameters` (memoized) or re-mount.

### Avoid Double-Fetching

```tsx
// ❌ Wrong: fetches the same query twice
const { data } = useAnalyticsQuery("spend_data", params);
return <LineChart queryKey="spend_data" parameters={params} />;

// ✅ Correct: let the chart fetch
return <LineChart queryKey="spend_data" parameters={params} />;
```

## LLM Checklist

Before finalizing code:

- [ ] `package.json` has `"type": "module"`
- [ ] `dev` script uses `NODE_ENV=development tsx watch server/index.ts`
- [ ] `await createApp({ plugins: [...] })` is used
- [ ] Charts use props NOT children
- [ ] Query parameters are memoized with `useMemo`
- [ ] Loading/error/empty states are explicit
