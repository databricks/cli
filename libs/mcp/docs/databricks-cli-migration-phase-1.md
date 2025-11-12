# Phase 1: Repository Setup & Analysis

**bd Issue**: `parity-39` (task)
**Status**: Open | **Priority**: P0
**Depends on**: parity-38 (Master Epic)
**Blocks**: parity-40 (Phase 2)

## Overview

This phase establishes the foundation for the migration by setting up the target branch, analyzing the commit history, and planning the directory structure.

## Duration

1-2 hours

## Prerequisites

- Access to both repositories
- Understanding of git history manipulation
- Databricks CLI codebase familiarity

## Tasks

### 1.1 Create Target Branch

**Objective**: Set up the working branch in Databricks CLI repo

**Steps**:
```bash
cd /Users/fabian.jakobs/Workspaces/cli
git checkout main
git pull origin main
git checkout -b apps-mcp
git push -u origin apps-mcp
```

**Verification**:
- Branch exists locally and remotely
- Based on latest main
- No uncommitted changes

### 1.2 Analyze Commit History

**Objective**: Document all commits from go-mcp for replay

**Steps**:
```bash
cd /Users/fabian.jakobs/Workspaces/app-mcp
git log --oneline --all > /tmp/go-mcp-commits.txt
git log --format="%H|%s|%an|%ae|%ad" --all > /tmp/go-mcp-commits-detailed.txt
```

**Commit Groups** (21 commits total):

1. **Foundation** (commits 1-5):
   - 86db47a: implement sandbox
   - 0f3a1d2: add mcp server
   - 3280b53: implement databricks provider
   - 7cf04b8: add scaffolding
   - 4d94a9b: implement workspace tools

2. **Feature Development** (commits 6-10):
   - 6dff19e: finishing touches
   - ad0c4e6: implement validation
   - 02ff240: implement deployment tool
   - 90d4e9f: remove "required_providers"
   - 1b665b9: add configuration for Databricks host

3. **Polish & Features** (commits 11-15):
   - dc1ad8e: initialization message appears on first tool call
   - 39e3119: update descriptions
   - f977f7c: Update .gitignore with comprehensive rules
   - c2a770a: use secure session id generators
   - 61cac00: fix error handling

4. **Advanced Features** (commits 16-21):
   - ae6a31b: Add trajectory tracking for MCP tool calls
   - e561b61: add AI agent plans
   - 0f841ec: Implement parity-8 to parity-13: Test coverage and path validation refactoring
   - b82f88b: code cleanups
   - 441f736: Implement parity tasks 7-37: Code quality improvements

**Deliverable**: Commit replay order document

### 1.3 Directory Structure Planning

**Objective**: Design target structure in Databricks CLI

**Current go-mcp Structure**:
```
app-mcp/
├── cmd/go-mcp/          # CLI entry point
├── pkg/
│   ├── config/          # Configuration
│   ├── mcp/             # MCP server
│   ├── providers/       # Tool providers
│   │   ├── databricks/
│   │   ├── io/
│   │   ├── workspace/
│   │   └── deployment/
│   ├── sandbox/         # Execution abstraction
│   │   ├── local/
│   │   └── dagger/
│   ├── trajectory/      # History logging
│   ├── templates/       # Template abstraction
│   ├── logging/         # Logging system
│   ├── session/         # Session management
│   ├── errors/          # Error types
│   ├── pathutil/        # Path utilities
│   └── fileutil/        # File utilities
└── internal/templates/  # Embedded templates
```

**Target Databricks CLI Structure**:
```
cli/
├── cmd/
│   └── apps/                    # NEW: Apps command group
│       ├── apps.go             # NEW: Apps root command
│       └── mcp/                # NEW: MCP subcommand
│           ├── mcp.go          # MCP command entry
│           ├── start.go        # Start MCP server
│           ├── check.go        # Environment check
│           └── config.go       # Config commands
├── libs/
│   └── mcp/                    # NEW: MCP library code
│       ├── server/             # MCP server implementation
│       ├── providers/          # Tool providers
│       │   ├── databricks/
│       │   ├── io/
│       │   ├── workspace/
│       │   └── deployment/
│       ├── sandbox/            # Execution abstraction
│       │   ├── local/
│       │   └── dagger/
│       ├── trajectory/         # History logging
│       ├── templates/          # Template abstraction
│       ├── session/            # Session management
│       ├── pathutil/           # Path utilities
│       └── fileutil/           # File utilities
└── internal/
    └── mcp/
        └── templates/          # Embedded templates
```

