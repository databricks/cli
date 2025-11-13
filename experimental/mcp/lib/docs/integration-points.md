# Integration Points Documentation

**Generated**: 2025-11-12
**For**: parity-39 (Phase 1: Repository Setup & Analysis)

## Overview

This document details how go-mcp components will integrate with Databricks CLI systems, including logging, configuration, context management, command structure, testing, and build systems.

## Integration Matrix

| Component | go-mcp Current | Databricks CLI | Integration Strategy | Difficulty |
|-----------|---------------|----------------|---------------------|------------|
| **CLI Framework** | Cobra | Cobra | Direct compatibility | ⭐ Easy |
| **Logging** | Custom `pkg/logging` | `libs/log` (slog) | Replace with CLI logging | ⭐⭐ Medium |
| **Configuration** | Viper-based | `cmdctx` + flags | Adapt to CLI patterns | ⭐⭐⭐ Complex |
| **Session/Context** | Custom `pkg/session` | `cmdctx` package | Merge concepts | ⭐⭐ Medium |
| **Error Handling** | Custom `pkg/errors` | Standard Go errors | Adapt or merge with `libs/errs` | ⭐ Easy |
| **Command Structure** | Single command | Command groups | Fit into `apps` group | ⭐⭐ Medium |
| **Testing** | Standard `go test` | `gotestsum` + acceptance | Adapt test patterns | ⭐⭐ Medium |
| **Build System** | Custom Makefile | CLI Makefile | Integrate targets | ⭐ Easy |

## 1. CLI Framework Integration (Cobra)

### Current State (go-mcp)

```go
// cmd/go-mcp/main.go
func main() {
    rootCmd := &cobra.Command{
        Use:   "go-mcp",
        Short: "MCP server for Databricks",
    }
    rootCmd.AddCommand(startCmd)
    rootCmd.AddCommand(checkCmd)
    rootCmd.Execute()
}
```

### Target State (CLI)

```go
// cmd/apps/apps.go
package apps

func NewAppsCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "apps",
        Short: "Databricks apps development tools",
        GroupID: "development",
    }
    cmd.AddCommand(mcp.NewMcpCmd())
    return cmd
}

// cmd/apps/mcp/mcp.go
package mcp

func NewMcpCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "mcp",
        Short: "Model Context Protocol server for AI agents",
    }
    cmd.AddCommand(newStartCmd())
    cmd.AddCommand(newCheckCmd())
    cmd.AddCommand(newConfigCmd())
    return cmd
}
```

### Integration Steps

1. Create `cmd/apps/apps.go` with root apps command
2. Create `cmd/apps/mcp/mcp.go` with MCP subcommand
3. Register in `cmd/cmd.go`:
   ```go
   import "github.com/databricks/cli/cmd/apps"
   cmd.AddCommand(apps.NewAppsCmd())
   ```
4. Split go-mcp's `cli.go` into focused command files

**Difficulty**: ⭐ Easy (direct Cobra compatibility)

## 2. Logging Integration

### Current State (go-mcp)

```go
// pkg/logging/logger.go
import "log/slog"

type Logger struct {
    logger    *slog.Logger
    sessionID string
    logFile   *os.File
}

func NewLogger(sessionID string) *Logger {
    logPath := filepath.Join(homeDir, ".go-mcp", "logs", sessionID+".log")
    file, _ := os.Create(logPath)
    handler := slog.NewJSONHandler(file, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    })
    return &Logger{
        logger:    slog.New(handler),
        sessionID: sessionID,
        logFile:   file,
    }
}

func (l *Logger) Info(msg string, args ...any) {
    l.logger.Info(msg, args...)
}
```

### Target State (CLI)

```go
// libs/mcp/server/server.go
import "github.com/databricks/cli/libs/log"

func (s *Server) Start(ctx context.Context) error {
    logger := log.GetLogger(ctx)
    log.Infof(ctx, "Starting MCP server (session: %s)", sessionID)
    // ...
}
```

### CLI Logging API

