import * as path from 'node:path';
import { fileURLToPath } from 'node:url';
import { analytics, createApp, server } from '@databricks/app-kit';
import { appRouterMiddleware } from './trpc.js';
import type { Application } from 'express';

// Define __dirname equivalent for ES modules
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// define path to static files
const staticPath = path.resolve(__dirname, '..', 'client');

const app = await createApp({
  plugins: [
    server({
      watch: process.env.NODE_ENV === 'development',
      staticPath,
      autoStart: false,
    }),
    analytics({
      timeout: 20_000,
    }),
  ],
});

// Inject tRPC routes for functionality not covered by AppKit
app.server.extend((express: Application) => {
  express.use('/trpc', [appRouterMiddleware()]);
});

await app.server.start();
