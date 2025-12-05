/**
 * This file is meant to contain the schema for the queries that are used in the app.
 * It is used to validate the parameters that are passed to the queries.
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