```go
// Available functions from libs/log
log.Trace(ctx, msg)      // Trace level
log.Debug(ctx, msg)      // Debug level
log.Info(ctx, msg)       // Info level
log.Warn(ctx, msg)       // Warn level
log.Error(ctx, msg)      // Error level

log.Tracef(ctx, format, args...)  // With formatting
log.Debugf(ctx, format, args...)
log.Infof(ctx, format, args...)
log.Warnf(ctx, format, args...)
log.Errorf(ctx, format, args...)
```

### Key Differences

| Feature | go-mcp | CLI | Migration Strategy |
|---------|--------|-----|-------------------|
| **Handler** | JSON to file | Configurable | Use CLI's handler setup |
| **Session logs** | Separate file per session | Single log stream | Remove session-specific files |
| **Context** | Not required | Required `ctx` param | Add `ctx` to all log calls |
| **Configuration** | Custom log levels | Via CLI flags | Use CLI's log level flags |

### Integration Steps

1. Remove `pkg/logging/` entirely
2. Replace all `logger.Info(...)` with `log.Info(ctx, ...)`
3. Remove session-specific log file creation
4. Trust CLI's logging configuration
5. Add `ctx context.Context` parameter to functions that log

**Difficulty**: ⭐⭐ Medium (straightforward API conversion, many call sites)

## 3. Configuration Integration

### Current State (go-mcp)

```go
// pkg/config/config.go
import "github.com/spf13/viper"

type Config struct {
    AllowDeployment    bool
    WithWorkspaceTools bool
    WarehouseID        string
    DatabricksHost     string
    IoConfig           *IoConfig
}

func Load() *Config {
    viper.SetConfigName("config")
    viper.AddConfigPath("$HOME/.go-mcp")
    viper.ReadInConfig()

    var cfg Config
    viper.Unmarshal(&cfg)
    return &cfg
}
```

### Target State (CLI)

```go
// cmd/apps/mcp/start.go
package mcp

import (
    "github.com/databricks/cli/libs/cmdctx"
    "github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
    var warehouseID string
    var allowDeployment bool

    cmd := &cobra.Command{
        Use:   "start",
        Short: "Start the MCP server",
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx := cmd.Context()

            // Create MCP config from flags
            cfg := &mcp.Config{
                WarehouseID:     warehouseID,
                AllowDeployment: allowDeployment,
            }

            // Use CLI's Databricks client from context
            w := cmdctx.GetWorkspaceClient(ctx)

            // Start server
            server := mcp.NewServer(cfg, w)
            return server.Start(ctx)
        },
    }

    // Define flags
    cmd.Flags().StringVar(&warehouseID, "warehouse-id", "", "Databricks warehouse ID")
    cmd.Flags().BoolVar(&allowDeployment, "allow-deployment", false, "Enable deployment tools")

    return cmd
}
```

### CLI Configuration Patterns

1. **Flags**: Command-specific configuration via Cobra flags
2. **Context**: Databricks client configuration via `cmdctx`
3. **No config files**: CLI doesn't use config files for tool settings

### cmdctx API

```go
// Get Databricks clients (auto-configured from profile)
w := cmdctx.GetWorkspaceClient(ctx)   // Workspace client
a := cmdctx.GetAccountClient(ctx)     // Account client

// Execution ID (unique per CLI invocation)
execID := cmdctx.ExecId(ctx)

// Custom context values (for MCP-specific state)
type mcpKey int
const workDirKey = mcpKey(1)

ctx = context.WithValue(ctx, workDirKey, "/path/to/workdir")
workDir := ctx.Value(workDirKey).(string)
```

### Integration Steps

1. Remove `pkg/config/` entirely
2. Define flags in command files (not config struct)
3. Use `cmdctx.GetWorkspaceClient(ctx)` for Databricks access
4. Store MCP session state in context with custom keys
5. Remove Viper dependency

**Difficulty**: ⭐⭐⭐ Complex (fundamental pattern change)

## 4. Session/Context Integration

### Current State (go-mcp)

```go
// pkg/session/session.go
type Session struct {
    ID      string
    WorkDir string
    Metrics *Metrics
    mu      sync.RWMutex
}

func (s *Session) SetWorkDir(dir string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.WorkDir = dir
}

func (s *Session) GetWorkDir() string {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.WorkDir
}
```

### Target State (CLI)

