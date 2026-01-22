/*
Package providers contains MCP tool providers.

The clitools provider implements CLI-based tools for Databricks integration:

- explore: Discover workspace resources and get workflow recommendations
- invoke_databricks_cli: Execute Databricks CLI commands
- databricks_configure_auth: Configure workspace authentication

Provider Interface:

	type Provider interface {
		RegisterTools(server *mcp.Server) error
	}

Providers are registered with the MCP server during initialization
and their tools become available to AI agents.

The CLI-based approach leverages existing bundle commands for app
scaffolding, validation, and deployment rather than duplicating API logic.
*/
package providers
