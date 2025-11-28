import { describe, it, expect, vi, beforeEach } from 'vitest';

// Mock @databricks/app-kit before importing the router
const mockServingEndpointsQuery = vi.fn();
vi.mock('@databricks/app-kit', () => ({
  getRequestContext: vi.fn(() => ({
    serviceDatabricksClient: {
      servingEndpoints: {
        query: mockServingEndpointsQuery,
      },
    },
  })),
}));

import { appRouter } from './trpc.js';

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

describe('queryModel', () => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any, @typescript-eslint/no-unsafe-argument
  const caller = appRouter.createCaller({} as any);

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should extract text from model response', async () => {
    // Mock the serving endpoint query response
    const mockQueryResponse = {
      choices: [
        {
          message: {
            content: [
              {
                type: 'reasoning',
                summary: [{ type: 'summary_text', text: 'Some reasoning' }],
              },
              {
                type: 'text',
                text: 'This is the response text',
              },
            ],
          },
        },
      ],
    };

    mockServingEndpointsQuery.mockResolvedValue(mockQueryResponse);

    const result = await caller.queryModel({ prompt: 'Test prompt' });

    expect(result).toBe('This is the response text');
    expect(mockServingEndpointsQuery).toHaveBeenCalledWith({
      name: 'databricks-gpt-oss-120b',
      messages: [
        {
          role: 'system',
          content: 'You are a helpful assistant that can answer questions and help with tasks.',
        },
        {
          role: 'user',
          content: 'Test prompt',
        },
      ],
    });
  });
});
