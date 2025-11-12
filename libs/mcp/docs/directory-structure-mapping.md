# Directory Structure Mapping

**Generated**: 2025-11-12
**For**: parity-39 (Phase 1: Repository Setup & Analysis)

## Overview

This document maps the current go-mcp directory structure to the target Databricks CLI structure, providing a clear migration path for all code and assets.

## Current go-mcp Structure

```
app-mcp/
├── cmd/
│   └── go-mcp/                    # CLI entry point (2 files)
│       ├── main.go
│       └── cli.go
├── pkg/
│   ├── config/                    # Configuration (2 files)
│   ├── errors/                    # Error types (1 file)
│   ├── fileutil/                  # File utilities (2 files)
│   ├── logging/                   # Logging system (2 files)
│   ├── mcp/                       # MCP server (2 files)
│   ├── pathutil/                  # Path utilities (2 files)
│   ├── providers/                 # Tool providers
│   │   ├── databricks/            # Databricks provider (9 files)
│   │   ├── deployment/            # Deployment provider (2 files)
│   │   ├── io/                    # IO/scaffolding provider (9 files)
│   │   ├── workspace/             # Workspace provider (10 files)
│   │   └── registry.go            # Provider registry
│   ├── sandbox/                   # Execution abstraction
│   │   ├── sandbox.go             # Interface definition
│   │   ├── factory.go             # Factory pattern
│   │   ├── local/                 # Local implementation (5 files)
│   │   └── dagger/                # Dagger implementation (4 files)
│   ├── session/                   # Session management (2 files)
│   ├── templates/                 # Template abstraction (1 file)
│   ├── trajectory/                # History logging (4 files)
│   └── version/                   # Version info (1 file)
├── internal/
│   └── templates/                 # Embedded templates
│       └── trpc/                  # TRPC template (embedded)
├── test/
│   └── integration/               # Integration tests (7 files)
├── plans/                         # Planning documents
├── docs/                          # Documentation
├── .beads/                        # Issue tracking
├── Makefile                       # Build system
├── go.mod                         # Dependencies
└── README.md                      # Project README

Total Go files: ~60
Lines of code: ~14,800
```

## Target Databricks CLI Structure

```
cli/
├── cmd/
│   ├── apps/                      # NEW: Apps command group
│   │   ├── apps.go                # NEW: Apps root command
│   │   └── mcp/                   # NEW: MCP subcommand
│   │       ├── mcp.go             # MCP command entry
│   │       ├── start.go           # Start MCP server
│   │       ├── check.go           # Environment check
│   │       └── config.go          # Config commands
│   └── cmd.go                     # MODIFY: Register apps command
├── libs/
│   ├── mcp/                       # NEW: MCP library code
│   │   ├── server/                # MCP server implementation
│   │   │   ├── server.go          # Server wrapper
│   │   │   └── health.go          # Health checks
│   │   ├── providers/             # Tool providers
│   │   │   ├── databricks/        # Databricks provider
│   │   │   ├── io/                # IO/scaffolding provider
│   │   │   ├── workspace/         # Workspace provider
│   │   │   ├── deployment/        # Deployment provider
│   │   │   └── registry.go        # Provider registry
│   │   ├── sandbox/               # Execution abstraction
│   │   │   ├── sandbox.go         # Interface
│   │   │   ├── factory.go         # Factory
│   │   │   ├── local/             # Local impl
│   │   │   └── dagger/            # Dagger impl
│   │   ├── session/               # Session management
│   │   ├── trajectory/            # History logging
│   │   ├── templates/             # Template abstraction
│   │   ├── pathutil/              # Path utilities
│   │   ├── fileutil/              # File utilities
│   │   ├── errors/                # Error types
│   │   └── version/               # Version info
│   ├── apps/                      # EXISTING: App runtime (keep as is)
│   ├── log/                       # EXISTING: Use instead of pkg/logging
│   └── cmdctx/                    # EXISTING: Use instead of pkg/config
├── internal/
│   └── mcp/                       # NEW: Internal MCP code
│       └── templates/             # Embedded templates
│           └── trpc/              # TRPC template
├── acceptance/
│   └── apps/                      # NEW: Acceptance tests
│       └── mcp/                   # MCP acceptance tests
└── go.mod                         # MODIFY: Add MCP dependencies
```

