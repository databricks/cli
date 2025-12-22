/**
 * SQL Type Helpers
 *
 * IMPORTANT: Databricks SQL returns ALL numeric values as STRINGS in JSON.
 * Always use these helpers for numeric fields from SQL queries.
 *
 * ❌ row.amount.toFixed(2)           // FAILS: string has no toFixed
 * ✅ toNumber(row.amount).toFixed(2) // WORKS
 * ✅ formatCurrency(row.amount)      // WORKS: "$123.45"
 */

export interface QueryResult {
  value: string;
}

/** Convert SQL string to number. Use for ANY numeric field from queries. */
export const toNumber = (val: string | number | null | undefined): number =>
  Number(val || 0);

/** Format as currency. Example: "123.45" → "$123.45" */
export const formatCurrency = (val: string | number | null | undefined): string =>
  `$${toNumber(val).toFixed(2)}`;

/** Format as percentage. Example: "85.5" → "85.5%" */
export const formatPercent = (val: string | number | null | undefined): string =>
  `${toNumber(val).toFixed(1)}%`;
