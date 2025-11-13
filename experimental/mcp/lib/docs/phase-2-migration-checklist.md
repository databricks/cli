# Phase 2: Migration Checklist

**Generated**: 2025-11-12
**For**: parity-40 (Phase 2: Code Structure Migration)
**Estimated Duration**: 4-6 hours

## Overview

This checklist provides a detailed, step-by-step guide for migrating go-mcp code into the Databricks CLI repository. Each task includes verification steps and rollback procedures.

## Pre-Migration Setup

### S1: Branch Verification

- [ ] Verify `apps-mcp` branch exists and is up to date
  ```bash
  cd /Users/fabian.jakobs/Workspaces/cli
  git checkout apps-mcp
  git pull origin apps-mcp
  git log --oneline -1
  ```
- [ ] Verify no uncommitted changes
  ```bash
  git status
  # Should show: "nothing to commit, working tree clean"
  ```
- [ ] Create backup branch
  ```bash
  git checkout -b apps-mcp-backup
  git checkout apps-mcp
  ```

**Rollback**: `git checkout apps-mcp-backup`

### S2: Dependency Installation

- [ ] Update Databricks SDK
  ```bash
  go get github.com/databricks/databricks-sdk-go@v0.90.0
  go mod tidy
  ```
- [ ] Add MCP dependencies
  ```bash
  go get github.com/modelcontextprotocol/go-sdk@v1.1.0
  go get github.com/zeebo/blake3@v0.2.4
  go get dagger.io/dagger@v0.19.6
  go mod tidy
  ```
- [ ] Verify dependencies
  ```bash
  go mod verify
  go list -m github.com/modelcontextprotocol/go-sdk
  go list -m github.com/zeebo/blake3
  go list -m dagger.io/dagger
  ```
- [ ] Test build with new dependencies
  ```bash
  go build ./cmd/databricks
  ```
- [ ] Commit dependencies
  ```bash
  git add go.mod go.sum
  git commit -m "Add MCP server dependencies

- github.com/modelcontextprotocol/go-sdk v1.1.0
- github.com/zeebo/blake3 v0.2.4
- dagger.io/dagger v0.19.6
- Update databricks-sdk-go to v0.90.0

For parity-40 (Phase 2: Code Structure Migration)"
  ```

**Verification**: `go mod verify && go build ./cmd/databricks`

**Rollback**: `git reset --hard HEAD~1`

## Phase 2A: Directory Structure Creation

### M1: Create Command Directories

- [ ] Create apps command directory
  ```bash
  mkdir -p cmd/apps
  ```
- [ ] Create MCP command directory
  ```bash
  mkdir -p cmd/apps/mcp
  ```
- [ ] Verify directories
  ```bash
  ls -la cmd/apps
  ls -la cmd/apps/mcp
  ```

**Verification**: Both directories should exist and be empty

### M2: Create Library Directories

- [ ] Create MCP library root
  ```bash
  mkdir -p libs/mcp
  ```
- [ ] Create server directory
  ```bash
  mkdir -p libs/mcp/server
  ```
- [ ] Create providers directories
  ```bash
  mkdir -p libs/mcp/providers/{databricks,io,workspace,deployment}
  ```
- [ ] Create sandbox directories
  ```bash
  mkdir -p libs/mcp/sandbox/{local,dagger}
  ```
- [ ] Create support directories
  ```bash
  mkdir -p libs/mcp/{session,trajectory,templates,pathutil,fileutil,errors,version}
  ```
- [ ] Verify structure
  ```bash
  tree -L 3 -d libs/mcp
  ```

**Verification**: All directories should exist with correct structure

### M3: Create Internal Directories

- [ ] Create internal MCP directory
  ```bash
  mkdir -p internal/mcp/templates/trpc
  ```
- [ ] Verify structure
  ```bash
  ls -la internal/mcp/templates/trpc
  ```

**Verification**: Directory path should be complete

### M4: Create Test Directories

- [ ] Create acceptance test directory
  ```bash
  mkdir -p acceptance/apps/mcp
  ```
- [ ] Verify structure
  ```bash
  ls -la acceptance/apps/mcp
  ```

**Verification**: Directory should exist

