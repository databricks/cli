# Convert Tests to Table Test Format

**Reviewer:** @pkosiec
**Date:** November 14, 2025
**Priority:** LOW (may be obsolete if token handling is removed)

## Comment

**Location:** libs/auth/token_loader_test.go:206

"Those are moved tests from the `token_test.go`, correct? Could you please keep them defined as table tests? It's much easier to maintain such cases in table test format. Thanks!"

## Action Items

1. Check if this is still relevant after token handling refactor (see item #01)
2. If still relevant: Convert the test cases at line 206 to table test format
3. Follow existing table test patterns in the codebase

## Files Affected

- `libs/auth/token_loader_test.go`

## Notes

This may become obsolete if the token handling code is removed entirely per @pietern's review.