```go
// libs/mcp/session/context.go
package session

type mcpKey int

const (
    sessionIDKey = mcpKey(1)
    workDirKey   = mcpKey(2)
    metricsKey   = mcpKey(3)
)

func WithSessionID(ctx context.Context, id string) context.Context {
    return context.WithValue(ctx, sessionIDKey, id)
}

func GetSessionID(ctx context.Context) string {
    v := ctx.Value(sessionIDKey)
    if v == nil {
        return ""
    }
    return v.(string)
}

func WithWorkDir(ctx context.Context, dir string) context.Context {
    return context.WithValue(ctx, workDirKey, dir)
}

func GetWorkDir(ctx context.Context) string {
    v := ctx.Value(workDirKey)
    if v == nil {
        return ""
    }
    return v.(string)
}
```

### Integration Steps

1. Convert `Session` struct to context-based state
2. Use context.WithValue for session data
3. Remove sync.RWMutex (context is immutable)
4. Pass updated context through call chain
5. Keep `pkg/session` but refactor to use context internally

**Difficulty**: ⭐⭐ Medium (concept change but straightforward)

## 5. Error Handling Integration

### Current State (go-mcp)

```go
// pkg/errors/errors.go
import "github.com/modelcontextprotocol/go-sdk/mcp"

func InvalidParams(message string, details ...interface{}) error {
    return mcp.NewError(-32602, message, details)
}

func InternalError(message string, details ...interface{}) error {
    return mcp.NewError(-32603, message, details)
}
```

### Target State (CLI)

Option 1: Keep MCP-specific errors in `libs/mcp/errors/`:
```go
// libs/mcp/errors/errors.go - KEEP AS IS
// MCP protocol errors are domain-specific
```

Option 2: Merge with `libs/errs/` if generic:
```go
// libs/errs/errors.go - Only if errors are generic
```

**Decision**: Keep `libs/mcp/errors/` for MCP-specific error codes. These are protocol-defined.

### Integration Steps

1. Move `pkg/errors/` to `libs/mcp/errors/`
2. No changes needed (MCP protocol errors)
3. Continue using for MCP tool responses

**Difficulty**: ⭐ Easy (minimal changes)

## 6. Command Structure Integration

### Current Structure (go-mcp)

```
databricks-cli <command> <args>

New structure:
databricks apps mcp start
databricks apps mcp check
databricks apps mcp config show
```

### CLI Command Hierarchy

```
databricks
├── bundle (group: development)
│   ├── deploy
│   ├── destroy
│   └── ...
├── workspace (group: workspace)
│   ├── list
│   ├── export
│   └── ...
├── apps (NEW, group: development)
│   └── mcp (NEW)
│       ├── start
│       ├── check
│       └── config
```

### Registration in cmd/cmd.go

```go
// cmd/cmd.go
import "github.com/databricks/cli/cmd/apps"

func New() *cobra.Command {
    // ... existing code ...

    // Add apps command
    cmd.AddCommand(apps.NewAppsCmd())

    return cmd
}
```

### Integration Steps

1. Create `cmd/apps/apps.go` with grouped apps command
2. Create `cmd/apps/mcp/` directory
3. Split `cli.go` into `start.go`, `check.go`, `config.go`
4. Register apps command in `cmd/cmd.go`
5. Test with `databricks apps mcp --help`

**Difficulty**: ⭐⭐ Medium (structural reorganization)

## 7. Testing Integration

### Current State (go-mcp)

```bash
# Unit tests
go test ./...

# Integration tests
go test ./test/integration/...

# Coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Target State (CLI)

```bash
# Unit tests (via CLI Makefile)
make test

# Acceptance tests
gotestsum --format pkgname ./acceptance/apps/mcp/...

