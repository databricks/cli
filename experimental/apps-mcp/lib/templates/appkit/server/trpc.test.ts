import { describe, it, expect } from 'vitest';
import { appRouter } from './trpc';

describe('tRPC Handler', () => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any, @typescript-eslint/no-unsafe-argument
  const caller = appRouter.createCaller({} as any);

  it('should return health check status', async () => {
    const health = await caller.healthcheck();

    expect(health).toBeDefined();
    expect(health.status).toBe('ok');
    expect(health.timestamp).toBeDefined();
    expect(new Date(health.timestamp).getTime()).toBeGreaterThan(0);
  });
});
