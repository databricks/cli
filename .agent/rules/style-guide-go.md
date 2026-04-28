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

**RULE: Define magic strings as named constants at the top of the file.** When a value is used in more than one place, define a package-level constant even if the literal is short. If a related value already has a constant in the same package, follow the existing pattern instead of introducing a parallel literal.

**RULE: When integrating external tools or detecting environment variables, include source reference URLs as comments.** This lets future readers trace where the behavior came from.

### Control flow

**RULE: When several branches are alternatives for the same decision, prefer `switch` over `if / else if`.** Same-decision means: you're dispatching on one value or one boolean question and each branch is a different answer to it. Ordered precedence chains (try this, else this, else this) are a different shape; leave those as early-return `if`s.

GOOD (alternatives for one decision):

```go
switch mode {
case modeText:
	renderText(out, result)
case modeJSON:
	renderJSON(out, result)
default:
	return fmt.Errorf("unknown mode %q", mode)
}
```

Also fine (ordered precedence, early-return style):

```go
// see libs/auth/storage/mode.go#ResolveStorageMode
if override != "" {
	return override, nil
}
if envValue != "" {
	return parseMode(envValue)
}
return loadFromFile()
```

**RULE: Collapse `if err != nil { return err }; return nil` to just `return err`.** The pattern is never doing anything useful in the intermediate step.

GOOD:

```go
return someCall()
```

BAD:

```go
if err := someCall(); err != nil {
	return err
}
return nil
```

### Determinism

**RULE: When you build a slice by iterating over a map and its order affects tests, logs, update masks, or the wire format, sort it before returning.** Go maps have randomized iteration order, so the same input produces different outputs across runs. Reviewers catch these when they produce flaky test output or noisy diffs in update masks. If the slice is purely internal and nothing downstream observes its order, sorting is unnecessary — some accepted code in `bundle/direct/dresources/` and `bundle/config/mutator/resourcemutator/` returns unsorted slices precisely because order doesn't matter there.

GOOD:

```go
fieldPaths := make([]string, 0, len(changes))
for p := range changes {
	fieldPaths = append(fieldPaths, p)
}
slices.Sort(fieldPaths)
return fieldPaths
```

BAD:

```go
fieldPaths := make([]string, 0, len(changes))
for p := range changes {
	fieldPaths = append(fieldPaths, p)
}
return fieldPaths
```

### Environment variables

**RULE: In library and product code, use `github.com/databricks/cli/libs/env` for reading environment variables, not `os.Getenv`.** `env.Get(ctx, name)` and `env.Lookup(ctx, name)` can be overridden per-context in tests, so you don't have to mutate process-wide state to exercise a code path. `os.Getenv` is still fine in `main`, tests, and acceptance/integration harnesses where no `ctx` is available and overrides aren't needed.

GOOD:

```go
import "github.com/databricks/cli/libs/env"

token := env.Get(ctx, "DATABRICKS_TOKEN")
if path, ok := env.Lookup(ctx, "DATABRICKS_CLI_PATH"); ok {
	// use path
}
```

BAD:

```go
token := os.Getenv("DATABRICKS_TOKEN")
```

### Lazy initialization

**RULE: Use `sync.OnceValue`, `sync.OnceValues`, or `sync.OnceFunc` for one-time initialization, not `sync.Once` with package variables.** The `sync.OnceValue[T]` family (Go 1.21+) removes the boilerplate and makes the cached result a first-class return value. Use `sync.OnceFunc` when the initialization has side effects and no return value (the CLI already uses it in places like `libs/cmdio/spinner.go`).

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

When a constructor's parameter list is long enough that callers forget the order or misread positional arguments, consider grouping dependencies into a struct. This is a judgment call. The CLI has many ordinary constructors with 4+ parameters, and that's fine when the arguments are obvious at the call site. The signal for switching is readability, not parameter count.

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

**RULE: Pick log levels deliberately.** Warn for things the user should know about and might act on. Debug for diagnostic signal a developer wants but a user shouldn't see by default. Error for actual failures that also surface as a returned error. A message that's warn-by-default but isn't user-actionable belongs at debug.

GOOD:

```go
if err := w.Config.Authenticate(); err != nil {
	// user can check their profile; worth warning
	log.Warnf(ctx, "could not authenticate: %v", err)
}

if err := cleanupExpiredCacheEntries(ctx); err != nil {
	// internal-only, user can't act on this
	log.Debugf(ctx, "cache cleanup failed: %v", err)
}
```

BAD:

```go
log.Warnf(ctx, "cache cleanup failed: %v", err) // noisy, not actionable
```