## Detailed Migration Map

### Commands (cmd/)

| Source | Target | Action | Notes |
|--------|--------|--------|-------|
| `cmd/go-mcp/main.go` | `cmd/apps/mcp/mcp.go` | Adapt | Convert to Cobra subcommand |
| `cmd/go-mcp/cli.go` | `cmd/apps/mcp/start.go` | Split | Extract start logic |
| `cmd/go-mcp/cli.go` | `cmd/apps/mcp/check.go` | Split | Extract check logic |
| `cmd/go-mcp/cli.go` | `cmd/apps/mcp/config.go` | Split | Extract config logic |
| N/A | `cmd/apps/apps.go` | Create | New root apps command |

**Key Changes**:
- Split monolithic `cli.go` into focused command files
- Adapt to CLI's command group pattern (`databricks apps mcp`)
- Register in `cmd/cmd.go` root command

### Libraries (pkg/ → libs/)

#### Server & Core

| Source | Target | Action | Notes |
|--------|--------|--------|-------|
| `pkg/mcp/server.go` | `libs/mcp/server/server.go` | Move | Direct port |
| `pkg/mcp/health.go` | `libs/mcp/server/health.go` | Move | Direct port |

#### Providers

| Source | Target | Action | Notes |
|--------|--------|--------|-------|
| `pkg/providers/databricks/*` | `libs/mcp/providers/databricks/*` | Move | 9 files, direct port |
| `pkg/providers/io/*` | `libs/mcp/providers/io/*` | Move | 9 files, direct port |
| `pkg/providers/workspace/*` | `libs/mcp/providers/workspace/*` | Move | 10 files, direct port |
| `pkg/providers/deployment/*` | `libs/mcp/providers/deployment/*` | Move | 2 files, direct port |
| `pkg/providers/registry.go` | `libs/mcp/providers/registry.go` | Move | Direct port |

#### Sandbox

| Source | Target | Action | Notes |
|--------|--------|--------|-------|
| `pkg/sandbox/sandbox.go` | `libs/mcp/sandbox/sandbox.go` | Move | Interface definition |
| `pkg/sandbox/factory.go` | `libs/mcp/sandbox/factory.go` | Move | Factory pattern |
| `pkg/sandbox/local/*` | `libs/mcp/sandbox/local/*` | Move | 5 files, direct port |
| `pkg/sandbox/dagger/*` | `libs/mcp/sandbox/dagger/*` | Move | 4 files, direct port |

#### Utilities & Support

| Source | Target | Action | Notes |
|--------|--------|--------|-------|
| `pkg/session/*` | `libs/mcp/session/*` | Adapt | Integrate with cmdctx |
| `pkg/trajectory/*` | `libs/mcp/trajectory/*` | Move | 4 files, direct port |
| `pkg/templates/*` | `libs/mcp/templates/*` | Move | Direct port |
| `pkg/pathutil/*` | `libs/mcp/pathutil/*` | Move | 2 files, direct port |
| `pkg/fileutil/*` | `libs/mcp/fileutil/*` | Move | 2 files, direct port |
| `pkg/errors/*` | `libs/mcp/errors/*` | Adapt | May merge with libs/errs |
| `pkg/version/*` | `libs/mcp/version/*` | Adapt | Use CLI version system |
| `pkg/config/*` | REMOVE | Replace | Use libs/cmdctx patterns |
| `pkg/logging/*` | REMOVE | Replace | Use libs/log |

### Internal Code

| Source | Target | Action | Notes |
|--------|--------|--------|-------|
| `internal/templates/trpc/*` | `internal/mcp/templates/trpc/*` | Move | Embedded templates |
| `internal/templates/embed.go` | `internal/mcp/templates/embed.go` | Move | Embedding logic |

### Tests

| Source | Target | Action | Notes |
|--------|--------|--------|-------|
| `test/integration/*` | `acceptance/apps/mcp/*` | Adapt | Convert to CLI test patterns |
| `pkg/*/\*_test.go` | `libs/mcp/*/\*_test.go` | Move | Unit tests move with code |

### Documentation & Assets

