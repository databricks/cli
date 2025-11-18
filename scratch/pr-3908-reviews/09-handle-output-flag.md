# Handle Global --output Flag

**Reviewer:** @pkosiec
**Date:** November 14, 2025
**Priority:** LOW - Can be separate PR

## Comment

**Location:** cmd/workspace/apps/logs.go:168

"As a follow-up, we'd need to handle the global `--output` flag as when you specify `--output json` it doesn't do anything. But again, that can be a separate PR, I can also help you with that later 👍"

## Action Items

1. This can be a separate PR (not a blocker for current PR)
2. Implement handling for `--output json` flag
3. Currently the flag is ignored
4. @pkosiec offered to help with this later

## Expected Behavior

- `databricks apps logs --output json` → Format log output as JSON

## Files Affected

- `cmd/workspace/apps/logs.go` (around line 168)

## Notes

**NOT A BLOCKER** - Can be done in a follow-up PR.
Consider deferring this to a separate issue/PR.
