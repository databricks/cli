# Databricks Apps Logs Command

## Overview
`databricks apps logs NAME` connects to an app's `/logz/stream` WebSocket endpoint, reusing the persistent OAuth cache produced by `databricks auth login`. It can:

- Fetch the last _N_ log lines (`--tail-lines`, defaults to 200).
- Stream logs continuously (`--follow`) with optional `--timeout`.
- Filter server-side via `--search` (same semantics as the UI) and client-side via `--source APP|SYSTEM`.
- Mirror output to a local file via `--output-file` (created with `0600` permissions to avoid leaking sensitive data).

## Implementation Notes
- CLI wiring: `cmd/workspace/apps/logs.go`.
- WebSocket/tailing logic: `libs/logstream/streamer.go`.
- Shared token acquisition: `libs/auth/token_loader.go`.

## Reusing the logstream Helper
Other commands can stream logs without reimplementing buffering or retries by importing `github.com/databricks/cli/libs/logstream` and calling:

```go
err := logstream.Run(ctx, logstream.Config{
	Dialer:        yourDialer,
	URL:           wsURL,
	Token:         token,
	TokenProvider: refreshToken,
	Tail:          tailLines,
	Follow:        follow,
	Search:        searchTerm,
	Sources:       sourceFilter,
	Writer:        output,
})
```

Only `Writer` is mandatory; omit other fields to use the defaults.

## Testing

### Unit
Run `go test ./cmd/workspace/apps` to cover:
- Tail buffering behavior across quiet periods.
- WebSocket reconnect/backoff.
- Search term dispatch.
- Source filtering.
- File output and abnormal-close handling.

### Behavior / manual
```
databricks apps logs <name> \
  --profile <profile> \
  --tail-lines 5 \
  --follow \
  --search ERROR \
  --source app \
  --output-file /tmp/app.log
```

### Acceptance
`TestAccept/bundle/apps` continues to run (see `acceptance/bundle/apps`) to guard bundle workflows that create/manage apps. Full end-to-end streaming against `/logz/stream` isn't exercised in acceptance because it requires a long-lived Databricks app and WebSocket endpoint, but the unit suite above covers the log-specific behavior deterministically.
