---
name: Logstream Helper
description: How to reuse libs/logstream to stream Databricks logs.
---
# Logstream Helper

Use this skill whenever you need a Databricks CLI command (or tool) to stream logs over WebSockets without rewriting buffering/follow logic.

## Prerequisites
- Go 1.24+
- Access to a Dialer (`*websocket.Dialer` or compatible) and an authenticated token (or provider) for the target endpoint.

## Steps
1. **Collect connection info**
   - Resolve the app/pipeline URL and call `buildLogsURL`-style helper to convert `https://...` into `wss://.../logz/stream`.
   - Normalize the origin if the server enforces CORS.

2. **Prepare auth**
   - Reuse `libs/auth.AcquireToken` to grab an OAuth token.
   - Provide a `TokenProvider` closure if the stream should refresh tokens (recommended for `--follow`).

3. **Create the dialer**
   - Call `newLogStreamDialer(cfg)` to clone the workspace HTTP transport so proxies/TLS settings are honored.

4. **Configure and run the streamer**
   ```go
   err := logstream.Run(ctx, logstream.Config{
       Dialer:        dialer,
       URL:           wsURL,
       Origin:        origin,
       Token:         token.AccessToken,
       TokenProvider: tokenProvider,
       Search:        searchTerm,
       Sources:       sourceFilters, // map[string]struct{}{"APP": {}}
       Tail:          tailLines,
       Follow:        follow,
       Prefetch:      2 * time.Second,
       Writer:        output, // stdout/file io.Writer
       UserAgent:     "databricks-cli <your command>",
   })
   ```
   - Only `Writer` is required; omit other fields to use defaults.

5. **Handle output destinations**
   - Wrap `cmd.OutOrStdout()` with `io.MultiWriter` to mirror logs into a file; create files with `0600` permissions for sensitive data.

6. **Testing**
   - Use `libs/logstream/streamer_test.go` as a template: spin up an `httptest.Server` + `websocket.Upgrader` to script frames, close codes, and timeouts.

## Tips
- Avoid re-exposing the old `--prefetch` flag; the default internal window already covers tail buffering.
- Group related flags via `cmdgroup.NewFlagGroup` for structured help output.
