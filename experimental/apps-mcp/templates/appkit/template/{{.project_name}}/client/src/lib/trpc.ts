import { createTRPCClient, httpBatchLink, loggerLink } from '@trpc/client';
import type { appRouter } from '../../../server/trpc';
import superjson from 'superjson';

export const trpc = createTRPCClient<typeof appRouter>({
  links: [
    loggerLink({
      enabled: (opts) => typeof window !== 'undefined' || (opts.direction === 'down' && opts.result instanceof Error),
    }),
    httpBatchLink({ url: '/trpc', transformer: superjson }),
  ],
});
