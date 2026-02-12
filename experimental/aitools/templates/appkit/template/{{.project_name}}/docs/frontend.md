# Frontend Guidelines

## Visualization Components

Components from `@databricks/appkit-ui/react` handle data fetching, loading states, and error handling internally.

Available: `AreaChart`, `BarChart`, `LineChart`, `PieChart`, `RadarChart`, `DataTable`

**Basic Usage:**

```typescript
import { BarChart, LineChart, DataTable, Card, CardContent, CardHeader, CardTitle } from '@databricks/appkit-ui/react';
import { sql } from "@databricks/appkit-ui/js";

function MyDashboard() {
  return (
    <div>
      <Card>
        <CardHeader><CardTitle>Sales by Region</CardTitle></CardHeader>
        <CardContent>
          <BarChart queryKey="sales_by_region" parameters={{}} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader><CardTitle>Revenue Trend</CardTitle></CardHeader>
        <CardContent>
          <LineChart queryKey="revenue_over_time" parameters={{ months: sql.number(12) }} />
        </CardContent>
      </Card>
    </div>
  );
}
```

Components automatically fetch data, show loading states, display errors, and render with sensible defaults.

**⚠️ CRITICAL: AppKit charts do NOT support children**

```typescript
// ❌ WRONG - AppKit charts don't accept Recharts children
import { BarChart } from '@databricks/appkit-ui/react';
import { Bar, XAxis, YAxis } from 'recharts';
<BarChart queryKey="sales" parameters={{}}>
  <XAxis dataKey="region" />  // TypeScript error!
  <Bar dataKey="revenue" />    // TypeScript error!
</BarChart>

// ✅ CORRECT - Use simple self-closing syntax
<BarChart queryKey="sales" parameters={{}} />
```

AppKit charts auto-configure axes, tooltips, and styling. Do NOT import Recharts components.

Databricks brand colors: `['#40d1f5', '#4462c9', '#EB1600', '#0B2026', '#4A4A4A', '#353a4a']`

**❌ Don't double-fetch:**

```typescript
// WRONG - redundant fetch
const { data } = useAnalyticsQuery('sales_data', {});
return <BarChart queryKey="sales_data" parameters={{}} />;

// CORRECT - let component handle it
return <BarChart queryKey="sales_data" parameters={{}} />;
```

## Layout Structure

```tsx
<div className="container mx-auto p-4">
  <h1 className="text-2xl font-bold mb-4">Page Title</h1>
  <form className="space-y-4 mb-8">{/* form inputs */}</form>
  <div className="grid gap-4">{/* list items */}</div>
</div>
```

## Component Organization

- Shared UI components: `@databricks/appkit-ui/react`
- Feature components: `client/src/components/FeatureName.tsx`
- Split components when logic exceeds ~100 lines or component is reused

## Radix UI Constraints

- `SelectItem` cannot have `value=""`. Use sentinel value like `"all"` for "show all" options.

## Map Libraries (react-leaflet)

For maps with React 19, use react-leaflet v5:

```bash
npm install react-leaflet@^5.0.0 leaflet @types/leaflet
```

```typescript
import 'leaflet/dist/leaflet.css';
```

## Best Practices

- Use shadcn/radix components (Button, Input, Card, etc.) for consistent UI, import them from `@databricks/appkit-ui/react`.
- **Use skeleton loaders**: Always use `<Skeleton>` components instead of plain "Loading..." text
- Define result types in `shared/types.ts` for reuse between frontend and backend
- Handle nullable fields: `value={field || ''}` for inputs
- Type callbacks explicitly: `onChange={(e: React.ChangeEvent<HTMLInputElement>) => ...}`
- Forms should have loading states: `disabled={isLoading}`
- Show empty states with helpful text when no data exists
