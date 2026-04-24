---
description: Style Guide for Go
globs: **/*.go
paths:
  - "**/*.go"
---

# Style Guide for Go

## General guidance

Code you author should be consistent with the codebase and concise. The code should be self-documenting based on its function and variable names.

**RULE: Document functions with a doc comment that starts with the function name and ends with a period.**

```go
// SomeFunc does something.
func SomeFunc() {
	...
}
```

**RULE: Avoid redundant and verbose comments.** Only add a comment if it complements the code rather than repeating it.

**RULE: Focus on making implementations as small and elegant as possible.** Avoid unnecessary loops and allocations. If dropping or relaxing a requirement would simplify things, ask the user about the trade-off.

### Modern Go (1.24+) idioms

**RULE: Use `for i := range X` for integer iteration, not `for i := 0; i < X; i++`.**

**RULE: Use builtin `min()` and `max()` where possible.** They work on any type and any number of values.

**RULE: Do not capture the for-range variable.** Since Go 1.22 a new copy is created each iteration, so the capture workaround is no longer needed.

**RULE: Use empty struct types for context keys.**

GOOD:

```go
type myKeyType struct{}
```

BAD:

```go
type myKeyType int
```

**RULE: Define magic strings as named constants at the top of the file.**

**RULE: When integrating external tools or detecting environment variables, include source reference URLs as comments.** This lets future readers trace where the behavior came from.

### Lazy initialization

**RULE: Use `sync.OnceValue` or `sync.OnceValues` for cached computations, not `sync.Once` with package variables.** The `sync.OnceValue[T]` family (Go 1.21+) removes the boilerplate and makes the cached result a first-class return value.

GOOD:

```go
var loadConfig = sync.OnceValues(func() (*Config, error) {
	return parseConfigFile("config.yml")
})

func GetConfig() (*Config, error) {
	return loadConfig()
}
```

BAD:

```go
var (
	config     *Config
	configErr  error
	configOnce sync.Once
)

func GetConfig() (*Config, error) {
	configOnce.Do(func() {
		config, configErr = parseConfigFile("config.yml")
	})
	return config, configErr
}
```

Caveats that apply to both: results are cached forever (including errors), and a panic is rethrown on every subsequent call. Create a new instance if you need retry semantics.

### Constructors

**RULE: When a constructor has 4 or more parameters, group them into a struct.** Readability wins fast as the parameter list grows, and adding a field later does not ripple through every call site.

GOOD:

```go
type ServiceDeps struct {
	Client    HTTPClient
	Logger    Logger
	Config    Config
	Validator Validator
	Metrics   MetricsCollector
}

func NewService(deps ServiceDeps) *Service { ... }
```

BAD:

```go
func NewService(client HTTPClient, logger Logger, config Config, validator Validator, metrics MetricsCollector) *Service { ... }
```

### Configuration patterns

- Bundle config uses `dyn.Value` for dynamic typing
- Config loading supports includes, variable interpolation, and target overrides
- Schema generation is automated from Go struct tags

## Context

**RULE: Always pass `context.Context` as a function argument; never store it in a struct.** Storing context in a struct obscures the lifecycle and prevents callers from setting per-call deadlines, cancellation, and metadata. See https://go.dev/blog/context-and-structs.

**RULE: Do not use `context.Background()` outside of `main.go` files.**

**RULE: In tests, use `t.Context()` (or `b.Context()` for benchmarks).**

## Logging

**RULE: Use `github.com/databricks/cli/libs/log` for debug/info/warn/error logging.** The `ctx` variable must be passed in by the caller.

```go
import "github.com/databricks/cli/libs/log"

log.Infof(ctx, "...")
log.Debugf(ctx, "...")
log.Warnf(ctx, "...")
log.Errorf(ctx, "...")
```

**RULE: Use `cmdio.LogString` to print to stdout.**

```go
import "github.com/databricks/cli/libs/cmdio"

cmdio.LogString(ctx, "...")
```

**RULE: Always output file paths with forward slashes, even on Windows.** Use `filepath.ToSlash` so acceptance test output is stable between OSes.
