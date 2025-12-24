import { createApp, server, {{.plugin_import}} } from '@databricks/appkit';

createApp({
  plugins: [
    server(),
    {{.plugin_usage}},
  ],
}).catch(console.error);
