# Remove PersistentAuthOpts Mutation Check

**Reviewer:** @pkosiec
**Date:** November 14, 2025
**Priority:** LOW (may be obsolete if token handling is removed)

## Comment

**Location:** libs/auth/token_loader.go:21

"If I understand correctly, this is defined purely for a single unit test that checks if the `PersistentAuthOpts` aren't mutated, correct?

I'd get rid of this as we don't need to do that. Please check my next comment for more details 👍"

## Action Items

1. Check if this is still relevant after token handling refactor (see item #01)
2. If still relevant: Remove the code at line 21 that's defined purely for testing
3. Remove the corresponding test that checks for mutation

## Files Affected

- `libs/auth/token_loader.go` (line 21)
- Related test code

## Notes

This may become obsolete if the token handling code is removed entirely per @pietern's review.
