---
description: Mark unstable commands and fields in their help, not behind environment variables
globs:
  - "cmd/**/*.go"
  - "bundle/**/*.go"
paths:
  - "cmd/**/*.go"
  - "bundle/**/*.go"
---

**RULE: Do not gate a new command, flag or bundle field behind an environment variable.** An unstable feature still ships enabled. Mark it so that the Stability Policy in `README.md` applies: start its `Short` with `*Beta*` or `*Private Preview*` (see `cmd/sandbox/sandbox.go`), or register it under `databricks experimental`. When the service side is in preview, a workspace without it already rejects the call; return that error to the user.

BAD:
```go
if env.Get(ctx, "DATABRICKS_CLI_FOO") != "true" {
	return errors.New("foo is experimental, set DATABRICKS_CLI_FOO=true to enable it")
}
```

GOOD:
```go
Short: "*Beta* Run the tests of a pipeline",
```
