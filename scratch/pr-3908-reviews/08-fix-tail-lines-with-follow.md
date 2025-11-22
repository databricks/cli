# Fix tail-lines Flag When Used with Follow

**Reviewer:** @pkosiec
**Date:** November 14, 2025
**Priority:** HIGH - Bug fix

## Comment

**Location:** cmd/workspace/apps/logs.go:154

"I tested the flag and seems to not work when running together with `-f` flag - it always shows me all the logs instead of last n lines 🤔 It works properly though when I don't have the `follow` flag specified."

Also mentioned at line 154:
"I'm not sure if that flag works properly - it always shows me all logs instead of last n lines 🤔"

## Action Items

1. Debug why `--tail-lines` doesn't work correctly with `-f` (follow) flag
2. The flag works correctly when follow is NOT specified
3. When both flags are used together, it should only show the last N lines and then start following
4. Fix the logic in `cmd/workspace/apps/logs.go` around line 154

## Expected Behavior

- `databricks apps logs --tail-lines 10` → Show last 10 lines only
- `databricks apps logs -f` → Follow/stream all logs
- `databricks apps logs -f --tail-lines 10` → Show last 10 lines, THEN continue following new logs

## Files Affected

- `cmd/workspace/apps/logs.go` (around line 154)

## Testing

Test all three scenarios above to ensure they work correctly.