**Design Decisions**:
- Place command logic in `cmd/apps/mcp/`
- Place library code in `libs/mcp/` (follows CLI pattern)
- Reuse existing `libs/log/` instead of custom logging
- Integrate with existing config system in `libs/cmdctx/`
- Remove `pkg/config/` and `pkg/logging/` (use CLI equivalents)

**Deliverable**: Directory structure diagram and migration map

### 1.4 Dependency Analysis

**Objective**: Identify dependency differences and conflicts

**go-mcp Dependencies** (unique):
```
github.com/modelcontextprotocol/go-sdk v1.1.0
github.com/spf13/viper v1.21.0              # May not be needed
github.com/zeebo/blake3 v0.2.4
```

**Databricks CLI Dependencies** (relevant):
```
github.com/databricks/databricks-sdk-go v0.89.0  # Already present
github.com/spf13/cobra v1.10.1                   # Already present
github.com/spf13/pflag v1.0.10                   # Already present
```

**Dependencies to Add**:
- `github.com/modelcontextprotocol/go-sdk v1.1.0`
- `github.com/zeebo/blake3 v0.2.4`

**Dependencies to Update** (if needed):
- `github.com/databricks/databricks-sdk-go` (0.89.0 → 0.90.0)

**Dependencies to Remove** (from go-mcp context):
- `github.com/spf13/viper` (use CLI's config pattern instead)

**Deliverable**: Dependency update plan

### 1.5 Integration Points Documentation

**Objective**: Document how go-mcp will integrate with CLI systems

**Integration Matrix**:

| Component | go-mcp Current | Databricks CLI | Integration Strategy |
|-----------|---------------|----------------|---------------------|
| CLI Framework | Cobra | Cobra | Direct compatibility |
| Logging | Custom pkg/logging | libs/log | Replace with libs/log |
| Configuration | Viper | Context-based | Adapt to CLI patterns |
| Session | Custom pkg/session | Context-based | Merge concepts |
| Error Handling | pkg/errors | Standard errors | Adapt to CLI patterns |
| Context | Basic context.Context | cmdctx package | Use cmdctx |
| Testing | Standard go test | gotestsum | Adapt test patterns |
| Build | Custom Makefile | CLI Makefile | Integrate targets |

**Key Integration Challenges**:
1. **Logging**: Migrate from custom logging to libs/log
2. **Configuration**: Adapt Viper-based config to CLI's flag-based config
3. **Session Management**: Integrate session state with CLI context
4. **Command Structure**: Fit `mcp start` pattern into CLI conventions

**Deliverable**: Integration strategy document

### 1.6 Create Migration Checklist

**Objective**: Detailed checklist for Phase 2 execution

**Code Migration Checklist**:
- [ ] Create `cmd/apps/` directory
- [ ] Create `cmd/apps/apps.go` with root command
- [ ] Create `cmd/apps/mcp/` directory structure
- [ ] Create `libs/mcp/` directory structure
- [ ] Copy and adapt command files
- [ ] Copy and adapt library files
- [ ] Update all import paths
- [ ] Register apps command in cmd/cmd.go
- [ ] Add dependencies to go.mod
- [ ] Initial build verification

**Deliverable**: Detailed migration checklist

## Deliverables

1. ✅ `apps-mcp` branch created and pushed
2. ✅ Commit history analysis document
3. ✅ Directory structure design
4. ✅ Dependency analysis and update plan
5. ✅ Integration points documentation
6. ✅ Detailed migration checklist for Phase 2

## Verification Steps

- [ ] Branch exists: `git branch | grep apps-mcp`
- [ ] Branch is based on latest main: `git log --oneline -1`
- [ ] All 21 commits documented with SHAs
- [ ] Directory structure approved
- [ ] Dependency conflicts identified and resolved
- [ ] Integration strategy clear

## Next Phase

**Phase 2: Code Structure Migration** - Copy and adapt code to new structure
