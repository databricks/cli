import { createApp, server } from '@databricks/appkit';

createApp({
  plugins: [server(),],
}).catch(console.error);
