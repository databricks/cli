# Master Plan: Migrate go-mcp into Databricks CLI

**bd Issue**: `parity-38` (epic)
**Status**: Open | **Priority**: P0
**Subtasks**: parity-39 (Phase 1), parity-40 (Phase 2), parity-41 (Phase 3), parity-42 (Phase 4), parity-43 (Phase 5)

## Overview

This plan outlines the migration of the standalone go-mcp CLI into the Databricks CLI as a `databricks apps mcp` subcommand. The migration will maintain full git history and integrate with the Databricks CLI's infrastructure.

## Goals

1. **Integration**: Expose go-mcp functionality as `databricks apps mcp` subcommand
2. **History Preservation**: Replay all 21 commits from go-mcp repo into Databricks CLI
3. **Infrastructure Alignment**: Integrate with Databricks CLI's build, test, logging, and config systems
4. **Code Quality**: Maintain or improve code quality and test coverage
5. **Documentation**: Ensure comprehensive documentation for the new subcommand

## Current State

### go-mcp Repository
- **Location**: `/Users/fabian.jakobs/Workspaces/app-mcp`
- **Commits**: 21 commits of development history
- **Structure**:
  - `cmd/go-mcp/`: CLI entry point
  - `pkg/`: Core packages (config, providers, mcp, sandbox, trajectory, templates, logging, session, errors)
  - Independent infrastructure (Viper config, custom logging, session management)

### Databricks CLI Repository
- **Location**: `/Users/fabian.jakobs/Workspaces/cli`
- **Current Branch**: `main`
- **Target Branch**: `apps-mcp` (to be created)
- **Structure**: Commands under `cmd/`, shared libs under `libs/`
- **Infrastructure**: Cobra commands, libs/log logging, Makefile build system, gotestsum testing

## Migration Phases

### Phase 1: Repository Setup & Analysis
**Duration**: 1-2 hours
**Deliverables**:
- New `apps-mcp` branch in Databricks CLI repo
- Commit history analysis and mapping
- Directory structure plan
- Dependency analysis

**Key Tasks**:
- Create and checkout `apps-mcp` branch from latest `main`
- Analyze all 21 commits for logical grouping
- Map go-mcp dependencies to Databricks CLI equivalents
- Design target directory structure under `cmd/apps/`

### Phase 2: Code Structure Migration
**Duration**: 2-3 hours
**Deliverables**:
- go-mcp code migrated to `cmd/apps/mcp/` in Databricks CLI
- Package paths updated to Databricks CLI module
- Initial command registration

**Key Tasks**:
- Copy pkg/ packages to appropriate location in Databricks CLI
- Move cmd/go-mcp to cmd/apps/mcp
- Update all import paths from `github.com/appdotbuild/go-mcp` to `github.com/databricks/cli`
- Create initial cobra command structure for `apps mcp`
- Register new command in `cmd/cmd.go`

### Phase 3: Infrastructure Integration
**Duration**: 3-4 hours
**Deliverables**:
- Logging migrated to `libs/log`
- Configuration integrated with Databricks CLI config system
- Session management aligned
- Error handling standardized

**Key Tasks**:
- Replace custom logging with `libs/log`
- Integrate configuration with Databricks CLI patterns
- Adapt session management to CLI context patterns
- Standardize error handling
- Update MCP server initialization to use CLI infrastructure

### Phase 4: Testing & Build Integration
**Duration**: 2-3 hours
**Deliverables**:
- Tests adapted to Databricks CLI patterns
- Build system integration
- CI/CD compatibility
- Test coverage maintained or improved

**Key Tasks**:
- Adapt test files to Databricks CLI testing patterns
- Update Makefile to include new packages
- Ensure gotestsum compatibility
- Run full test suite and fix any issues
- Verify build process

### Phase 5: Documentation & Finalization
**Duration**: 1-2 hours
**Deliverables**:
- Updated documentation
- Command help text
- Migration guide
- Code review ready PR

**Key Tasks**:
- Update README with new command structure
- Add command documentation
- Create migration guide for users
- Final code cleanup
- Prepare PR description with history explanation

## Commit Replay Strategy

The 21 commits from go-mcp will be replayed in the Databricks CLI repo in logical groups:

1. **Foundation commits** (commits 1-5): Initial sandbox, MCP server setup
2. **Provider commits** (commits 6-10): Databricks, IO, workspace providers
3. **Feature commits** (commits 11-15): Validation, deployment, trajectory tracking
4. **Polish commits** (commits 16-21): Error handling, security, code quality

Each replay commit will:
- Reference the original commit SHA in the message
- Maintain the logical intent of the original change
- Be adapted to the Databricks CLI structure

## Technical Considerations

### Import Path Changes
```
github.com/appdotbuild/go-mcp → github.com/databricks/cli
```

### Logging Migration
```go
// Before (go-mcp)
import "github.com/appdotbuild/go-mcp/pkg/logging"
logger := logging.NewLogger(...)

// After (Databricks CLI)
import "github.com/databricks/cli/libs/log"
logger := log.GetLogger(ctx)
```

### Configuration Migration
```go
// Before (go-mcp)
import "github.com/spf13/viper"
cfg := config.LoadConfig()

// After (Databricks CLI)
// Use CLI's root command context and flags
```

### Command Structure
```
databricks                           # Root command (existing)
└── apps                            # New top-level command
    └── mcp                         # MCP server command
        ├── start                   # Start MCP server (main command)
        ├── check                   # Environment check
        └── config                  # Configuration management
            ├── show                # Show config
            └── validate            # Validate config
```

## Dependencies

### New Dependencies to Add
- `github.com/modelcontextprotocol/go-sdk` (v1.1.0)
- `github.com/zeebo/blake3` (v0.2.4)

### Existing Dependencies to Leverage
- `github.com/databricks/databricks-sdk-go` (already in both)
- `github.com/spf13/cobra` (already in both)

## Risk Mitigation

1. **Breaking Changes**: Work in feature branch until fully tested
2. **History Loss**: Document original commit SHAs in new commits
3. **Integration Issues**: Incremental migration with testing at each phase
4. **Dependency Conflicts**: Carefully review and update dependencies
5. **Test Coverage**: Maintain or improve test coverage throughout migration

## Success Criteria

- [ ] All go-mcp functionality accessible via `databricks apps mcp`
- [ ] Full test suite passes
- [ ] Build system works correctly
- [ ] Documentation is complete and clear
- [ ] Code review approved
- [ ] No regression in existing Databricks CLI functionality
- [ ] Git history properly preserved and documented

## Timeline

**Total Estimated Time**: 9-14 hours over 2-3 days

- **Day 1**: Phases 1-2 (Repository setup and code migration)
- **Day 2**: Phase 3 (Infrastructure integration)
- **Day 3**: Phases 4-5 (Testing and documentation)

## Next Steps

1. Review and approve this plan
2. Begin Phase 1: Repository Setup & Analysis
3. Create detailed task breakdown in bd
4. Execute phases sequentially with checkpoints

## References

- go-mcp repository: `/Users/fabian.jakobs/Workspaces/app-mcp`
- Databricks CLI repository: `/Users/fabian.jakobs/Workspaces/cli`
- go-mcp documentation: `CLAUDE.md`, `AGENTS.md`
- Original commits: 21 commits (441f736 to 86db47a)
