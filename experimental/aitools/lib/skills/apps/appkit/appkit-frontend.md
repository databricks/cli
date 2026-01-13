# AppKit Frontend (@databricks/appkit-ui)

## Imports

```tsx
// React-facing APIs
import { useAnalyticsQuery, Card, Skeleton, BarChart } from "@databricks/appkit-ui/react";

// Non-React utilities (sql markers, arrow, SSE)
import { sql } from "@databricks/appkit-ui/js";
```

## Charts

All charts support:
- **Query mode**: `queryKey` + `parameters`
- **Data mode**: `data` (inline JSON, no server)

**Available charts:** `BarChart`, `LineChart`, `AreaChart`, `PieChart`, `DonutChart`, `HeatmapChart`, `ScatterChart`, `RadarChart`

### Chart Props Reference

```tsx
<BarChart
  queryKey="sales_by_region"
  parameters={{}}
  xKey="region"                    // X-axis field
  yKey={["revenue", "expenses"]}   // Y-axis field(s) - string or string[]
  colors={['#40d1f5', '#4462c9']}  // Custom colors
  stacked                          // Stack bars
  orientation="horizontal"         // "vertical" (default) | "horizontal"
  showLegend                       // Show legend
  height={400}                     // Height in pixels (default: 300)
/>

<LineChart
  queryKey="trend_data"
  parameters={{}}
  xKey="date"
  yKey="value"
  smooth                           // Smooth curves (default: true)
  showSymbol={false}               // Hide data point markers
/>
```

### CRITICAL: Charts Do NOT Accept Children

```tsx
// ❌ WRONG - AppKit charts are NOT Recharts wrappers
import { BarChart } from "@databricks/appkit-ui/react";
import { Bar, XAxis, YAxis } from "recharts";

<BarChart queryKey="data" parameters={{}}>
  <XAxis dataKey="x" />  // ❌ TypeScript error
  <Bar dataKey="y" />    // ❌ Not supported
</BarChart>

// ✅ CORRECT - use props instead
<BarChart
  queryKey="data"
  parameters={{}}
  xKey="x"
  yKey="y"
/>
```

## SQL Helpers (`sql.*`)

Use for typed parameters (returns `{ __sql_type, value }`):

```ts
sql.string(value)     // STRING
sql.number(value)     // NUMERIC
sql.boolean(value)    // BOOLEAN
sql.date(value)       // DATE ("YYYY-MM-DD" or Date)
sql.timestamp(value)  // TIMESTAMP
sql.binary(value)     // STRING (hex) - use UNHEX(:param) in SQL
```

## useAnalyticsQuery

```tsx
const { data, loading, error } = useAnalyticsQuery(
  queryName: string,
  params: Record<string, SQLTypeMarker>
);
```

**Options:**
- `format?: "JSON" | "ARROW"` (default `"JSON"`)
- `autoStart?: boolean` (default `true`)

**Limitations:**
- No `enabled` option - use conditional rendering
- No `refetch()` - change parameters or re-mount

## useChartData

```tsx
const { data, loading, error } = useChartData({
  queryKey: "my_query",
  parameters: {},
  format: "auto",  // "json" | "arrow" | "auto"
  transformer: (data) => transformedData,
});
```

## DataTable

```tsx
import { DataTable } from "@databricks/appkit-ui/react";

<DataTable
  queryKey="users_list"
  parameters={{}}
  filterColumn="email"
  filterPlaceholder="Filter by email..."
  pageSize={25}
  pageSizeOptions={[10, 25, 50, 100]}
/>
```

## connectSSE (Custom Streaming)

```tsx
import { connectSSE } from "@databricks/appkit-ui/js";

connectSSE({
  url: endpoint,
  payload: { key: "value" },  // POST if provided
  onMessage: async ({ data }) => { /* handle */ },
  onError: (error) => { /* handle */ },
  signal: controller.signal,
  maxRetries: 3,
  retryDelay: 2000,
  timeout: 300000,
});
```

## ArrowClient (Advanced)

```tsx
import { ArrowClient } from "@databricks/appkit-ui/js";

const table = await ArrowClient.processArrowBuffer(buffer);
const columns = ArrowClient.extractArrowColumns(table);
const { xData, yDataMap } = ArrowClient.extractChartData(table, "date", ["value"]);
```

## UI Components (Primitives)

Import from `@databricks/appkit-ui/react`:

**Layout:** `Card`, `CardHeader`, `CardTitle`, `CardDescription`, `CardContent`, `CardFooter`

**Forms:** `Button`, `Input`, `Textarea`, `Select`, `SelectTrigger`, `SelectValue`, `SelectContent`, `SelectItem`, `Checkbox`, `RadioGroup`, `Switch`, `Label`

**Feedback:** `Skeleton`, `Spinner`, `Progress`, `Alert`, `Badge`, `Empty`

**Overlays:** `Dialog`, `DialogTrigger`, `DialogContent`, `DialogHeader`, `DialogTitle`, `DialogDescription`, `DialogFooter`, `Popover`, `Tooltip`, `TooltipTrigger`, `TooltipContent`, `TooltipProvider`, `Sheet`

**Navigation:** `Tabs`, `TabsList`, `TabsTrigger`, `TabsContent`, `Accordion`, `Breadcrumb`, `NavigationMenu`

**Data Display:** `Table`, `DataTable`, `ScrollArea`, `Separator`

### Radix Constraint

`SelectItem` cannot have `value=""`. Use a sentinel value like `"all"`:

```tsx
<SelectItem value="all">All items</SelectItem>  // ✅
<SelectItem value="">All items</SelectItem>     // ❌ Error
```

### TooltipProvider Requirement

Wrap root with `TooltipProvider` if using tooltips:

```tsx
import { TooltipProvider } from "@databricks/appkit-ui/react";

function App() {
  return (
    <TooltipProvider>
      {/* Your app */}
    </TooltipProvider>
  );
}
```

## Stylesheet

```css
/* main.css */
@import "@databricks/appkit-ui/styles.css";
```

### Theme Customization

Override CSS variables for both light and dark modes:

```css
:root {
  --radius: 0.625rem;
  --background: oklch(1 0 0);
  --foreground: oklch(0.141 0.005 285.823);
  --primary: oklch(0.21 0.006 285.885);
  --destructive: oklch(0.577 0.245 27.325);
  /* ... see full list in llms.txt */
}

@media (prefers-color-scheme: dark) {
  :root {
    --background: oklch(0.141 0.005 285.823);
    --foreground: oklch(0.985 0 0);
    /* ... */
  }
}
```

## SQL Result Types

Databricks SQL JSON results may return numeric fields as strings (especially `DECIMAL`). Convert explicitly:

```ts
const value = Number(row.amount);
```

For large datasets, prefer `format: "ARROW"` for better numeric fidelity.
