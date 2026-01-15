# AppKit Project Scaffolding

## Canonical Project Layout

```
my-app/
├── server/
│   ├── index.ts              # backend entry point
│   └── .env                  # local dev env vars (gitignore)
├── client/
│   ├── index.html
│   ├── vite.config.ts
│   └── src/
│       ├── main.tsx
│       └── App.tsx
├── config/
│   └── queries/
│       └── my_query.sql
├── app.yaml
├── package.json
└── tsconfig.json
```

## package.json

```json
{
  "name": "my-app",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "scripts": {
    "dev": "NODE_ENV=development tsx watch server/index.ts",
    "build": "npm run build:server && npm run build:client",
    "build:server": "tsdown --out-dir build server/index.ts",
    "build:client": "tsc -b && vite build --config client/vite.config.ts",
    "start": "node build/index.mjs"
  },
  "dependencies": {
    "@databricks/appkit": "^0.1.2",
    "@databricks/appkit-ui": "^0.1.2",
    "react": "^19.2.3",
    "react-dom": "^19.2.3"
  },
  "devDependencies": {
    "@types/node": "^20.0.0",
    "@types/react": "^19.0.0",
    "@types/react-dom": "^19.0.0",
    "@vitejs/plugin-react": "^5.1.1",
    "tsdown": "^0.15.7",
    "tsx": "^4.19.0",
    "typescript": "~5.6.0",
    "vite": "^7.2.4"
  }
}
```

## client/index.html

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>My App</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

## client/src/main.tsx

```tsx
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
```

## client/vite.config.ts

```ts
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
});
```

## tsconfig.json

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "lib": ["ES2022", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "moduleResolution": "bundler",
    "jsx": "react-jsx",
    "strict": true,
    "skipLibCheck": true,
    "noEmit": true,
    "allowImportingTsExtensions": true,
    "verbatimModuleSyntax": true
  },
  "include": ["server", "client/src"]
}
```

## server/index.ts

```ts
import { createApp, server, analytics } from "@databricks/appkit";

await createApp({
  plugins: [server(), analytics({})],
});
```

## app.yaml (Databricks Apps Config)

```yaml
command:
  - node
  - build/index.mjs
env:
  - name: DATABRICKS_WAREHOUSE_ID
    valueFrom: sql-warehouse
```

## Type Generation

### Vite Plugin

```ts
// client/vite.config.ts
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { appKitTypesPlugin } from "@databricks/appkit";

export default defineConfig({
  plugins: [
    react(),
    appKitTypesPlugin({
      outFile: "src/appKitTypes.d.ts",
      watchFolders: ["../config/queries"],
    }),
  ],
});
```

### CLI

```bash
npx appkit-generate-types [rootDir] [outFile] [warehouseId]
npx appkit-generate-types . client/src/appKitTypes.d.ts
npx appkit-generate-types --no-cache  # Force regeneration
```

## Running the App

```bash
# Install dependencies
npm install

# Development (starts backend + Vite dev server)
npm run dev

# Production build
npm run build
npm start
```

## Adding to Existing React/Vite App

1. Install dependencies:
```bash
npm install @databricks/appkit @databricks/appkit-ui react react-dom
npm install -D tsx tsdown vite @vitejs/plugin-react typescript
```

2. Move Vite app to `client/` folder:
```
client/index.html
client/vite.config.ts
client/src/
```

3. Create `server/index.ts`:
```ts
import { createApp, server, analytics } from "@databricks/appkit";

await createApp({
  plugins: [server(), analytics({})],
});
```

4. Update `package.json` scripts as shown above