| Source | Target | Action | Notes |
|--------|--------|--------|-------|
| `README.md` | `libs/mcp/README.md` | Adapt | CLI-focused version |
| `plans/*` | `libs/mcp/docs/plans/*` | Move | Keep for reference |
| `docs/*` | `libs/mcp/docs/*` | Move | Technical docs |
| `.beads/*` | N/A | Skip | Issue tracking (not needed in CLI) |
| `Makefile` | `Makefile` | Integrate | Add MCP targets to CLI Makefile |

## Import Path Changes

All import paths must be updated from `github.com/databricks/go-mcp` to `github.com/databricks/cli`:

```go
// Before
import "github.com/databricks/go-mcp/pkg/providers/databricks"
import "github.com/databricks/go-mcp/pkg/sandbox"

// After
import "github.com/databricks/cli/libs/mcp/providers/databricks"
import "github.com/databricks/cli/libs/mcp/sandbox"
```

**Estimated Changes**: ~200 import statements across 60 files

## CLI Integration Points

### 1. Command Registration

```go
// cmd/cmd.go
import "github.com/databricks/cli/cmd/apps"

// In NewRootCmd():
cmd.AddCommand(apps.NewAppsCmd())
```

### 2. Logging Integration

Replace `pkg/logging` with `libs/log`:

```go
// Before
logger := logging.NewLogger(sessionID)

// After
logger := log.NewLogger(ctx)
```

### 3. Configuration Integration

Replace `pkg/config` with `libs/cmdctx`:

```go
// Before
cfg := config.Load()

// After
cfg := cmdctx.GetConfig(ctx)
```

### 4. Session Integration

Merge `pkg/session` with `cmdctx`:

```go
// Before
session := session.NewSession()
session.SetWorkDir("/path")

// After
ctx := cmdctx.WithWorkDir(ctx, "/path")
workDir := cmdctx.GetWorkDir(ctx)
```

## File Count Summary

| Category | Source (go-mcp) | Target (CLI) | Change |
|----------|----------------|--------------|--------|
| Command files | 2 | 5 | +3 (split for clarity) |
| Library files | 50 | 48 | -2 (logging, config removed) |
| Internal files | 1 | 1 | 0 |
| Test files | 7 | 7 | 0 |
| **Total Go files** | **60** | **61** | **+1** |

## Directory Creation Checklist

- [ ] Create `cmd/apps/`
- [ ] Create `cmd/apps/mcp/`
- [ ] Create `libs/mcp/`
- [ ] Create `libs/mcp/server/`
- [ ] Create `libs/mcp/providers/`
- [ ] Create `libs/mcp/providers/databricks/`
- [ ] Create `libs/mcp/providers/io/`
- [ ] Create `libs/mcp/providers/workspace/`
- [ ] Create `libs/mcp/providers/deployment/`
- [ ] Create `libs/mcp/sandbox/`
- [ ] Create `libs/mcp/sandbox/local/`
- [ ] Create `libs/mcp/sandbox/dagger/`
- [ ] Create `libs/mcp/session/`
- [ ] Create `libs/mcp/trajectory/`
- [ ] Create `libs/mcp/templates/`
- [ ] Create `libs/mcp/pathutil/`
- [ ] Create `libs/mcp/fileutil/`
- [ ] Create `libs/mcp/errors/`
- [ ] Create `libs/mcp/version/`
- [ ] Create `internal/mcp/`
- [ ] Create `internal/mcp/templates/`
- [ ] Create `internal/mcp/templates/trpc/`
- [ ] Create `acceptance/apps/`
- [ ] Create `acceptance/apps/mcp/`

## Notes

1. **libs/apps vs libs/mcp**: The existing `libs/apps` is for running Databricks apps (Python, Node runtimes). Our `libs/mcp` is for the MCP server and tools. These are complementary, not conflicting.

2. **Import Path Consistency**: The CLI uses `github.com/databricks/cli` for all imports. We'll follow this pattern.

3. **Testing Pattern**: The CLI uses `acceptance/` for end-to-end tests (gotestsum). Unit tests stay alongside code.

4. **Configuration**: The CLI doesn't use Viper. Configuration is context-based via `cmdctx` and flags.

5. **Logging**: The CLI has a sophisticated logging system in `libs/log`. We'll adapt to it.

## Verification

- [ ] All source files accounted for in mapping
- [ ] Import path changes identified
- [ ] Integration points documented
- [ ] Directory structure validated against CLI patterns
- [ ] No naming conflicts with existing CLI code
