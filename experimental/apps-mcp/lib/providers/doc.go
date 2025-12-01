/*
Package providers contains MCP tool providers.

Each provider implements a set of related tools:

- databricks: Databricks API integration
- io: Project scaffolding and validation
- clitools: CLI exploration and invocation
- deployment: Application deployment (optional)

Provider Interface:

	type Provider interface {
		RegisterTools(server *mcp.Server) error
	}

Providers are registered with the MCP server during initialization
and their tools become available to AI agents.
*/
package providers
