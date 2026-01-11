// Package mcp provides Model Context Protocol (MCP) server functionality
// integrated into the Databricks CLI.
package mcp

// Config holds MCP server configuration.
// Configuration is populated from CLI flags and Databricks client context.
type Config struct{}

// DefaultConfig returns a Config with sensible default values.
func DefaultConfig() *Config {
	return &Config{}
}
