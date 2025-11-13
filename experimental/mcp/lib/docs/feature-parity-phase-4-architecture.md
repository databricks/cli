# Phase 4: Architecture Improvements

## Objective

Refactor the Go codebase architecture to match beneficial patterns from the Rust implementation, improving type safety, maintainability, and extensibility.

## Priority

**🔵 LOW** - Code quality improvements that don't add new features but improve maintainability.

## Context

The Rust implementation uses several patterns that provide stronger type safety and clearer abstractions:
- State machine with explicit enums
- Strategy pattern for validation
- Provider registry for auto-discovery
- Standardized error handling with error codes

These patterns can improve the Go codebase without changing external behavior.

## Implementation Steps

### Improvement 1: State Machine with Enums

#### Current State (Go)
```go
type ProjectState struct {
    State       string     `json:"state"` // "Scaffolded", "Validated", "Deployed"
    ValidatedAt *time.Time `json:"validated_at,omitempty"`
    Checksum    string     `json:"checksum,omitempty"`
    DeployedAt  *time.Time `json:"deployed_at,omitempty"`
}
```

#### Proposed Improvement

**pkg/providers/io/state.go:**

```go
package io

import (
    "fmt"
    "time"
)

// ProjectStateType represents the lifecycle state
type ProjectStateType int

const (
    StateScaffolded ProjectStateType = iota
    StateValidated
    StateDeployed
)

// String returns the string representation
func (s ProjectStateType) String() string {
    switch s {
        case StateScaffolded:
            return "Scaffolded"
        case StateValidated:
            return "Validated"
        case StateDeployed:
            return "Deployed"
        default:
            return "Unknown"
    }
}

// ParseProjectStateType parses a string into a state type
func ParseProjectStateType(s string) (ProjectStateType, error) {
    switch s {
        case "Scaffolded":
            return StateScaffolded, nil
        case "Validated":
            return StateValidated, nil
        case "Deployed":
            return StateDeployed, nil
        default:
            return 0, fmt.Errorf("invalid state: %s", s)
    }
}

// ProjectState represents the project lifecycle state
type ProjectState struct {
    State       ProjectStateType `json:"state"`
    ValidatedAt *time.Time       `json:"validated_at,omitempty"`
    Checksum    string           `json:"checksum,omitempty"`
    DeployedAt  *time.Time       `json:"deployed_at,omitempty"`
}

// CanTransitionTo checks if a state transition is valid
func (s ProjectState) CanTransitionTo(next ProjectStateType) bool {
    switch s.State {
    case StateScaffolded:
        return next == StateValidated
    case StateValidated:
        return next == StateDeployed || next == StateValidated // Re-validation allowed
    case StateDeployed:
        return next == StateValidated // Re-validation allowed, then re-deploy
    default:
        return false
    }
}

// TransitionTo attempts to transition to a new state
func (s *ProjectState) TransitionTo(next ProjectStateType) error {
    if !s.CanTransitionTo(next) {
        return fmt.Errorf("invalid state transition: %s -> %s", s.State, next)
    }

    s.State = next
    return nil
}

// MarshalJSON implements custom JSON marshaling
func (s ProjectState) MarshalJSON() ([]byte, error) {
    type Alias ProjectState
    return json.Marshal(&struct {
        State string `json:"state"`
        *Alias
    }{
        State: s.State.String(),
        Alias: (*Alias)(&s),
    })
}

// UnmarshalJSON implements custom JSON unmarshaling
func (s *ProjectState) UnmarshalJSON(data []byte) error {
    type Alias ProjectState
    aux := &struct {
        State string `json:"state"`
        *Alias
    }{
        Alias: (*Alias)(s),
    }

    if err := json.Unmarshal(data, &aux); err != nil {
        return err
    }

    state, err := ParseProjectStateType(aux.State)
    if err != nil {
        return err
    }

    s.State = state
    return nil
}
```

