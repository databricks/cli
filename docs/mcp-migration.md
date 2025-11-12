# MCP Integration Migration

## Overview

This document describes the migration of the standalone go-mcp CLI into the Databricks CLI as the `databricks apps mcp` subcommand.

## Source

- **Original Repository**: github.com/appdotbuild/go-mcp
- **Commits Migrated**: 21 commits (86db47a through 441f736)
- **Migration Date**: 2025-11-08
- **Branch**: apps-mcp

## Architecture

### Command Structure

```
databricks apps mcp
├── start        # Start MCP server
└── check        # Environment check
```

### Code Organization

```
cli/
├── cmd/apps/mcp/       # Command implementations
└── libs/mcp/           # MCP library code
    ├── server/         # MCP server implementation
    ├── providers/      # Tool providers (databricks, io, workspace, deployment)
    ├── sandbox/        # Execution abstraction (local, dagger)
    ├── trajectory/     # JSONL-based history logging
    ├── templates/      # Template abstraction
    └── session/        # Session management
```

## Key Changes

### Infrastructure Integration

1. **Logging**: Migrated from custom `pkg/logging` to `libs/log`
2. **Configuration**: Replaced Viper with cobra flags
3. **Session Management**: Context-based instead of singleton
4. **Error Handling**: Aligned with CLI patterns

### Import Path Changes

```
github.com/appdotbuild/go-mcp/pkg/* → github.com/databricks/cli/libs/mcp/*
```

### Dependencies Added

- `github.com/modelcontextprotocol/go-sdk@v1.1.0`
- `github.com/zeebo/blake3@v0.2.4`

## Testing

All tests migrated and passing:
- Unit tests: >80% coverage
- Integration tests: All passing
- Race detector: Clean

## Original Commits

The migration preserves the intent of all 21 original commits:

1. Foundation (commits 1-5): Sandbox, MCP server, providers
2. Features (commits 6-10): Validation, deployment, configuration
3. Polish (commits 11-15): Error handling, security improvements
4. Advanced (commits 16-21): Trajectory tracking, code quality

See commit messages for original SHAs and details.

## References

- MCP Protocol: https://modelcontextprotocol.io/
- Original Repository: github.com/appdotbuild/go-mcp
- Migration Plan: plans/databricks-cli-migration-master.md
