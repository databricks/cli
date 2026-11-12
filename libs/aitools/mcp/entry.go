package mcp

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/aitools/mcpconfig"
)

// EntryServes reports whether entry is a Databricks entry that serves service for profileName through configFile, which
// the caller resolved with ResolveConfigFile. Service names are compared without case. The entry's config file is the one
// it pins in its environment, or the default file when it pins none, resolved as Bind resolves it, so that a pin named
// "~/..." or a path through a symlink matches the file it names. An entry whose pin cannot be resolved is an error, since
// it might be the one that serves the binding.
func EntryServes(ctx context.Context, entry map[string]any, service, profileName, configFile string) (bool, error) {
	if !mcpconfig.IsDatabricksEntry(entry) {
		return false, nil
	}
	argv, _ := mcpconfig.Argv(entry)
	if !strings.EqualFold(flagValue(argv, mcpconfig.ServiceFlag), service) || flagValue(argv, mcpconfig.ProfileFlag) != profileName {
		return false, nil
	}
	pinned, _ := mcpconfig.EntryEnv(entry)[mcpconfig.ConfigFileEnv].(string)
	path, err := ResolveConfigFile(ctx, pinned)
	if err != nil {
		return false, fmt.Errorf("MCP server entry for %s on profile %s: %w", service, profileName, err)
	}
	return SamePath(path, configFile), nil
}

// flagValue returns the value that follows flag in argv, or "" when flag is absent.
func flagValue(argv []string, flag string) string {
	if i := slices.Index(argv, flag); i >= 0 && i+1 < len(argv) {
		return argv[i+1]
	}
	return ""
}
