# COMPLETED: Move logstream Package to libs/apps

**Date Completed:** 2025-11-18
**Review Item:** #02 - Move logstream package to libs/apps

## Changes Made

### 1. Moved package directory
- Used `git mv` to preserve history
- `libs/logstream/streamer.go` → `libs/apps/logstream/streamer.go`
- `libs/logstream/streamer_test.go` → `libs/apps/logstream/streamer_test.go`

### 2. Updated imports
- `cmd/workspace/apps/logs.go`: Changed import from `github.com/databricks/cli/libs/logstream` to `github.com/databricks/cli/libs/apps/logstream`
- Imports are now alphabetically sorted

### 3. Removed empty directory
- Removed `libs/logstream/` directory after files were moved

## Testing

- ✅ Build successful: `make build`
- ✅ Logstream tests pass: `go test ./libs/apps/logstream/...`
- ✅ Apps tests pass: `go test ./cmd/workspace/apps/...`

## Files Modified

- `libs/logstream/streamer.go` → `libs/apps/logstream/streamer.go` (MOVED)
- `libs/logstream/streamer_test.go` → `libs/apps/logstream/streamer_test.go` (MOVED)
- `cmd/workspace/apps/logs.go` - Updated import path

## Reference

- Review comment from @pietern on Nov 17, 2025
- Makes it clearer that logstream is specific to the apps functionality
