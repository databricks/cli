# SQL Query Files

**IMPORTANT**: ALWAYS use SQL files in `config/queries/` for data retrieval. NEVER use tRPC for SQL queries.

- Store ALL SQL queries in `config/queries/` directory
- Name files descriptively: `trip_statistics.sql`, `user_metrics.sql`, `sales_by_region.sql`
- Reference by filename (without extension) in `useAnalyticsQuery` or directly in a visualization component passing it as `queryKey`
- App Kit automatically executes queries against configured Databricks warehouse
- Benefits: Built-in caching, proper connection pooling, better performance

## Query Schemas

Define the shape of QUERY RESULTS (not input parameters) in `config/queries/schema.ts` using Zod schemas.

- **These schemas validate the COLUMNS RETURNED by SQL queries**
- Input parameters are passed separately to `useAnalyticsQuery()` as the second argument
- Schema field names must match your SQL SELECT column names/aliases

Example:

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

**IMPORTANT: Refreshing Type Definitions**

After adding or modifying query schemas in `config/queries/schema.ts`:

1. **DO NOT** manually edit `client/src/appKitTypes.d.ts` - this file is auto-generated
2. Run `npm run dev` to automatically regenerate the TypeScript type definitions
3. The dev server will scan your SQL files and schema definitions and update `appKitTypes.d.ts` accordingly

## SQL Type Handling (Critical)

**ALL numeric values from Databricks SQL are returned as STRINGS in JSON responses.** This includes results from `ROUND()`, `AVG()`, `SUM()`, `COUNT()`, etc. Always convert before using numeric methods:

```typescript
// ❌ WRONG - fails at runtime
<span>{row.total_amount.toFixed(2)}</span>

// ✅ CORRECT - convert to number first
<span>{Number(row.total_amount).toFixed(2)}</span>
```

**Helper Functions:**

Use the helpers from `shared/types.ts` for consistent formatting:

```typescript
import { toNumber, formatCurrency, formatPercent } from '../../shared/types';

// Convert to number
const amount = toNumber(row.amount);  // "123.45" → 123.45

// Format as currency
const formatted = formatCurrency(row.amount);  // "123.45" → "$123.45"

// Format as percentage
const percent = formatPercent(row.rate);  // "85.5" → "85.5%"
```

## Query Parameterization

SQL queries can accept parameters to make them dynamic and reusable.

**Key Points:**
- Parameters use colon prefix: `:parameter_name`
- Databricks infers types from values automatically
- For optional string parameters, use pattern: `(:param = '' OR column = :param)`
- **For optional date parameters, use sentinel dates** (`'1900-01-01'` and `'9999-12-31'`) instead of empty strings

### SQL Parameter Syntax

```sql
-- config/queries/filtered_data.sql
SELECT *
FROM my_table
WHERE column_value >= :min_value
  AND column_value <= :max_value
  AND category = :category
  AND (:optional_filter = '' OR status = :optional_filter)
```

### Frontend Parameter Passing

```typescript
import { sql } from "@databricks/app-kit-ui/js";

const { data } = useAnalyticsQuery('filtered_data', {
  min_value: sql.number(minValue),
  max_value: sql.number(maxValue),
  category: sql.string(category),
  optional_filter: sql.string(optionalFilter || ''),  // empty string for optional params
});
```

### Date Parameters

Use `sql.date()` for date parameters with `YYYY-MM-DD` format strings.

**Frontend - Using Date Parameters:**

```typescript
import { sql } from '@databricks/app-kit-ui/js';
import { useState } from 'react';

function MyComponent() {
  const [startDate, setStartDate] = useState<string>('2016-02-01');
  const [endDate, setEndDate] = useState<string>('2016-02-29');

  const queryParams = {
    start_date: sql.date(startDate),  // Pass YYYY-MM-DD string to sql.date()
    end_date: sql.date(endDate),
  };

  const { data } = useAnalyticsQuery('my_query', queryParams);

  // ...
}
```

**SQL - Date Filtering:**

```sql
-- Filter by date range using DATE() function
SELECT COUNT(*) as trip_count
FROM samples.nyctaxi.trips
WHERE DATE(tpep_pickup_datetime) >= :start_date
  AND DATE(tpep_pickup_datetime) <= :end_date
```

**Date Helper Functions:**

```typescript
// Helper to get dates relative to today
const daysAgo = (n: number) => {
  const date = new Date(Date.now() - n * 86400000);
  return sql.date(date)
};

const params = {
  start_date: daysAgo(7),             // 7 days ago
  end_date: sql.date(daysAgo(0)),     // Today
};
```

### Optional Date Parameters - Use Sentinel Dates

Databricks App Kit validates parameter types before query execution. **DO NOT use empty strings (`''`) for optional date parameters** as this causes validation errors.

**✅ CORRECT - Use Sentinel Dates:**

```typescript
// Frontend: Use sentinel dates for "no filter" instead of empty strings
const revenueParams = {
  group_by: 'month',
  start_date: sql.date('1900-01-01'),  // Sentinel: effectively no lower bound
  end_date: sql.date('9999-12-31'),    // Sentinel: effectively no upper bound
  country: sql.string(country || ''),
  property_type: sql.string(propertyType || ''),
};
```

```sql
-- SQL: Simple comparison since sentinel dates are always valid
WHERE b.check_in >= CAST(:start_date AS DATE)
  AND b.check_in <= CAST(:end_date AS DATE)
```

**Why Sentinel Dates Work:**
- `1900-01-01` is before any real data (effectively no lower bound filter)
- `9999-12-31` is after any real data (effectively no upper bound filter)
- Always valid DATE types, so no parameter validation errors
- All real dates fall within this range, so no filtering occurs

**Parameter Types Summary:**
- ALWAYS use sql.* helper functions from the `@databricks/app-kit-ui/js` package to define SQL parameters
- **Strings/Numbers**: Use directly in SQL with `:param_name`
- **Dates**: Use with `CAST(:param AS DATE)` in SQL
- **Optional Strings**: Use empty string default, check with `(:param = '' OR column = :param)`
- **Optional Dates**: Use sentinel dates (`sql.date('1900-01-01')` and `sql.date('9999-12-31')`) instead of empty strings