**Commit Point**:
```bash
git add cmd/apps libs/mcp internal/mcp acceptance/apps
git commit -m "Create directory structure for MCP integration

- cmd/apps/mcp/ for MCP commands
- libs/mcp/ for MCP library code
- internal/mcp/templates/ for embedded templates
- acceptance/apps/mcp/ for acceptance tests

For parity-40 (Phase 2: Code Structure Migration)"
```

**Rollback**: `git reset --hard HEAD~1`

## Phase 2B: Copy Library Code (No Modifications)

### M5: Copy Utilities (No Dependencies)

Copy files that have no external dependencies first.

- [ ] Copy pathutil
  ```bash
  cp -r /Users/fabian.jakobs/Workspaces/app-mcp/pkg/pathutil/* libs/mcp/pathutil/
  ```
- [ ] Copy fileutil
  ```bash
  cp -r /Users/fabian.jakobs/Workspaces/app-mcp/pkg/fileutil/* libs/mcp/fileutil/
  ```
- [ ] Copy version
  ```bash
  cp -r /Users/fabian.jakobs/Workspaces/app-mcp/pkg/version/* libs/mcp/version/
  ```
- [ ] Copy errors
  ```bash
  cp -r /Users/fabian.jakobs/Workspaces/app-mcp/pkg/errors/* libs/mcp/errors/
  ```
- [ ] Verify files copied
  ```bash
  ls libs/mcp/pathutil
  ls libs/mcp/fileutil
  ls libs/mcp/version
  ls libs/mcp/errors
  ```

**Verification**: All files should be present in target directories

### M6: Copy Templates

- [ ] Copy template code
  ```bash
  cp -r /Users/fabian.jakobs/Workspaces/app-mcp/pkg/templates/* libs/mcp/templates/
  ```
- [ ] Copy embedded templates
  ```bash
  cp -r /Users/fabian.jakobs/Workspaces/app-mcp/internal/templates/* internal/mcp/templates/
  ```
- [ ] Verify templates
  ```bash
  ls libs/mcp/templates
  ls internal/mcp/templates
  ```

**Verification**: Template files should be present

### M7: Copy Sandbox

- [ ] Copy sandbox interface
  ```bash
  cp /Users/fabian.jakobs/Workspaces/app-mcp/pkg/sandbox/sandbox.go libs/mcp/sandbox/
  cp /Users/fabian.jakobs/Workspaces/app-mcp/pkg/sandbox/factory.go libs/mcp/sandbox/
  ```
- [ ] Copy local implementation
  ```bash
  cp -r /Users/fabian.jakobs/Workspaces/app-mcp/pkg/sandbox/local/* libs/mcp/sandbox/local/
  ```
- [ ] Copy Dagger implementation
  ```bash
  cp -r /Users/fabian.jakobs/Workspaces/app-mcp/pkg/sandbox/dagger/* libs/mcp/sandbox/dagger/
  ```
- [ ] Verify sandbox
  ```bash
  ls libs/mcp/sandbox
  ls libs/mcp/sandbox/local
  ls libs/mcp/sandbox/dagger
  ```

**Verification**: All sandbox files should be present

### M8: Copy Session

- [ ] Copy session code
  ```bash
  cp -r /Users/fabian.jakobs/Workspaces/app-mcp/pkg/session/* libs/mcp/session/
  ```
- [ ] Verify session
  ```bash
  ls libs/mcp/session
  ```

**Verification**: Session files should be present

### M9: Copy Trajectory

- [ ] Copy trajectory code
  ```bash
  cp -r /Users/fabian.jakobs/Workspaces/app-mcp/pkg/trajectory/* libs/mcp/trajectory/
  ```
- [ ] Verify trajectory
  ```bash
  ls libs/mcp/trajectory
  ```

**Verification**: Trajectory files should be present

### M10: Copy Providers

- [ ] Copy Databricks provider
  ```bash
  cp -r /Users/fabian.jakobs/Workspaces/app-mcp/pkg/providers/databricks/* libs/mcp/providers/databricks/
  ```
- [ ] Copy IO provider
  ```bash
  cp -r /Users/fabian.jakobs/Workspaces/app-mcp/pkg/providers/io/* libs/mcp/providers/io/
  ```
- [ ] Copy Workspace provider
  ```bash
  cp -r /Users/fabian.jakobs/Workspaces/app-mcp/pkg/providers/workspace/* libs/mcp/providers/workspace/
  ```
