# SQL Query Files

**IMPORTANT**: ALWAYS use SQL files in `config/queries/` for data retrieval. NEVER use tRPC for SQL queries.

- Store ALL SQL queries in `config/queries/` directory
- Name files descriptively: `trip_statistics.sql`, `user_metrics.sql`, `sales_by_region.sql`
- Reference by filename (without extension) in `useAnalyticsQuery` or directly in a visualization component passing it as `queryKey`
- App Kit automatically executes queries against configured Databricks warehouse
- Benefits: Built-in caching, proper connection pooling, better performance

## Query Schemas

Once the schema and the result of a query has been discovered, create its corresponding type in `config/queries/schema.ts` using a zod schema.

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

## SQL Type Handling

**IMPORTANT**: Numeric fields from Databricks SQL (especially `ROUND()`, `AVG()`, `SUM()`) are returned as strings in JSON. Always convert before using numeric methods:

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
const { data } = useAnalyticsQuery('filtered_data', {
  min_value: minValue,
  max_value: maxValue,
  category: category,
  optional_filter: optionalFilter || '',  // empty string for optional params
});
```

### Date Parameters

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

### Optional Date Parameters - Use Sentinel Dates

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

**Why Sentinel Dates Work:**
- `1900-01-01` is before any real data (effectively no lower bound filter)
- `9999-12-31` is after any real data (effectively no upper bound filter)
- Always valid DATE types, so no parameter validation errors
- All real dates fall within this range, so no filtering occurs

**Parameter Types Summary:**
- **Strings/Numbers**: Use directly in SQL with `:param_name`
- **Dates**: Format as `YYYY-MM-DD`, use with `CAST(:param AS DATE)` in SQL
- **Optional Strings**: Use empty string default, check with `(:param = '' OR column = :param)`
- **Optional Dates**: Use sentinel dates (`'1900-01-01'` and `'9999-12-31'`) instead of empty strings
