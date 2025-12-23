import { analytics, createApp, server } from '@databricks/app-kit';
import { appRouterMiddleware } from './trpc.js';
import type { Application } from 'express';

const app = await createApp({
  plugins: [
    server({
      autoStart: false,
    }),
    analytics(),
  ],
});

// Inject tRPC routes for functionality not covered by AppKit
app.server.extend((express: Application) => {
  express.use('/trpc', [appRouterMiddleware()]);
});

await app.server.start();