- [ ] Copy Deployment provider
  ```bash
  cp -r /Users/fabian.jakobs/Workspaces/app-mcp/pkg/providers/deployment/* libs/mcp/providers/deployment/
  ```
- [ ] Copy provider registry
  ```bash
  cp /Users/fabian.jakobs/Workspaces/app-mcp/pkg/providers/registry.go libs/mcp/providers/
  ```
- [ ] Verify providers
  ```bash
  ls libs/mcp/providers/databricks
  ls libs/mcp/providers/io
  ls libs/mcp/providers/workspace
  ls libs/mcp/providers/deployment
  ls libs/mcp/providers/registry.go
  ```

**Verification**: All provider files should be present

### M11: Copy Server

- [ ] Copy server code
  ```bash
  cp -r /Users/fabian.jakobs/Workspaces/app-mcp/pkg/mcp/* libs/mcp/server/
  ```
- [ ] Verify server
  ```bash
  ls libs/mcp/server
  ```

**Verification**: Server files should be present

**Commit Point**:
```bash
git add libs/mcp internal/mcp
git commit -m "Copy MCP library code from go-mcp

Copied all library code with original structure:
- Utilities (pathutil, fileutil, version, errors)
- Templates (code + embedded)
- Sandbox (interface + local + dagger)
- Session and trajectory
- Providers (databricks, io, workspace, deployment)
- MCP server

Import paths still reference old package. Will fix in next phase.

For parity-40 (Phase 2: Code Structure Migration)"
```

**Rollback**: `git reset --hard HEAD~1`

## Phase 2C: Update Import Paths

This is the most extensive task. Use automated tools where possible.

### M12: Update Import Paths (Automated)

- [ ] Create import path update script
  ```bash
  cat > /tmp/update-imports.sh << 'EOF'
  #!/bin/bash
  # Update all import paths from go-mcp to CLI
  find libs/mcp -name "*.go" -type f -exec sed -i '' \
    's|github.com/databricks/go-mcp/pkg|github.com/databricks/cli/libs/mcp|g' {} +

  find libs/mcp -name "*.go" -type f -exec sed -i '' \
    's|github.com/databricks/go-mcp/internal|github.com/databricks/cli/internal/mcp|g' {} +

  echo "Import paths updated"
  EOF
  chmod +x /tmp/update-imports.sh
  ```
- [ ] Run import update script
  ```bash
  /tmp/update-imports.sh
  ```
- [ ] Verify no old imports remain
  ```bash
  grep -r "github.com/databricks/go-mcp" libs/mcp
  # Should return: no matches
  ```
- [ ] Run go mod tidy
  ```bash
  go mod tidy
  ```
- [ ] Verify compilation (expect failures for config/logging)
  ```bash
  go build ./libs/mcp/... 2>&1 | tee /tmp/build-errors.txt
  ```

**Verification**: Build will fail due to missing pkg/config and pkg/logging, but import paths should be correct

**Commit Point**:
```bash
git add libs/mcp
git commit -m "Update import paths from go-mcp to CLI

Changed all imports:
- github.com/databricks/go-mcp/pkg → github.com/databricks/cli/libs/mcp
- github.com/databricks/go-mcp/internal → github.com/databricks/cli/internal/mcp

Build still fails due to missing config/logging integration.

For parity-40 (Phase 2: Code Structure Migration)"
```

**Rollback**: `git reset --hard HEAD~1`

## Phase 2D: Configuration Integration

### M13: Remove Config Dependency

- [ ] Find all config usage
  ```bash
  grep -r "pkg/config" libs/mcp > /tmp/config-usage.txt
  grep -r "config.Config" libs/mcp >> /tmp/config-usage.txt
  cat /tmp/config-usage.txt
  ```
- [ ] Create config adapter in libs/mcp
  ```go
  // libs/mcp/config.go
  package mcp

  // Config holds MCP server configuration
  type Config struct {
      AllowDeployment    bool
      WithWorkspaceTools bool
      WarehouseID        string
      DatabricksHost     string
      IoConfig           *IoConfig
  }

  type IoConfig struct {
      Template   *TemplateConfig
      Validation *ValidationConfig
      Dagger     *DaggerConfig
  }

  // ... other config structs ...
  ```
