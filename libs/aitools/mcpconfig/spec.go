// Package mcpconfig describes where a coding agent keeps its user-scope MCP
// server entries. It is separate from libs/aitools/agents so that the agent
// registry can carry the specs without importing the MCP packages.
package mcpconfig

import (
	"context"
	"slices"
	"strings"
)

// ConfigFileEnv is the environment variable that selects the Databricks config file. An agent
// entry pins its value when this CLI sees it set, and the MCP binding reads the same variable.
// A write that does not pin it removes it; other variables are the user's.
const ConfigFileEnv = "DATABRICKS_CONFIG_FILE"

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

// The flags that `aitools mcp serve` takes, which an entry carries to name its binding.
const (
	ServiceFlag = "--service"
	ProfileFlag = "--profile"
)

// EnvKeys are the entry keys that hold environment variables: env for most agents, and environment for OpenCode.
var EnvKeys = []string{"env", "environment"}

// EntryEnv returns the environment variables of an entry, under the first of EnvKeys that holds an object. It returns
// nil when the entry has none.
func EntryEnv(entry map[string]any) map[string]any {
	for _, key := range EnvKeys {
		if vars, ok := entry[key].(map[string]any); ok {
			return vars
		}
	}
	return nil
}

// IsDatabricksEntry reports whether entry runs `databricks aitools mcp serve`,
// which is how we recognize entries we wrote.
func IsDatabricksEntry(entry map[string]any) bool {
	argv, ok := Argv(entry)
	if !ok {
		return false
	}
	args := argv[1:]
	return IsDatabricksBinary(argv[0]) &&
		len(args) >= 3 && args[0] == "aitools" && args[1] == "mcp" && args[2] == "serve"
}

// IsDatabricksBinary reports whether cmd names the databricks binary, as a bare
// name or a path.
func IsDatabricksBinary(cmd string) bool {
	return IsBinary(cmd, "databricks", "databricks.exe")
}

// IsBinary reports whether cmd, a bare name or a path, has one of the given
// base names, ignoring case as Windows does. Both separators are accepted so
// that a Windows path is recognized on any host.
func IsBinary(cmd string, names ...string) bool {
	base := cmd[strings.LastIndexAny(cmd, `/\`)+1:]
	return slices.ContainsFunc(names, func(name string) bool {
		return strings.EqualFold(base, name)
	})
}

// ValidServerName reports whether name may be used as an MCP server name: one or
// more ASCII letters, digits, '-' or '_'. That is the strictest set of names the
// agents accept, so a name that passes is valid for every agent.
func ValidServerName(name string) bool {
	if name == "" {
		return false
	}
	for i := range len(name) {
		c := name[i]
		switch {
		case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9', c == '-', c == '_':
		default:
			return false
		}
	}
	return true
}
