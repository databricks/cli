# Go MCP Server Implementation Plan

## Overview

Create a Go-based MCP server with similar functionality to `edda_mcp`, focusing on Databricks integration and sandboxed execution. The implementation will use idiomatic Go patterns rather than directly translating Rust code.

## High-Level Architecture

```
go-mcp/
├── cmd/
│   └── go-mcp/           # Main server binary
├── pkg/
│   ├── mcp/              # MCP protocol implementation
│   ├── sandbox/          # Execution sandbox abstraction
│   ├── providers/        # Tool providers (Databricks, IO, Workspace)
│   ├── config/           # Configuration management
│   └── session/          # Session context
├── internal/
│   └── templates/        # Embedded templates
└── test/
    └── integration/      # Integration tests
```

## Technology Stack

- **MCP Protocol**: Use Go MCP SDK or implement from scratch
- **Databricks**: [Databricks SDK for Go](https://github.com/databricks/databricks-sdk-go)
- **Sandbox**: Interface-based abstraction with local filesystem and future Dagger support
- **Testing**: Standard Go testing with table-driven tests
- **Configuration**: JSON config files with environment variable overrides

## High-Level Implementation Steps

### Step 1: Project Setup and Foundation
- Initialize Go module structure
- Set up core dependencies
- Implement configuration system
- Create logging and error handling patterns

### Step 2: Sandbox Abstraction
- Define sandbox interface
- Implement local filesystem sandbox
- Add unit tests for sandbox operations
- Prepare Dagger backend interface (stub for future)

### Step 3: MCP Protocol Implementation
- Implement MCP server handler
- Create tool registration system
- Build request/response handling
- Add session management

### Step 4: Databricks Provider
- Integrate Databricks SDK
- Implement catalog/schema/table operations
- Add SQL execution with polling
- Create provider tool definitions

### Step 5: I/O Provider
- Implement template management
- Create scaffold operation
- Add validation in sandbox
- Embed default templates

### Step 6: Workspace Provider
- Implement file operations (read, write, edit)
- Add bash execution
- Create grep/glob utilities
- Implement path validation

### Step 7: Integration and Polish
- Combine all providers
- Add health checks
- Implement graceful shutdown
- Create integration tests

### Step 8: CLI and Deployment
- Build CLI with flags
- Add version checking
- Create installation script
- Write documentation

## Success Criteria

- [ ] MCP server starts and responds to protocol handshake
- [ ] Databricks tools work with real Databricks workspace
- [ ] Sandbox executes commands and manages files
- [ ] Templates can be scaffolded and validated
- [ ] Workspace tools operate within project boundaries
- [ ] All unit tests pass with >80% coverage
- [ ] Integration tests verify end-to-end workflows

## Dependencies

### Required
- Go 1.21+
- Docker (for sandbox operations)
- Databricks workspace (for integration testing)

### Go Packages
- `github.com/databricks/databricks-sdk-go` - Databricks client
- MCP SDK (TBD - may need custom implementation)
- `github.com/spf13/cobra` - CLI framework (optional)
- `github.com/spf13/viper` - Configuration management

## Timeline Estimate

- Steps 1-2: Foundation and Sandbox (2-3 days)
- Steps 3-4: MCP and Databricks (3-4 days)
- Steps 5-6: I/O and Workspace (2-3 days)
- Steps 7-8: Integration and Polish (2-3 days)

**Total: 9-13 days** for a single developer

## Notes

- Focus on clean interfaces to enable future Dagger sandbox backend
- Keep provider implementations independent for testability
- Use context.Context throughout for cancellation and timeouts
- Follow Go conventions: accept interfaces, return structs
- Defer Google Sheets integration (out of scope)