- [ ] Update server.go to use new config
  ```go
  // libs/mcp/server/server.go
  import "github.com/databricks/cli/libs/mcp"

  func NewServer(cfg *mcp.Config, ...) *Server {
      // ...
  }
  ```
- [ ] Remove all imports of `pkg/config`
- [ ] Test build
  ```bash
  go build ./libs/mcp/...
  ```

**Verification**: `go build ./libs/mcp/...` should succeed

**Commit Point**:
```bash
git add libs/mcp
git commit -m "Replace pkg/config with MCP-local config

Created libs/mcp/config.go with MCP configuration types.
Removed dependency on pkg/config (which used Viper).
Configuration will be populated from command flags in Phase 3.

For parity-40 (Phase 2: Code Structure Migration)"
```

**Rollback**: `git reset --hard HEAD~1`

## Phase 2E: Logging Integration

### M14: Replace Logging Calls

- [ ] Find all logging usage
  ```bash
  grep -r "pkg/logging" libs/mcp > /tmp/logging-usage.txt
  cat /tmp/logging-usage.txt
  ```
- [ ] Update session to not create logger
  ```go
  // libs/mcp/session/session.go
  // Remove: import "github.com/databricks/cli/libs/mcp/logging"
  // Remove: logger *logging.Logger field
  ```
- [ ] Update server to use CLI logging
  ```go
  // libs/mcp/server/server.go
  import "github.com/databricks/cli/libs/log"

  func (s *Server) Start(ctx context.Context) error {
      log.Info(ctx, "Starting MCP server")
      // ...
  }
  ```
- [ ] Replace all `logger.Info(...)` with `log.Info(ctx, ...)`
  - Search pattern: `logger\.(Info|Debug|Warn|Error)`
  - Replace with: `log.$1(ctx, ...)`
- [ ] Add `ctx context.Context` parameter where missing
- [ ] Test build
  ```bash
  go build ./libs/mcp/...
  ```

**Verification**: `go build ./libs/mcp/...` should succeed with no logging imports

**Commit Point**:
```bash
git add libs/mcp
git commit -m "Replace pkg/logging with libs/log

Replaced all logging calls:
- logger.Info(...) → log.Info(ctx, ...)
- logger.Debug(...) → log.Debug(ctx, ...)
- logger.Warn(...) → log.Warn(ctx, ...)
- logger.Error(...) → log.Error(ctx, ...)

Added ctx parameter to functions as needed.
Removed session-specific log file creation.

For parity-40 (Phase 2: Code Structure Migration)"
```

**Rollback**: `git reset --hard HEAD~1`

## Phase 2F: Command Implementation

### M15: Create Apps Root Command

- [ ] Create `cmd/apps/apps.go`
  ```go
  package apps

  import (
      "github.com/databricks/cli/cmd/apps/mcp"
      "github.com/spf13/cobra"
  )

  func NewAppsCmd() *cobra.Command {
      cmd := &cobra.Command{
          Use:   "apps",
          Short: "Databricks apps development tools",
          Long:  "Tools for developing and managing Databricks applications, including MCP servers for AI agents.",
          GroupID: "development",
      }

      cmd.AddCommand(mcp.NewMcpCmd())

      return cmd
  }
  ```
- [ ] Test compilation
  ```bash
  go build ./cmd/apps
  ```

**Verification**: Should compile without errors

### M16: Create MCP Root Command

- [ ] Create `cmd/apps/mcp/mcp.go`
  ```go
  package mcp

  import "github.com/spf13/cobra"

  func NewMcpCmd() *cobra.Command {
      cmd := &cobra.Command{
          Use:   "mcp",
          Short: "Model Context Protocol server for AI agents",
          Long:  "Start and manage an MCP server that provides Databricks tools to AI agents via the Model Context Protocol.",
      }

      cmd.AddCommand(newStartCmd())
      cmd.AddCommand(newCheckCmd())

      return cmd
  }
  ```
- [ ] Test compilation
  ```bash
  go build ./cmd/apps/mcp
  ```

**Verification**: Should compile without errors

### M17: Create Start Command

