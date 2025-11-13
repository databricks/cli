# Phase 3: Infrastructure Integration

**bd Issue**: `parity-41` (task)
**Status**: Open | **Priority**: P0
**Depends on**: parity-40 (Phase 2)
**Blocks**: parity-42 (Phase 4)

## Overview

This phase replaces go-mcp's custom infrastructure (logging, config, error handling) with Databricks CLI equivalents to ensure proper integration.

## Duration

3-4 hours

## Prerequisites

- Phase 2 completed
- Code structure in place
- Initial build attempted

## Tasks

### 3.1 Logging Migration

**Objective**: Replace custom pkg/logging with libs/log

**Current Pattern** (go-mcp):
```go
import "github.com/appdotbuild/go-mcp/pkg/logging"

logger := logging.NewLogger(sessionID, logLevel)
logger.Info("message", slog.String("key", "value"))
```

**Target Pattern** (Databricks CLI):
```go
import "github.com/databricks/cli/libs/log"

logger := log.GetLogger(ctx)
logger.Info("message", slog.String("key", "value"))
```

**Files to Update**:
1. `libs/mcp/server/server.go`
2. `libs/mcp/providers/databricks/provider.go`
3. `libs/mcp/providers/io/provider.go`
4. `libs/mcp/providers/workspace/provider.go`
5. `libs/mcp/providers/deployment/provider.go`
6. `libs/mcp/trajectory/tracker.go`
7. `libs/mcp/session/session.go`
8. `cmd/apps/mcp/*.go`

**Search and Replace**:
```bash
cd /Users/fabian.jakobs/Workspaces/cli

# Find all files using old logging
grep -r "pkg/logging" cmd/apps/mcp libs/mcp

# For each file, replace:
# 1. Import statement
sed -i '' 's|"github.com/databricks/cli/libs/mcp/logging"|"github.com/databricks/cli/libs/log"|g' \
  $(find cmd/apps/mcp libs/mcp -name "*.go")

# 2. Logger initialization (manual review needed)
#    Change: logger := logging.NewLogger(sessionID, level)
#    To:     logger := log.GetLogger(ctx)
```

**Key Changes**:

1. **Logger Creation**:
```go
// Before
logger := logging.NewLogger(sessionID, logLevel)

// After
logger := log.GetLogger(ctx)
```

2. **Logger Passing**:
```go
// Before
func NewProvider(config *config.Config, logger *slog.Logger) *Provider

// After
func NewProvider(ctx context.Context, config *Config) *Provider {
    logger := log.GetLogger(ctx)
    // ...
}
```

3. **Remove Session-based Log Files**:
- Remove log file creation in session
- Rely on CLI's logging infrastructure
- Keep trajectory logging (separate concern)

**Verification**:
```bash
# No references to old logging package
grep -r "pkg/logging" cmd/apps/mcp libs/mcp || echo "✓ Clean"

# Build should improve
go build ./cmd/cli
```

### 3.2 Configuration Integration

**Objective**: Integrate with Databricks CLI config system

**Current Pattern** (go-mcp):
```go
import (
    "github.com/appdotbuild/go-mcp/pkg/config"
    "github.com/spf13/viper"
)

cfg := config.LoadConfig()
cfg.Validate()
```

**Target Pattern** (Databricks CLI):
```go
// Use cobra flags and CLI context
type Config struct {
    AllowDeployment    bool
    WithWorkspaceTools bool
    IoConfig           *IoConfig
    WarehouseID        string
    DatabricksHost     string
}

// Load from command flags
```

