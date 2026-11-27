---
description: Style Guide for Go
globs: **/*.go
paths:
  - "**/*.go"
---

# Style Guide for Go

Code you author should be consistent with the codebase and concise. Many conventions (modern Go idioms, `os.Getenv`, `sync.Once`, `context.Background()`, stdlib `log`, error wrapping) are enforced by `./task lint`; run it rather than relying on memory.

**RULE: Focus on making implementations as small and elegant as possible.** If dropping or relaxing a requirement would simplify things, ask the user about the trade-off.

**RULE: Do not declare inline function closures just to name a repeated operation.** Extract a regular named function instead.

**RULE: Collapse `if err := f(); err != nil { return err }; return nil` to `return f()`.**

**RULE: When a slice built from map iteration is observable (tests, logs, update masks, wire format), sort it.** Unsorted is fine when nothing downstream observes the order.

**RULE: When mutating an API response by round-tripping JSON, decode with `json.Decoder` + `UseNumber()`.** `map[string]any` round-trips corrupt int64 values above 2^53 and alphabetize keys.

**RULE: `encoding/csv` with `UseCRLF = true` also rewrites newlines embedded in quoted fields to `\r\n`.**

**RULE: Do not compile regexps, call `reflect`, or do other non-trivial work in package-level `var` initializers or `init()`.** They run on every CLI invocation. Wrap the value in `sync.OnceValue` and make sure a test calls it, so a bad pattern panics in tests rather than at runtime.

**RULE: Pass `context.Context` as a function argument; never store it in a struct.**

**RULE: Use `cmdio.LogString(ctx, ...)` to print to stdout and `libs/log` (`log.Debugf(ctx, ...)` etc.) for logging.** Warn only for things the user should know about and might act on; diagnostics a user can't act on go to debug.

**RULE: Always output file paths with forward slashes (`filepath.ToSlash`), even on Windows**, so acceptance test output is stable across OSes.
