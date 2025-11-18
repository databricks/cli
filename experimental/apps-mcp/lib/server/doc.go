/*
Package server implements the Model Context Protocol (MCP) server.

The MCP server provides AI agents with tools to interact with Databricks.
It uses the official MCP Go SDK and supports stdio transport.

Usage:

	ctx := context.Background()
	cfg := &config.Config{
		WarehouseID: "abc123",
	}
	server := server.NewServer(cfg, ctx)
	err := server.Run(ctx)

Architecture:

The server uses a provider-based architecture where each provider
registers its tools independently. Providers include:

- Databricks: Query catalogs, schemas, tables, execute SQL
- IO: Scaffold and validate TypeScript applications
- Workspace: File operations in project directories
- Deployment: Deploy applications (optional)

Session Management:

Sessions track state across tool calls including:
- Working directory (set by scaffold, used by workspace tools)
- Metrics and telemetry
- Trajectory logging (JSONL history)

Sandbox:

Tools execute in a sandbox abstraction that can be:
- Local: Direct filesystem and shell access
- Dagger: Not implemented (stub only)
*/
package server