# Coverage
make test-coverage
```

### CLI Testing Patterns

**Unit Tests**: Alongside code in `libs/mcp/`
```go
// libs/mcp/providers/io/scaffold_test.go
func TestScaffold(t *testing.T) {
    // Test implementation
}
```

**Acceptance Tests**: In `acceptance/apps/mcp/`
```go
// acceptance/apps/mcp/start_test.go
func TestMCPStart(t *testing.T) {
    t.Skipif(os.Getenv("CLOUD_ENV") == "")

    // Full integration test
    cmd := exec.Command("databricks", "apps", "mcp", "start")
    // ...
}
```

### Integration Steps

1. Keep unit tests alongside code in `libs/mcp/`
2. Move `test/integration/` tests to `acceptance/apps/mcp/`
3. Adapt tests to use `gotestsum` assertions
4. Add `make test` targets for MCP
5. Run tests via CLI test infrastructure

**Difficulty**: ⭐⭐ Medium (test framework adaptation)

## 8. Build System Integration

### Current State (go-mcp)

```makefile
# Makefile
build:
    go build -o go-mcp ./cmd/go-mcp

test:
    go test ./...

install:
    go install ./cmd/go-mcp
```

### Target State (CLI)

```makefile
# Makefile (add to existing)
.PHONY: test-mcp
test-mcp:
    go test ./libs/mcp/...
    go test ./acceptance/apps/mcp/...

.PHONY: build
build: ## Build the CLI binary
    # Existing build includes MCP automatically
    go build -o bin/databricks ./cmd/databricks
```

### Integration Steps

1. Remove standalone go-mcp Makefile targets
2. Add `test-mcp` target to CLI Makefile (optional)
3. MCP code builds automatically with CLI
4. Update CI/CD to test MCP alongside CLI

**Difficulty**: ⭐ Easy (minimal changes)

## Summary: Integration Checklist

### Phase 2 Integration Tasks

- [ ] **CLI Framework**
  - [ ] Create `cmd/apps/apps.go`
  - [ ] Create `cmd/apps/mcp/` commands
  - [ ] Register in `cmd/cmd.go`

- [ ] **Logging**
  - [ ] Replace all `logger.X()` with `log.X(ctx, ...)`
  - [ ] Remove `pkg/logging/`
  - [ ] Remove session log files

- [ ] **Configuration**
  - [ ] Convert Viper config to Cobra flags
  - [ ] Use `cmdctx.GetWorkspaceClient(ctx)`
  - [ ] Remove `pkg/config/`
  - [ ] Remove Viper dependency

- [ ] **Session/Context**
  - [ ] Convert Session to context-based
  - [ ] Add WithWorkDir/GetWorkDir helpers
  - [ ] Update all session call sites

- [ ] **Error Handling**
  - [ ] Move `pkg/errors/` to `libs/mcp/errors/`
  - [ ] Update import paths

- [ ] **Command Structure**
  - [ ] Split `cli.go` into focused commands
  - [ ] Test `databricks apps mcp` commands
  - [ ] Verify help text and flags

- [ ] **Testing**
  - [ ] Move integration tests to `acceptance/`
  - [ ] Adapt to CLI test patterns
  - [ ] Add MCP test targets

- [ ] **Build System**
  - [ ] Integrate with CLI Makefile
  - [ ] Update CI/CD pipelines
  - [ ] Test builds

## Risk Assessment

| Integration Point | Risk | Impact | Mitigation |
|------------------|------|--------|------------|
| Logging | Low | Medium | Well-defined API, many call sites | Thorough testing |
| Configuration | Medium | High | Pattern change affects initialization | Careful refactoring, staged commits |
| Context/Session | Low | Medium | Concept shift but localized | Good test coverage |
| Command Structure | Low | Medium | Structural change | Test all command paths |
| Testing | Low | Low | Framework change | Gradual migration |
| Build System | Very Low | Low | Additive changes | Verify builds |

**Overall Risk**: Medium. Configuration migration is the most complex, but all changes are well-understood patterns.

## Notes

1. **Databricks Client**: CLI automatically provides authenticated Databricks clients via `cmdctx`. No need to recreate auth logic.

2. **Context Propagation**: All CLI commands receive a context. This context must be threaded through MCP server and tool handlers.

3. **Flag Naming**: Follow CLI conventions (`--warehouse-id`, not `--warehouse_id`).

4. **Error Messages**: CLI has user-friendly error formatting. MCP errors should integrate with this.

5. **Testing Philosophy**: CLI prefers acceptance tests for end-to-end scenarios. Unit tests for isolated logic.

## Next Steps

After Phase 1 documentation is complete, Phase 2 will execute these integration tasks systematically, validating each integration point before moving to the next.