- [ ] Create `cmd/apps/mcp/start.go`
  ```go
  package mcp

  import (
      "github.com/databricks/cli/libs/cmdctx"
      "github.com/databricks/cli/libs/log"
      "github.com/databricks/cli/libs/mcp"
      "github.com/databricks/cli/libs/mcp/server"
      "github.com/spf13/cobra"
  )

  func newStartCmd() *cobra.Command {
      var warehouseID string
      var allowDeployment bool
      var dockerImage string
      var useDagger bool

      cmd := &cobra.Command{
          Use:   "start",
          Short: "Start the MCP server",
          Long:  "Start the Model Context Protocol server to provide Databricks tools to AI agents.",
          RunE: func(cmd *cobra.Command, args []string) error {
              ctx := cmd.Context()

              // Get Databricks client from context
              w := cmdctx.GetWorkspaceClient(ctx)

              // Build MCP config from flags
              cfg := &mcp.Config{
                  AllowDeployment:    allowDeployment,
                  WithWorkspaceTools: true,
                  WarehouseID:        warehouseID,
                  DatabricksHost:     w.Config.Host,
                  IoConfig: &mcp.IoConfig{
                      Validation: &mcp.ValidationConfig{
                          DockerImage: dockerImage,
                          UseDagger:   useDagger,
                      },
                  },
              }

              log.Infof(ctx, "Starting MCP server")

              // Create and start server
              srv := server.NewServer(cfg, w)
              return srv.Start(ctx)
          },
      }

      // Define flags
      cmd.Flags().StringVar(&warehouseID, "warehouse-id", "", "Databricks SQL Warehouse ID")
      cmd.Flags().BoolVar(&allowDeployment, "allow-deployment", false, "Enable deployment tools")
      cmd.Flags().StringVar(&dockerImage, "docker-image", "node:20-alpine", "Docker image for validation")
      cmd.Flags().BoolVar(&useDagger, "use-dagger", true, "Use Dagger for containerized validation")

      return cmd
  }
  ```
- [ ] Test compilation
  ```bash
  go build ./cmd/apps/mcp
  ```

**Verification**: Should compile without errors

### M18: Create Check Command

- [ ] Create `cmd/apps/mcp/check.go`
  ```go
  package mcp

  import (
      "github.com/databricks/cli/libs/cmdctx"
      "github.com/databricks/cli/libs/cmdio"
      "github.com/databricks/cli/libs/log"
      "github.com/spf13/cobra"
  )

  func newCheckCmd() *cobra.Command {
      cmd := &cobra.Command{
          Use:   "check",
          Short: "Check MCP server environment",
          Long:  "Verify that the environment is correctly configured for running the MCP server.",
          RunE: func(cmd *cobra.Command, args []string) error {
              ctx := cmd.Context()

              log.Info(ctx, "Checking MCP server environment")

              // Check Databricks authentication
              w := cmdctx.GetWorkspaceClient(ctx)
              me, err := w.CurrentUser.Me(ctx)
              if err != nil {
                  return err
              }

              cmdio.LogString(ctx, "✓ Databricks authentication: OK")
              cmdio.LogString(ctx, "  User: "+me.UserName)
              cmdio.LogString(ctx, "  Host: "+w.Config.Host)

              // Check MCP SDK
              cmdio.LogString(ctx, "✓ MCP SDK: OK")

              // Check Dagger (optional)
              cmdio.LogString(ctx, "✓ Dagger SDK: OK (optional)")

              cmdio.LogString(ctx, "\nEnvironment is ready for MCP server")

              return nil
          },
      }

      return cmd
  }
  ```
- [ ] Test compilation
  ```bash
  go build ./cmd/apps/mcp
  ```

**Verification**: Should compile without errors

**Commit Point**:
```bash
git add cmd/apps
git commit -m "Implement MCP commands in CLI

Created:
- cmd/apps/apps.go: Root apps command
- cmd/apps/mcp/mcp.go: MCP subcommand root
- cmd/apps/mcp/start.go: Start MCP server
- cmd/apps/mcp/check.go: Environment check

Commands integrate with CLI logging, config, and Databricks client.

For parity-40 (Phase 2: Code Structure Migration)"
```

**Rollback**: `git reset --hard HEAD~1`

## Phase 2G: Command Registration

### M19: Register Apps Command

- [ ] Open `cmd/cmd.go`
- [ ] Add import:
  ```go
  import "github.com/databricks/cli/cmd/apps"
  ```
- [ ] Add command registration in `New()`:
  ```go
  func New() *cobra.Command {
      // ... existing code ...

      // Add apps command
      cmd.AddCommand(apps.NewAppsCmd())

      return cmd
  }
  ```
