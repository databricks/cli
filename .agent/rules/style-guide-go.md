---
description: Style Guide for Go
globs: "**/*.go"
paths:
  - "**/*.go"
---

## General guidance

Please make sure code that you author is consistent with the codebase and concise.

The code should be self-documenting based on the code and function names.

Functions should be documented with a doc comment as follows:

```go
// SomeFunc does something.
func SomeFunc() {
	...
}
```

Note how the comment starts with the name of the function and is followed by a period.

Avoid redundant and verbose comments. Use terse comments and only add comments if it complements, not repeats the code.

Focus on making implementation as small and elegant as possible. Avoid unnecessary loops and allocations. If you see an opportunity of making things simpler by dropping or relaxing some requirements, ask user about the trade-off.

Use modern idiomatic Golang features (version 1.24+). Specifically:
 - Use for-range for integer iteration where possible. Instead of for i:=0; i < X; i++ {} you must write for i := range X{}.
 - Use builtin min() and max() where possible (works on any type and any number of values).
 - Do not capture the for-range variable, since go 1.22 a new copy of the variable is created for each loop iteration.
 - Use empty struct types for context keys: `type myKeyType struct{}` (not `int`).
 - Define magic strings as named constants at the top of the file.
 - When integrating external tools or detecting environment variables, include source reference URLs as comments so they can be traced later.

### Configuration Patterns

- Bundle config uses `dyn.Value` for dynamic typing
- Config loading supports includes, variable interpolation, and target overrides
- Schema generation is automated from Go struct tags

## Context

Always pass `context.Context` as a function argument; never store it in a struct. Storing context in a struct obscures the lifecycle and prevents callers from setting per-call deadlines, cancellation, and metadata (see https://go.dev/blog/context-and-structs). Do not use `context.Background()` outside of `main.go` files. In tests, use `t.Context()` (or `b.Context()` for benchmarks).

## Logging

Use the following for logging:

```go
import "github.com/databricks/cli/libs/log"

log.Infof(ctx, "...")
log.Debugf(ctx, "...")
log.Warnf(ctx, "...")
log.Errorf(ctx, "...")
```

Note that the 'ctx' variable here is something that should be passed in as
an argument by the caller.

Use cmdio.LogString to print to stdout:

```go
import "github.com/databricks/cli/libs/cmdio"

cmdio.LogString(ctx, "...")
```

Always output file path with forward slashes, even on Windows, so that acceptance test output is stable between OSes. Use filepath.ToSlash for this.