#### Benefits
- Compile-time validation of state values
- Explicit transition rules
- Prevents invalid states at type level
- Clear state machine semantics

#### Migration
- Update all code using `state.State` string comparisons
- Use `state.State == StateValidated` instead of `state.State == "Validated"`
- Add validation on state transitions
- Update tests

### Improvement 2: Validation Strategy Pattern

#### Current State
Validation logic mixed with if/else statements in `validate.go`

#### Proposed Improvement

**pkg/providers/io/validation/strategy.go:**

```go
package validation

import (
    "context"
    "go-mcp/pkg/sandbox"
)

// Strategy defines the validation interface
type Strategy interface {
    Name() string
    Validate(ctx context.Context, sb sandbox.Sandbox) (*Result, error)
}

// Result contains validation results
type Result struct {
    Success     bool     `json:"success"`
    Message     string   `json:"message"`
    ProgressLog []string `json:"progress_log"`
    Duration    int64    `json:"duration_ms"`
}

// StrategyFactory creates validation strategies
type StrategyFactory func(config Config) Strategy

var strategies = make(map[string]StrategyFactory)

// Register registers a validation strategy
func Register(name string, factory StrategyFactory) {
    strategies[name] = factory
}

// Get retrieves a validation strategy by name
func Get(name string, config Config) (Strategy, error) {
    factory, ok := strategies[name]
    if !ok {
        return nil, fmt.Errorf("unknown validation strategy: %s", name)
    }
    return factory(config), nil
}

// Config holds validation configuration
type Config struct {
    DockerImage string
    Timeout     int
}
```

**pkg/providers/io/validation/trpc.go:**

```go
package validation

import (
    "context"
    "fmt"
    "time"

    "go-mcp/pkg/sandbox"
)

// TRPCStrategy validates tRPC + React projects
type TRPCStrategy struct {
    config Config
}

func init() {
    Register("trpc", func(cfg Config) Strategy {
        return &TRPCStrategy{config: cfg}
    })
}

func (s *TRPCStrategy) Name() string {
    return "TRPC Validation"
}

func (s *TRPCStrategy) Validate(ctx context.Context, sb sandbox.Sandbox) (*Result, error) {
    result := &Result{
        Success:     true,
        ProgressLog: []string{},
    }

    start := time.Now()

    // Step 1: Build
    result.ProgressLog = append(result.ProgressLog, "🔄 Starting validation: build + type check + tests")
    result.ProgressLog = append(result.ProgressLog, "⏳ Step 1/3: Running build...")

    stepStart := time.Now()
    execResult, err := sb.Exec(ctx, "npm run build")
    if err != nil || execResult.ExitCode != 0 {
        result.Success = false
        result.Message = fmt.Sprintf("Build failed: %s", execResult.Stderr)
        result.ProgressLog = append(result.ProgressLog, "❌ Build failed")
        return result, nil
    }
    result.ProgressLog = append(result.ProgressLog, fmt.Sprintf("✅ Build passed (%.1fs)", time.Since(stepStart).Seconds()))

    // Step 2: Type check
    result.ProgressLog = append(result.ProgressLog, "⏳ Step 2/3: Running type check...")
    stepStart = time.Now()
    execResult, err = sb.Exec(ctx, "cd client && npx tsc --noEmit")
    if err != nil || execResult.ExitCode != 0 {
        result.Success = false
        result.Message = fmt.Sprintf("Type check failed: %s", execResult.Stderr)
        result.ProgressLog = append(result.ProgressLog, "❌ Type check failed")
        return result, nil
    }
    result.ProgressLog = append(result.ProgressLog, fmt.Sprintf("✅ Type check passed (%.1fs)", time.Since(stepStart).Seconds()))

    // Step 3: Tests
    result.ProgressLog = append(result.ProgressLog, "⏳ Step 3/3: Running tests...")
    stepStart = time.Now()
    execResult, err = sb.Exec(ctx, "npm test")
    if err != nil || execResult.ExitCode != 0 {
        result.Success = false
        result.Message = fmt.Sprintf("Tests failed: %s", execResult.Stderr)
        result.ProgressLog = append(result.ProgressLog, "❌ Tests failed")
        return result, nil
    }
    result.ProgressLog = append(result.ProgressLog, fmt.Sprintf("✅ Tests passed (%.1fs)", time.Since(stepStart).Seconds()))

    result.Duration = time.Since(start).Milliseconds()
    result.Message = fmt.Sprintf("✅ All checks passed! Total: %.1fs", time.Since(start).Seconds())
    result.ProgressLog = append(result.ProgressLog, result.Message)

    return result, nil
}
```

