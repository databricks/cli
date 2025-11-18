# Format Constants into Const Block

**Reviewer:** @pkosiec
**Date:** November 14, 2025
**Priority:** LOW

## Comment

**Location:** libs/logstream/streamer.go:25

Format constants in a const block (code suggestion provided)

## Action Items

1. Update `libs/logstream/streamer.go` (will be `libs/apps/logstream/streamer.go` after move)
2. Replace individual const declarations with a const block at line 25:

```go
const (
    handshakeErrorBodyLimit = 4 * 1024
    defaultUserAgent = "databricks-cli logstream"
    initialReconnectBackoff = 200 * time.Millisecond
    maxReconnectBackoff = 5 * time.Second
    closeCodeUnauthorized = 4401
    closeCodeForbidden = 4403
)
```

## Files Affected

- `libs/logstream/streamer.go:25` (will become `libs/apps/logstream/streamer.go` after package move)

## Dependencies

- Should be done after moving logstream package to libs/apps (see item #02)
