# @databricks/bundles

TypeScript support for Databricks Asset Bundles extends [Databricks Asset Bundles](https://docs.databricks.com/aws/en/dev-tools/bundles/) so that you can:
- Define jobs and pipelines as TypeScript code with full type safety
- Dynamically create jobs and pipelines using metadata and programming constructs
- Modify jobs and pipelines defined in YAML or TypeScript during bundle deployment
- Get IDE autocomplete and type checking for all Databricks resources

## Status

🚧 **This is an experimental migration from Python to TypeScript.** The API is subject to change.

## Requirements

- Node.js 18.0.0 or higher
- [Databricks CLI](https://github.com/databricks/cli) version 0.271.0 or above

## Installation

```bash
npm install @databricks/bundles
# or
yarn add @databricks/bundles
# or
pnpm add @databricks/bundles
```

## Getting Started

### 1. Authenticate with Databricks

```bash
databricks configure
```

### 2. Create a Bundle Project

```bash
databricks bundle init experimental-jobs-as-code
```

### 3. Define Resources in TypeScript

Create a file `resources/index.ts`:

```typescript
import { defineResources, Job, Task } from "@databricks/bundles";

export function loadResources() {
  return defineResources({
    jobs: {
      myJob: Job({
        name: "My TypeScript Job",
        tasks: [
          Task({
            taskKey: "main",
            notebookTask: {
              notebookPath: "/path/to/notebook",
            },
            newCluster: {
              sparkVersion: "13.3.x-scala2.12",
              nodeTypeId: "i3.xlarge",
              numWorkers: 2,
            },
          }),
        ],
      }),
    },
  });
}
```

### 4. Configure databricks.yml

```yaml
bundle:
  name: my-bundle

python:
  resources:
    - "resources:loadResources"
```

### 5. Deploy

```bash
databricks bundle deploy
```

## Development

### Build

```bash
npm run build
```

### Test

```bash
npm test                # Run tests once
npm run test:watch      # Run tests in watch mode
npm run test:coverage   # Run tests with coverage
```

### Type Check

```bash
npm run typecheck
```

### Lint

```bash
npm run lint
npm run lint:fix
```

### Format

```bash
npm run format
npm run format:check
```

## Project Structure

```
typescript/
├── src/
│   ├── core/          # Core framework (Bundle, Resources, Variables)
│   ├── resources/     # Resource type definitions
│   ├── build/         # Build system for CLI integration
│   └── codegen/       # Code generator from OpenAPI specs
├── generated/         # Auto-generated type definitions
│   ├── jobs/
│   ├── pipelines/
│   ├── schemas/
│   └── volumes/
└── tests/            # Test suite
```

## Migration from Python

If you're migrating from the Python version (`databricks-bundles`), see [MIGRATION.md](./MIGRATION.md) for a detailed guide.

## License

See [LICENSE](../LICENSE)

## Privacy Notice

Databricks CLI use is subject to the [Databricks License](https://github.com/databricks/cli/blob/main/LICENSE) and [Databricks Privacy Notice](https://www.databricks.com/legal/privacynotice), including any Usage Data provisions.
