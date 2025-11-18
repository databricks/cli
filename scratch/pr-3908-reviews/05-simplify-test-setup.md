# Simplify Test Setup

**Reviewer:** @pkosiec
**Date:** November 14, 2025
**Priority:** LOW (may be obsolete if token handling is removed)

## Comment

**Location:** libs/auth/token_loader_test.go:53

"I'd just skip that part, and use the `mockOAuthEndpointSupplier`, and then check the length of the opts, without counting the calls etc. 👍 No need for it IMO."

## Action Items

1. Check if this is still relevant after token handling refactor (see item #01)
2. If still relevant:
   - Use `mockOAuthEndpointSupplier` instead of complex setup
   - Check the length of opts
   - Remove call counting logic
   - Simplify the test to be more straightforward

## Files Affected

- `libs/auth/token_loader_test.go` (line 53)

## Notes

This may become obsolete if the token handling code is removed entirely per @pietern's review.
