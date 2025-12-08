/**
 * Query Result Schemas - Define the COLUMNS RETURNED by each SQL query.
 *
 * These schemas validate QUERY RESULTS, not input parameters.
 * - Input parameters are passed to useAnalyticsQuery() as the second argument
 * - These schemas define the shape of data[] returned by the query
 *
 * Example:
 *   SQL: SELECT name, age FROM users WHERE city = :city
 *   Schema: z.array(z.object({ name: z.string(), age: z.number() }))
 *   Usage: useAnalyticsQuery('users', { city: 'NYC' })
 *                                       ^ input params   ^ schema validates this result
 */

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