**Strategy**:
1. Create `libs/mcp/config/config.go` with config types
2. Use cobra flags for configuration (no Viper)
3. Load config from flags in command execution
4. Remove dependency on separate config file (use CLI's databricks config)

**Implementation**:

1. **Create `libs/mcp/config/config.go`**:
```go
package config

import "fmt"

type Config struct {
    AllowDeployment    bool
    WithWorkspaceTools bool
    IoConfig           *IoConfig
    WarehouseID        string
    DatabricksHost     string
}

type IoConfig struct {
    Template   *TemplateConfig
    Validation *ValidationConfig
}

type TemplateConfig struct {
    Name string
}

type ValidationConfig struct {
    Command     string
    DockerImage string
}

func (c *Config) Validate() error {
    if c.WarehouseID == "" {
        return fmt.Errorf("warehouse_id is required")
    }
    return nil
}
```

2. **Update `cmd/apps/mcp/start.go`**:
```go
func newStartCommand() *cobra.Command {
    var cfg mcpconfig.Config

    cmd := &cobra.Command{
        Use:   "start",
        Short: "Start MCP server",
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx := cmd.Context()

            // Validate config
            if err := cfg.Validate(); err != nil {
                return err
            }

            // Create and start server
            server := mcpserver.NewServer(&cfg, ctx)
            return server.Run(ctx)
        },
    }

    // Add flags
    cmd.Flags().BoolVar(&cfg.AllowDeployment, "allow-deployment", false, "Allow deployment operations")
    cmd.Flags().BoolVar(&cfg.WithWorkspaceTools, "with-workspace-tools", true, "Enable workspace tools")
    cmd.Flags().StringVar(&cfg.WarehouseID, "warehouse-id", "", "Databricks warehouse ID (required)")
    cmd.Flags().StringVar(&cfg.DatabricksHost, "databricks-host", "", "Databricks workspace URL")

    cmd.MarkFlagRequired("warehouse-id")

    return cmd
}
```

3. **Remove Viper Dependency**:
```bash
# Remove from imports across codebase
find cmd/apps/mcp libs/mcp -name "*.go" -exec sed -i '' \
  '/github.com\/spf13\/viper/d' {} +
```

**Verification**:
```bash
# No viper imports
grep -r "viper" cmd/apps/mcp libs/mcp || echo "✓ Clean"

# Config flags work
./cli apps mcp start --help | grep warehouse-id
```

### 3.3 Session Management Integration

**Objective**: Integrate session with CLI context patterns

**Current Pattern** (go-mcp):
```go
session := session.NewSession()
session.SetWorkDir("/path")
workDir := session.GetWorkDir()
```

**Target Pattern** (Databricks CLI):
```go
// Use context values for session state
ctx = session.WithWorkDir(ctx, "/path")
workDir := session.GetWorkDir(ctx)
```

**Changes**:

1. **Update `libs/mcp/session/session.go`**:
```go
package session

import (
    "context"
    "sync"
)

type contextKey int

const (
    workDirKey contextKey = iota
    sessionIDKey
)

// WithWorkDir adds work directory to context
func WithWorkDir(ctx context.Context, workDir string) context.Context {
    return context.WithValue(ctx, workDirKey, workDir)
}

// GetWorkDir retrieves work directory from context
func GetWorkDir(ctx context.Context) string {
    if v := ctx.Value(workDirKey); v != nil {
        return v.(string)
    }
    return ""
}

// Session stores mutable state
type Session struct {
    mu      sync.RWMutex
    metrics map[string]interface{}
    Tracker *trajectory.Tracker
}

func NewSession() *Session {
    return &Session{
        metrics: make(map[string]interface{}),
    }
}

// Context-based accessors
func WithSession(ctx context.Context, s *Session) context.Context {
    return context.WithValue(ctx, sessionIDKey, s)
}

func GetSession(ctx context.Context) *Session {
    if v := ctx.Value(sessionIDKey); v != nil {
        return v.(*Session)
    }
    return nil
}
```

2. **Update Providers**:
```go
// In scaffold_data_app
ctx = session.WithWorkDir(ctx, projectPath)

// In workspace tools
workDir := session.GetWorkDir(ctx)
if workDir == "" {
    return errors.New("work directory not set")
}
```

**Verification**:
- Session state accessible via context
- Thread-safe operations maintained
- No global state

### 3.4 Error Handling Standardization

**Objective**: Align error handling with CLI patterns

**Current Pattern** (go-mcp):
```go
import "github.com/databricks/cli/libs/mcp/errors"

return errors.InvalidParams("message")
return errors.InternalError("message", details)
```

**Target Pattern** (Databricks CLI):
```go
import (
    "fmt"
    "github.com/databricks/cli/libs/diag"
)

// For MCP protocol errors, keep existing pattern
// For CLI errors, use standard errors
return fmt.Errorf("error: %w", err)

// Or for diagnostics:
return diag.Errorf("error occurred: %s", msg)
```

**Strategy**:
1. Keep `libs/mcp/errors` for MCP protocol errors (JSON-RPC codes)
2. Use standard errors in command layer
3. Convert between error types at boundary

**Implementation**:
```go
// In cmd/apps/mcp/start.go
func (cmd *startCommand) RunE(cmd *cobra.Command, args []string) error {
    // ...
    if err := server.Run(ctx); err != nil {
        // MCP errors come through here
        // Convert to CLI error format if needed
        return fmt.Errorf("MCP server error: %w", err)
    }
    return nil
}
```

**Verification**:
- MCP protocol errors still valid JSON-RPC
- Command errors follow CLI patterns
- Error messages clear and actionable

### 3.5 Context Propagation

**Objective**: Ensure proper context usage throughout

**Pattern**:
```go
// All functions should accept context as first parameter
func NewProvider(ctx context.Context, cfg *Config) *Provider
func (p *Provider) HandleTool(ctx context.Context, args map[string]interface{}) (*Result, error)
```

**Files to Check**:
- All provider constructors
- All tool handlers
- Server initialization
- Sandbox operations

**Changes**:
```bash
# Review all function signatures
grep -r "func.*New" libs/mcp | grep -v "_test.go"
grep -r "func.*Handle" libs/mcp | grep -v "_test.go"

# Ensure context.Context is first parameter
```

**Verification**:
- Context flows from command → server → providers → sandbox
- Cancellation works correctly
- Logger retrieval works via context

### 3.6 Remove Unused Dependencies

**Objective**: Clean up dependencies no longer needed

**To Remove**:
- `github.com/spf13/viper` (if not used elsewhere in CLI)

**Steps**:
```bash
cd /Users/fabian.jakobs/Workspaces/cli

# Remove unused imports
go mod tidy

# Verify no orphaned dependencies
go mod why github.com/spf13/viper
```

### 3.7 Build Verification

**Objective**: Ensure code compiles cleanly

**Steps**:
```bash
cd /Users/fabian.jakobs/Workspaces/cli

# Clean build
go clean
go build ./cmd/cli

# Verify no errors
echo $?  # Should be 0

# Test basic functionality
./cli apps mcp --help
```

**Success Criteria**:
- No compilation errors
- No import errors
- Commands execute without panic
- Help text displays correctly

### 3.8 Create Infrastructure Integration Commit

**Objective**: Commit the infrastructure changes

**Commit Message**:
```
Integrate MCP with Databricks CLI infrastructure (migration step 2/3)

Replace go-mcp custom infrastructure with Databricks CLI equivalents:

- Logging: Migrate from pkg/logging to libs/log
- Config: Replace Viper with cobra flags and CLI config patterns
- Session: Integrate with context-based state management
- Error handling: Align with CLI error patterns
- Context propagation: Ensure proper context flow

Changes:
- Updated all providers to use libs/log
- Removed dependency on Viper for configuration
- Session management now context-based
- MCP protocol errors preserved, CLI errors standardized

Phase 3 of 5: Infrastructure Integration
Status: Builds cleanly, ready for testing integration

Related:
- Phase 2: Code Structure Migration (complete)
- Phase 4: Testing & Build Integration (next)
```

**Steps**:
```bash
cd /Users/fabian.jakobs/Workspaces/cli
git add -A
git commit -F- <<'EOF'
Integrate MCP with Databricks CLI infrastructure (migration step 2/3)

[Full message above]
EOF
```

## Deliverables

1. ✅ Logging migrated to libs/log
2. ✅ Configuration integrated with CLI patterns
3. ✅ Session management context-based
4. ✅ Error handling standardized
5. ✅ Context propagation verified
6. ✅ Unused dependencies removed
7. ✅ Clean build achieved
8. ✅ Infrastructure integration commit created

## Verification Steps

- [ ] No pkg/logging references remain
- [ ] No Viper dependencies in MCP code
- [ ] All providers use log.GetLogger(ctx)
- [ ] Session state flows through context
- [ ] Code builds without errors: `go build ./cmd/cli`
- [ ] Commands execute: `./cli apps mcp --help`
- [ ] Logger works in providers

## Known Issues

None expected - this phase should result in a clean, building codebase.

## Next Phase

**Phase 4: Testing & Build Integration** - Adapt tests and integrate with CLI build system
