export interface QueryResult {
  value: string;
}

// SQL type helpers - numeric fields from Databricks SQL return as strings
export const toNumber = (val: string | number | null | undefined): number =>
  Number(val || 0);

export const formatCurrency = (val: string | number | null | undefined): string =>
  `$${toNumber(val).toFixed(2)}`;

export const formatPercent = (val: string | number | null | undefined): string =>
  `${toNumber(val).toFixed(1)}%`;