**pkg/providers/io/validation/custom.go:**

```go
package validation

import (
    "context"
    "fmt"
    "time"

    "go-mcp/pkg/sandbox"
)

// CustomStrategy runs a user-defined validation command
type CustomStrategy struct {
    config  Config
    command string
}

func init() {
    Register("custom", func(cfg Config) Strategy {
        return &CustomStrategy{
            config:  cfg,
            command: cfg.Command, // Needs to be added to Config
        }
    })
}

func (s *CustomStrategy) Name() string {
    return "Custom Validation"
}

func (s *CustomStrategy) Validate(ctx context.Context, sb sandbox.Sandbox) (*Result, error) {
    result := &Result{
        Success:     true,
        ProgressLog: []string{},
    }

    start := time.Now()
    result.ProgressLog = append(result.ProgressLog, fmt.Sprintf("🔄 Running custom command: %s", s.command))

    execResult, err := sb.Exec(ctx, s.command)
    if err != nil || execResult.ExitCode != 0 {
        result.Success = false
        result.Message = fmt.Sprintf("Command failed: %s", execResult.Stderr)
        result.ProgressLog = append(result.ProgressLog, "❌ Command failed")
        return result, nil
    }

    result.Duration = time.Since(start).Milliseconds()
    result.Message = fmt.Sprintf("✅ Command succeeded (%.1fs)", time.Since(start).Seconds())
    result.ProgressLog = append(result.ProgressLog, result.Message)

    return result, nil
}
```

#### Usage

**pkg/providers/io/validate.go:**

```go
func (p *Provider) validateDataApp(ctx context.Context, args ValidateDataAppInput) (*ValidateDataAppResult, error) {
    // ... sandbox setup ...

    // Get validation strategy
    strategyName := "trpc"
    if p.config.IOConfig.Validation != nil && p.config.IOConfig.Validation.Command != "" {
        strategyName = "custom"
    }

    strategy, err := validation.Get(strategyName, validation.Config{
        DockerImage: dockerImage,
        Timeout:     timeout,
    })
    if err != nil {
        return nil, err
    }

    // Run validation
    result, err := strategy.Validate(ctx, sandbox)
    if err != nil {
        return nil, err
    }

    // ... rest of validation logic ...
}
```

#### Benefits
- Easy to add new validation strategies
- Clear separation of concerns
- Testable in isolation
- Pluggable architecture

### Improvement 3: Provider Registry

#### Current State
Manual provider initialization in `cli.go`

#### Proposed Improvement

**pkg/providers/registry.go:**

