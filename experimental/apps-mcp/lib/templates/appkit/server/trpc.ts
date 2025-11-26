import { initTRPC } from '@trpc/server';
import { createExpressMiddleware } from '@trpc/server/adapters/express';
import superjson from 'superjson';
import type express from 'express';
import { getRequestContext } from '@databricks/app-kit';
import { z } from 'zod';

const t = initTRPC.create({
  transformer: superjson,
});

const publicProcedure = t.procedure;
const router = t.router;

export const appRouter = router({
  healthcheck: publicProcedure.query(() => {
    return { status: 'ok', timestamp: new Date().toISOString() };
  }),
  // example RPC for querying a serving endpoint
  queryModel: publicProcedure
    .input(
      z.object({
        prompt: z.string(),
      })
    )
    .query(async ({ input: { prompt } }) => {
      const { serviceDatabricksClient: client } = getRequestContext();
      const endpointName = 'databricks-gpt-oss-120b';
      const response = await client.servingEndpoints.query({
        name: endpointName,
        messages: [
          {
            role: 'system',
            content: 'You are a helpful assistant that can answer questions and help with tasks.',
          },
          {
            role: 'user',
            content: prompt,
          },
        ],
      });

      const content = response.choices?.[0]?.message?.content as unknown as { type: string; text: string }[];
      if (Array.isArray(content)) {
        const last = content[content.length - 1];
        return last?.text || '';
      }

      // Fallback: stringify the entire response
      return JSON.stringify(response);
    }),
});

export function appRouterMiddleware(): express.RequestHandler {
  return createExpressMiddleware({ router: appRouter });
}
