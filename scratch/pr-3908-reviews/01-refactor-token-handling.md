# Refactor Token Handling (MAJOR)

**Reviewer:** @pietern
**Date:** November 17, 2025
**Priority:** HIGH - This is a major architectural change

## Comment

"The token changes (`cmd/auth/token_*` and `libs/auth/token_*`) are not necessary. You can use the `TokenSource` function on the SDK configuration directly to get a fresh OAuth token. I confirmed that the following patch works: https://gist.github.com/pietern/41d02acc2907416244789c7408fb232f"

## Action Items

1. Remove the custom token handling code from `cmd/auth/token_*`
2. Remove the custom token handling code from `libs/auth/token_*`
3. Use the SDK's `TokenSource` function directly instead
4. Review the gist at https://gist.github.com/pietern/41d02acc2907416244789c7408fb232f for the working implementation
5. Apply similar pattern to get fresh OAuth tokens

## Related Comments

This also addresses:
- **libs/auth/token_loader.go:21** - Remove code defined purely for checking if `PersistentAuthOpts` aren't mutated
- **libs/auth/token_loader_test.go:53** - Simplify test setup (may become unnecessary)
- **libs/auth/token_loader_test.go:206** - Table test format (may become unnecessary)

## Files Affected

- `cmd/auth/token_*`
- `libs/auth/token_*`
- Any code that uses these token utilities