- [ ] Test compilation
  ```bash
  go build ./cmd/databricks
  ```
- [ ] Test help output
  ```bash
  ./bin/databricks apps --help
  ./bin/databricks apps mcp --help
  ./bin/databricks apps mcp start --help
  ```

**Verification**: Commands should appear in help output

**Commit Point**:
```bash
git add cmd/cmd.go
git commit -m "Register apps command in CLI

Added apps command group with MCP subcommand.

Usage:
  databricks apps mcp start
  databricks apps mcp check

For parity-40 (Phase 2: Code Structure Migration)"
```

**Rollback**: `git reset --hard HEAD~1`

## Phase 2H: Testing

### M20: Run Unit Tests

- [ ] Run all MCP unit tests
  ```bash
  go test ./libs/mcp/... -v
  ```
- [ ] Review and fix any test failures
- [ ] Run with race detector
  ```bash
  go test ./libs/mcp/... -race
  ```

**Verification**: All tests should pass

### M21: Manual Integration Test

- [ ] Build CLI
  ```bash
  make build
  ```
- [ ] Run environment check
  ```bash
  ./bin/databricks apps mcp check
  ```
- [ ] Start MCP server (in separate terminal)
  ```bash
  ./bin/databricks apps mcp start --warehouse-id <YOUR_WAREHOUSE_ID>
  ```
- [ ] Test with Claude desktop (if available)
  - Update Claude config to use new binary
  - Test basic MCP operations

**Verification**: Server should start without errors

**Commit Point**:
```bash
git add .
git commit -m "Complete Phase 2: Code Structure Migration

Migration complete:
- All library code migrated to libs/mcp/
- Import paths updated
- Configuration adapted to CLI patterns
- Logging integrated with libs/log
- Commands implemented and registered
- Unit tests passing

Ready for Phase 3: Advanced Integration

For parity-40 (Phase 2: Code Structure Migration)"
```

## Phase 2I: Documentation

### M22: Update README

- [ ] Create `libs/mcp/README.md`
  ```markdown
  # MCP Server for Databricks CLI

  This package provides Model Context Protocol (MCP) server functionality
  integrated into the Databricks CLI.

  ## Usage

  Start the MCP server:
  ```
  databricks apps mcp start --warehouse-id <WAREHOUSE_ID>
  ```

  Check environment:
  ```
  databricks apps mcp check
  ```

  ## Architecture

  See `/plans/` directory for detailed documentation:
  - `directory-structure-mapping.md`: Code organization
  - `integration-points.md`: CLI integration details
  - `dependency-analysis.md`: Dependencies and versions
  ```
- [ ] Copy plans directory
  ```bash
  cp -r /Users/fabian.jakobs/Workspaces/app-mcp/plans libs/mcp/docs/
  ```

**Commit Point**:
```bash
git add libs/mcp/README.md libs/mcp/docs
git commit -m "Add MCP documentation

Added README and planning documents for reference.

For parity-40 (Phase 2: Code Structure Migration)"
```

## Summary

### Completed Tasks

- [ ] S1: Branch verification
- [ ] S2: Dependency installation
- [ ] M1-M4: Directory structure creation
- [ ] M5-M11: Library code copying
- [ ] M12: Import path updates
- [ ] M13: Configuration integration
- [ ] M14: Logging integration
- [ ] M15-M18: Command implementation
- [ ] M19: Command registration
- [ ] M20-M21: Testing
- [ ] M22: Documentation

### Success Criteria

- [ ] All files migrated from go-mcp
- [ ] All import paths updated
- [ ] No references to pkg/config or pkg/logging
- [ ] Commands registered and functional
- [ ] Unit tests passing
- [ ] Manual integration test successful
- [ ] Documentation in place

### Final Verification

```bash
# Build succeeds
make build

# Tests pass
go test ./libs/mcp/...

# Commands work
./bin/databricks apps mcp --help
./bin/databricks apps mcp check

# No old imports
! grep -r "github.com/databricks/go-mcp" libs/mcp cmd/apps
```

## Next Steps

After Phase 2 completion, proceed to:
- **Phase 3**: Advanced integration (context threading, error handling)
- **Phase 4**: Testing and validation
- **Phase 5**: Documentation and release preparation
