# Testing Guidelines

## Unit Tests (Vitest)

**CRITICAL**: Use vitest for all tests. Put tests next to the code (e.g. src/\*.test.ts)

```typescript
import { describe, it, expect } from 'vitest';

describe('Feature Name', () => {
  it('should do something', () => {
    expect(true).toBe(true);
  });

  it('should handle async operations', async () => {
    const result = await someAsyncFunction();
    expect(result).toBeDefined();
  });
});
```

**Best Practices:**
- Use `describe` blocks to group related tests
- Use `it` for individual test cases
- Use `expect` for assertions
- Tests run with `npm test` (runs `vitest run`)

❌ **Do not write unit tests for:**
- SQL files under `config/queries/` - little value in testing static SQL
- Types associated with queries - these are just schema definitions

## Smoke Test (Playwright)

The template includes a smoke test at `tests/smoke.spec.ts` that verifies the app loads correctly.

**What the smoke test does:**
- Opens the app
- Waits for data to load (SQL query results)
- Verifies key UI elements are visible
- Captures screenshots and console logs to `.smoke-test/` directory
- Always captures artifacts, even on test failure

**When customizing the app**, update `tests/smoke.spec.ts` to match your UI:
- Change heading selector to match your app title (replace 'Minimal Databricks App')
- Update data assertions to match your query results (replace 'hello world' check)
- Keep the test simple - just verify app loads and displays data
- The default test expects specific template content; update these expectations after customization

**Keep smoke tests simple:**
- Only verify that the app loads and displays initial data
- Wait for key elements to appear (page title, main content)
- Capture artifacts for debugging
- Run quickly (< 5 seconds)

**For extended E2E tests:**
- Create separate test files in `tests/` directory (e.g., `tests/user-flow.spec.ts`)
- Use `npm run test:e2e` to run all Playwright tests
- Keep complex user flows, interactions, and edge cases out of the smoke test
