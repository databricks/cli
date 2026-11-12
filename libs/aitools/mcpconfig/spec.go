// Package mcpconfig describes where a coding agent keeps its user-scope MCP
// server entries. It is separate from libs/aitools/agents so that the agent
// registry can carry the specs without importing the MCP packages.
package mcpconfig

import (
	"context"
	"slices"
	"strings"
)

// Format is the on-disk format of an agent's MCP config file.
type Format int

const (
	FormatJSON Format = iota
	FormatTOML
)

// Spec describes an agent's user-scope MCP config file.
type Spec struct {
	// Path returns the absolute path of the config file.
	Path func(ctx context.Context) (string, error)
	// Format is the format of the config file.
	Format Format
	// Root is the dot-separated key path of the table that holds server entries,
	// such as "mcpServers" or "mcp.servers".
	Root string
	// Entry builds the entry for a stdio server that runs cmd with args and env.
	// env is empty when there are no variables to pin.
	Entry func(cmd string, args []string, env map[string]string) map[string]any
	// IsOurs reports whether an entry was written by this CLI. It receives entries
	// as they were decoded from the file.
	IsOurs func(entry map[string]any) bool
}

// Argv returns the command line of a stdio entry as argv[0] and its arguments.
// It accepts the `command` string plus `args` shape used by most agents and the
// single `command` array that OpenCode uses. Entries decoded from JSON or TOML
// carry arguments as []any, so that is the only list type it accepts.
func Argv(entry map[string]any) ([]string, bool) {
	switch command := entry["command"].(type) {
	case string:
		args, _ := entry["args"].([]any)
		argv := []string{command}
		for _, arg := range args {
			s, ok := arg.(string)
			if !ok {
				return nil, false
			}
			argv = append(argv, s)
		}
		return argv, true
	case []any:
		argv := make([]string, 0, len(command))
		for _, part := range command {
			s, ok := part.(string)
			if !ok {
				return nil, false
			}
			argv = append(argv, s)
		}
		if len(argv) == 0 {
			return nil, false
		}
		return argv, true
	default:
		return nil, false
	}
}

// IsDatabricksEntry reports whether entry runs `databricks aitools mcp serve`,
// which is how we recognize entries we wrote. It is the IsOurs of every agent.
func IsDatabricksEntry(entry map[string]any) bool {
	argv, ok := Argv(entry)
	if !ok {
		return false
	}
	args := argv[1:]
	return IsBinary(argv[0], "databricks", "databricks.exe") &&
		len(args) >= 3 && args[0] == "aitools" && args[1] == "mcp" && args[2] == "serve"
}

// IsBinary reports whether cmd, a bare name or a path, has one of the given
// base names. Both separators are accepted so that a Windows path is recognized
// on any host.
func IsBinary(cmd string, names ...string) bool {
	base := cmd[strings.LastIndexAny(cmd, `/\`)+1:]
	return slices.Contains(names, base)
}
