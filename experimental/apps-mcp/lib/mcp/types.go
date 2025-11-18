// Package mcp provides a minimal implementation of the Model Context Protocol for stdio-based servers.
package mcp

import (
	"context"
)

// Implementation represents server or client implementation details.
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Tool represents an MCP tool definition.
type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

// CallToolRequest represents a request to call a tool.
type CallToolRequest struct {
	Params CallToolParams
}

// Content represents content in a tool result.
type Content interface {
	isContent()
}

// TextContent represents text content.
type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (t *TextContent) isContent() {}

// ToolHandler is a low-level handler for tool calls.
type ToolHandler func(context.Context, *CallToolRequest) (*CallToolResult, error)

// ToolHandlerFor is a typed handler for tool calls with automatic marshaling.
type ToolHandlerFor[In, Out any] func(context.Context, *CallToolRequest, In) (*CallToolResult, Out, error)