```go
package providers

import (
    "fmt"
    "sync"

    "go-mcp/pkg/config"
    "go-mcp/pkg/logging"
    "go-mcp/pkg/mcp"
    "go-mcp/pkg/session"
)

// Provider interface for all providers
type Provider interface {
    Name() string
    RegisterTools(server *mcp.Server) error
}

// Factory creates a provider instance
type Factory func(cfg *config.Config, sess *session.Session, logger *logging.Logger) (Provider, error)

var (
    registry = make(map[string]Factory)
    mu       sync.RWMutex
)

// Register registers a provider factory
func Register(name string, factory Factory) {
    mu.Lock()
    defer mu.Unlock()

    registry[name] = factory
}

// Create creates a provider by name
func Create(name string, cfg *config.Config, sess *session.Session, logger *logging.Logger) (Provider, error) {
    mu.RLock()
    factory, ok := registry[name]
    mu.RUnlock()

    if !ok {
        return nil, fmt.Errorf("unknown provider: %s", name)
    }

    return factory(cfg, sess, logger)
}

// CreateAll creates all registered providers
func CreateAll(cfg *config.Config, sess *session.Session, logger *logging.Logger) ([]Provider, error) {
    mu.RLock()
    names := make([]string, 0, len(registry))
    for name := range registry {
        names = append(names, name)
    }
    mu.RUnlock()

    providers := make([]Provider, 0, len(names))
    for _, name := range names {
        provider, err := Create(name, cfg, sess, logger)
        if err != nil {
            return nil, fmt.Errorf("failed to create provider %s: %w", name, err)
        }
        providers = append(providers, provider)
    }

    return providers, nil
}
```

#### Provider Registration

Each provider registers itself via `init()`:

**pkg/providers/databricks/provider.go:**

```go
func init() {
    providers.Register("databricks", func(cfg *config.Config, sess *session.Session, logger *logging.Logger) (providers.Provider, error) {
        return NewProvider(cfg, sess, logger)
    })
}
```

#### Usage

**cmd/go-mcp/cli.go:**

```go
func startServer(cfg *config.Config, sess *session.Session, logger *logging.Logger) error {
    server := mcp.NewServer(cfg, sess, logger)

    // Create and register all providers
    providers, err := providers.CreateAll(cfg, sess, logger)
    if err != nil {
        return err
    }

    for _, provider := range providers {
        if err := provider.RegisterTools(server); err != nil {
            return fmt.Errorf("failed to register %s tools: %w", provider.Name(), err)
        }
    }

    return server.Run(ctx)
}
```

#### Benefits
- Auto-discovery of providers
- Reduces boilerplate in main
- Easy to add/remove providers
- Clear provider lifecycle

### Improvement 4: Error Handling Improvements

#### Current State
Error handling is functional but could be more consistent

#### Proposed Improvement

**pkg/errors/catalog.go:**

```go
package errors

// ErrorCode represents a specific error condition
type ErrorCode string

const (
    // Configuration errors
    ErrCodeConfigInvalid     ErrorCode = "CONFIG_INVALID"
    ErrCodeConfigMissing     ErrorCode = "CONFIG_MISSING"

    // Sandbox errors
    ErrCodeSandboxFailed     ErrorCode = "SANDBOX_FAILED"
    ErrCodeSandboxNotFound   ErrorCode = "SANDBOX_NOT_FOUND"

    // Databricks errors
    ErrCodeDatabricksAuth    ErrorCode = "DATABRICKS_AUTH"
    ErrCodeDatabricksQuery   ErrorCode = "DATABRICKS_QUERY"

    // Validation errors
    ErrCodeValidationFailed  ErrorCode = "VALIDATION_FAILED"
    ErrCodeValidationState   ErrorCode = "VALIDATION_STATE"

    // Deployment errors
    ErrCodeDeploymentFailed  ErrorCode = "DEPLOYMENT_FAILED"
    ErrCodeDeploymentState   ErrorCode = "DEPLOYMENT_STATE"
)

// CodedError wraps an error with a code
type CodedError struct {
    Code    ErrorCode
    Message string
    Cause   error
}

func (e *CodedError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *CodedError) Unwrap() error {
    return e.Cause
}

// New creates a new coded error
func New(code ErrorCode, message string) error {
    return &CodedError{
        Code:    code,
        Message: message,
    }
}

// Wrap wraps an error with a code and message
func Wrap(code ErrorCode, message string, err error) error {
    return &CodedError{
        Code:    code,
        Message: message,
        Cause:   err,
    }
}

// WithSuggestion adds a helpful suggestion to an error
func WithSuggestion(err error, suggestion string) error {
    if ce, ok := err.(*CodedError); ok {
        ce.Message = fmt.Sprintf("%s\nSuggestion: %s", ce.Message, suggestion)
        return ce
    }
    return err
}
```

