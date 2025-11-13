# MCP Server for Databricks CLI

This package provides Model Context Protocol (MCP) server functionality
integrated into the Databricks CLI.

## Usage

Start the MCP server:
```bash
databricks apps mcp start --warehouse-id <WAREHOUSE_ID>
```

Check environment:
```bash
databricks apps mcp check
```

## Architecture

See `docs/` directory for detailed documentation:
- `directory-structure-mapping.md`: Code organization
- `integration-points.md`: CLI integration details
- `dependency-analysis.md`: Dependencies and versions
- `phase-2-migration-checklist.md`: Migration guide

## Implementation Status

Phase 2 complete:
- ✅ All library code migrated to libs/mcp/
- ✅ Import paths updated
- ✅ Configuration integrated with CLI patterns
- ✅ Logging integrated with libs/log
- ✅ Commands implemented and registered
- ✅ Build succeeds

## Development

Build:
```bash
make build
```

Test:
```bash
go test ./libs/mcp/...
```
