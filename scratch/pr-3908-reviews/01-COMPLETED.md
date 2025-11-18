# COMPLETED: Refactor Token Handling

**Date Completed:** 2025-11-18
**Review Item:** #01 - Refactor token handling to use SDK's TokenSource

## Changes Made

### 1. Updated cmd/workspace/apps/logs.go
- Removed import of `github.com/databricks/cli/libs/auth`
- Removed `tokenAcquireTimeout` constant (no longer needed)
- Replaced custom `auth.AcquireToken()` calls with SDK's `cfg.GetTokenSource()`
- Simplified token provider implementation to use `tokenSource.Token(ctx)`

**Before:**
```go
authArgs := &auth.AuthArguments{Host: cfg.Host, AccountID: cfg.AccountID}
tokenRequest := auth.AcquireTokenRequest{
    AuthArguments: authArgs,
    ProfileName:   cfg.Profile,
    Timeout:       tokenAcquireTimeout,
}
token, err := auth.AcquireToken(ctx, tokenRequest)
```

**After:**
```go
tokenSource := cfg.GetTokenSource()
if tokenSource == nil {
    return errors.New("configuration does not support OAuth tokens")
}

initialToken, err := tokenSource.Token(ctx)
```

### 2. Reverted cmd/auth/token.go and cmd/auth/token_test.go
- Used `git checkout origin/main` to restore original implementation
- These files were modified to use the new token_loader abstraction
- Since we don't need that abstraction, reverted to simpler original code

### 3. Removed libs/auth/token_loader.go and libs/auth/token_loader_test.go
- Used `git rm` to remove newly added files
- These files were created to abstract token loading logic
- No longer needed since logs.go uses SDK's TokenSource directly

## Side Effects

This change also resolves these review items:
- **#03** - Convert tests to table test format - NO LONGER APPLICABLE (files removed)
- **#04** - Remove PersistentAuthOpts mutation check - NO LONGER APPLICABLE (files removed)
- **#05** - Simplify test setup - NO LONGER APPLICABLE (files removed)

## Testing

- ✅ Build successful: `make build`
- ✅ Auth tests pass: `go test ./cmd/auth/...`
- ✅ Apps tests pass: `go test ./cmd/workspace/apps/...`

## Files Modified

- `cmd/workspace/apps/logs.go` - Refactored to use TokenSource
- `cmd/auth/token.go` - Reverted to main
- `cmd/auth/token_test.go` - Reverted to main
- `libs/auth/token_loader.go` - DELETED
- `libs/auth/token_loader_test.go` - DELETED

## Reference

- Review comment from @pietern on Nov 17, 2025
- Working patch: https://gist.github.com/pietern/41d02acc2907416244789c7408fb232f