#### Usage

```go
// Instead of:
return fmt.Errorf("validation failed: %w", err)

// Use:
return errors.Wrap(errors.ErrCodeValidationFailed, "validation failed", err)

// With suggestion:
err := errors.Wrap(errors.ErrCodeDatabricksAuth, "authentication failed", err)
return errors.WithSuggestion(err, "Check that DATABRICKS_TOKEN is set and valid")
```

#### Benefits
- Consistent error format
- Machine-readable error codes
- Helpful suggestions for users
- Better error tracking and analytics

### Improvement 5: Performance Optimizations

#### Areas to Profile and Optimize

1. **File Operations**: Batch reads/writes where possible
2. **JSON Parsing**: Use `json.Decoder` for streaming
3. **Allocations**: Reduce allocations in hot paths
4. **Goroutines**: Use sync.Pool for worker pools
5. **Logging**: Add sampling for high-frequency events

#### Tools

```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=.

# Memory profiling
go test -memprofile=mem.prof -bench=.

# Analyze
go tool pprof cpu.prof

# Race detection
go test -race ./...
```

## Testing Strategy

### For Each Improvement
1. Write tests for new abstractions
2. Ensure backward compatibility
3. Update integration tests
4. Run performance benchmarks

### Migration Testing
- Test with existing `.edda_state` files
- Verify state transitions work correctly
- Ensure no breaking changes for users

## Success Criteria

- [ ] State machine with enums implemented
  - [ ] Type-safe states
  - [ ] Transition validation
  - [ ] Backward compatible JSON serialization
- [ ] Validation strategy pattern implemented
  - [ ] TRPC strategy working
  - [ ] Custom strategy working
  - [ ] Easy to add new strategies
- [ ] Provider registry implemented
  - [ ] All providers auto-register
  - [ ] Reduced boilerplate in main
  - [ ] Easy to add/remove providers
- [ ] Error handling improved
  - [ ] Error codes catalog
  - [ ] Suggestions for common errors
  - [ ] Consistent error format
- [ ] Performance optimized
  - [ ] Hot paths identified and optimized
  - [ ] Benchmarks show improvements
  - [ ] No performance regressions
- [ ] All tests passing
- [ ] Documentation updated

## Timeline

- **Days 1-2**: State machine refactoring
- **Days 3-4**: Validation strategy pattern
- **Days 5-6**: Provider registry
- **Days 7-8**: Error handling improvements
- **Days 9-10**: Performance optimizations

## Documentation Updates

Update CLAUDE.md with new patterns:

```markdown
## Architecture Patterns

### State Machine
Project lifecycle managed by explicit state machine with enum types and transition validation.

### Strategy Pattern
Validation strategies are pluggable and registered via factory pattern.

### Provider Registry
Providers auto-register via init() functions for automatic discovery.

### Error Handling
All errors use coded errors with suggestions for common issues.
```

## Migration Path

1. **Phase 1**: Add new abstractions alongside existing code
2. **Phase 2**: Migrate code incrementally, one component at a time
3. **Phase 3**: Remove old code once migration complete
4. **Phase 4**: Update documentation and examples

## Risk Mitigation

- Maintain backward compatibility for `.edda_state` file format
- Use feature flags for new patterns during migration
- Comprehensive testing at each step
- Gradual rollout to avoid disruption

## Completion

After completing all architecture improvements, the Go version will have:
- Stronger type safety
- Better error messages
- Clearer abstractions
- Easier maintenance
- Full feature parity with Rust

## Summary

This phase focuses on internal code quality improvements that make the codebase more maintainable, extensible, and robust without changing external behavior. These improvements provide a solid foundation for future enhancements.
