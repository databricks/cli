# Address Comment at logs.go:240

**Reviewer:** @pkosiec
**Date:** November 14, 2025
**Priority:** UNKNOWN

## Comment

**Location:** cmd/workspace/apps/logs.go:240

"I think this comment is still applicable ☝️"

## Action Items

1. Need to find what the previous comment was at this location
2. Check the PR conversation history or previous review rounds
3. Address whatever that comment requested

## Files Affected

- `cmd/workspace/apps/logs.go` (line 240)

## Notes

This references a previous comment that wasn't included in the Nov 14+ review batch.
Need to look at earlier review comments to find the original issue.

To find the original comment, run:
```bash
gh api repos/databricks/cli/pulls/3908/comments --jq '.[] | select(.path == "cmd/workspace/apps/logs.go" and .line == 240)'
```

Or check the PR conversation at: https://github.com/databricks/cli/pull/3908
