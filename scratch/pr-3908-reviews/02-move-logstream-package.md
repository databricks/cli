# Move logstream Package to libs/apps

**Reviewer:** @pietern
**Date:** November 17, 2025
**Priority:** MEDIUM

## Comment

"Please move `logstream` to `libs/apps` to make it clearer that it is specific to apps."

## Action Items

1. Move the entire `libs/logstream` package to `libs/apps/logstream`
2. Update all import statements across the codebase
3. Update any documentation that references the old path
4. Verify tests still pass after the move

## Files Affected

- `libs/logstream/` → `libs/apps/logstream/`
- Any files importing `github.com/databricks/cli/libs/logstream`
- Documentation files
