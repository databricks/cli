package mcp

import (
	"context"
	"errors"
	"strings"
)

// WorkspaceRoot represents a workspace root directory.
type WorkspaceRoot struct {
	URI  string
	Name string
	Path string // Extracted file path from URI
}

// GetWorkspaceRoots fetches the workspace roots from the agent via roots/list.
// This is called on-demand by tools that need workspace context.
// Results are cached after the first call.
func (s *MCPServer) GetWorkspaceRoots(ctx context.Context) ([]WorkspaceRoot, error) {
	// Check if we already have roots cached
	if s.cachedRoots != nil {
		return s.cachedRoots, nil
	}

	// Note: This is a simplified implementation that doesn't actually make
	// the roots/list call because it requires more complex bidirectional
	// communication setup. For now, we'll return an empty list and tools
	// should rely on explicit project_path parameters.

	// TODO: Implement bidirectional communication to actually call roots/list
	// This would require:
	// 1. Sending a roots/list request to stdout
	// 2. Reading the response from stdin (while also handling other requests)
	// 3. Thread-safe coordination between request handling and tool execution

	s.cachedRoots = []WorkspaceRoot{}
	return s.cachedRoots, nil
}

// GetWorkspaceRoot returns the single workspace root, or an error if there are zero or multiple.
func (s *MCPServer) GetWorkspaceRoot(ctx context.Context) (*WorkspaceRoot, error) {
	roots, err := s.GetWorkspaceRoots(ctx)
	if err != nil {
		return nil, err
	}

	if len(roots) == 0 {
		return nil, errors.New("no workspace roots found - agent did not provide workspace context")
	}

	if len(roots) > 1 {
		return nil, errors.New("multiple workspace roots found - expected exactly one")
	}

	return &roots[0], nil
}

// ExtractPathFromURI extracts the file path from a file:// URI.
func ExtractPathFromURI(uri string) string {
	// Remove file:// prefix if present
	if strings.HasPrefix(uri, "file://") {
		return uri[7:]
	}
	return uri
}
