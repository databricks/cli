package mcp_test

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/shellquote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEntryServes reports whether an agent entry is a Databricks entry bound to a service, profile and config file, in
// each layout the agents write: env or environment, and an args list or a command array.
func TestEntryServes(t *testing.T) {
	ctx := t.Context()
	cfgPath := bindingTestWriteConfig(t, bindingTestProfileContents(bindingTestToken))
	resolved := bindingTestResolved(t, cfgPath)
	serve := []string{"aitools", "mcp", "serve", "--profile", bindingTestProfile, "--service", bindingTestService, "--agent", "x"}

	claude := map[string]any{"type": "stdio", "command": "databricks", "args": anyList(serve), "env": map[string]any{"DATABRICKS_CONFIG_FILE": cfgPath}}
	codex := map[string]any{"command": "databricks", "args": anyList(serve)}
	opencode := map[string]any{"type": "local", "command": anyList(append([]string{"databricks"}, serve...)), "environment": map[string]any{"DATABRICKS_CONFIG_FILE": cfgPath}}
	openCodeUnpinned := map[string]any{"type": "local", "command": anyList(append([]string{"databricks"}, serve...))}

	tests := []struct {
		name       string
		entry      map[string]any
		service    string
		profile    string
		configFile string
		wantServes bool
	}{
		{name: "claude env pinned to the file", entry: claude, service: bindingTestService, profile: bindingTestProfile, configFile: resolved, wantServes: true},
		{name: "opencode environment pinned to the file", entry: opencode, service: bindingTestService, profile: bindingTestProfile, configFile: resolved, wantServes: true},
		{name: "service is case-insensitive", entry: claude, service: "System.AI.GitHub", profile: bindingTestProfile, configFile: resolved, wantServes: true},
		{name: "unpinned matches the default file", entry: codex, service: bindingTestService, profile: bindingTestProfile, configFile: defaultFile(t), wantServes: true},
		{name: "unpinned opencode matches the default file", entry: openCodeUnpinned, service: bindingTestService, profile: bindingTestProfile, configFile: defaultFile(t), wantServes: true},
		{name: "profile named like a flag", entry: map[string]any{"command": "databricks", "args": anyList([]string{"aitools", "mcp", "serve", "--profile", "--service", "--service", bindingTestService, "--agent", "x"})}, service: bindingTestService, profile: "--service", configFile: defaultFile(t), wantServes: true},
		{name: "repeated profile takes its last value", entry: map[string]any{"command": "databricks", "args": anyList([]string{"aitools", "mcp", "serve", "--profile", "other", "--profile", bindingTestProfile, "--service", bindingTestService, "--agent", "x"})}, service: bindingTestService, profile: bindingTestProfile, configFile: defaultFile(t), wantServes: true},
		{name: "repeated profile does not match its earlier value", entry: map[string]any{"command": "databricks", "args": anyList([]string{"aitools", "mcp", "serve", "--profile", bindingTestProfile, "--profile", "other", "--service", bindingTestService, "--agent", "x"})}, service: bindingTestService, profile: bindingTestProfile, configFile: defaultFile(t), wantServes: false},
		{name: "profile in the equals form", entry: map[string]any{"command": "databricks", "args": anyList([]string{"aitools", "mcp", "serve", "--profile=" + bindingTestProfile, "--service", bindingTestService, "--agent", "x"})}, service: bindingTestService, profile: bindingTestProfile, configFile: defaultFile(t), wantServes: true},
		{name: "pinned file differs", entry: claude, service: bindingTestService, profile: bindingTestProfile, configFile: defaultFile(t), wantServes: false},
		{name: "unpinned differs from another file", entry: codex, service: bindingTestService, profile: bindingTestProfile, configFile: resolved, wantServes: false},
		{name: "other profile", entry: claude, service: bindingTestService, profile: "other", configFile: resolved, wantServes: false},
		{name: "other service", entry: claude, service: "system.ai.genie_one_mcp", profile: bindingTestProfile, configFile: resolved, wantServes: false},
		{
			name:    "not a Databricks entry",
			entry:   map[string]any{"command": "ug", "args": []any{"mcp-proxy"}},
			service: bindingTestService, profile: bindingTestProfile, configFile: resolved, wantServes: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mcp.EntryServes(ctx, tt.entry, tt.service, tt.profile, tt.configFile)
			require.NoError(t, err)
			assert.Equal(t, tt.wantServes, got)
		})
	}
}

// anyList converts a string list to the []any that a decoded JSON or TOML entry holds.
func anyList(values []string) []any {
	out := make([]any, 0, len(values))
	for _, v := range values {
		out = append(out, v)
	}
	return out
}

// defaultFile returns the default config file, as the command resolves it.
func defaultFile(t *testing.T) string {
	t.Helper()
	path, err := mcp.ResolveConfigFile(t.Context(), "")
	require.NoError(t, err)
	return path
}

// adoptEntry returns an unmanaged serve entry with the given environment, which is omitted when nil.
func adoptEntry(env map[string]any, args ...any) map[string]any {
	entry := map[string]any{"command": "databricks", "args": append([]any{"aitools", "mcp", "serve"}, args...)}
	if env != nil {
		entry["env"] = env
	}
	return entry
}

// TestAdoptCommand checks that an unmanaged serve entry names the install that adopts it: the service and profile its
// arguments name, bound to the config file the entry serves. An entry without a service or profile names no command.
func TestAdoptCommand(t *testing.T) {
	ctx := t.Context()
	cfgPath := filepath.Join(t.TempDir(), "databrickscfg")
	pinned, err := mcp.ResolveConfigFile(ctx, cfgPath)
	require.NoError(t, err)
	defaultFile, err := mcp.ResolveConfigFile(ctx, "")
	require.NoError(t, err)
	flags := []any{"--profile", "logfood", "--service", "system.ai.github", "--agent", "claude-code"}

	// An unpinned entry binds to the default file, which a command that runs with no config file override binds to.
	t.Setenv("DATABRICKS_CONFIG_FILE", "")
	assert.Equal(t, "databricks aitools mcp add --names system.ai.github --profile logfood --agents claude-code",
		mcp.AdoptCommand(ctx, adoptEntry(nil, flags...), "claude-code"))
	// A shell that selects another file makes the command name the default file explicitly.
	t.Setenv("DATABRICKS_CONFIG_FILE", cfgPath)
	assert.Equal(t, "DATABRICKS_CONFIG_FILE="+shellquote.BashArg(filepath.ToSlash(defaultFile))+" databricks aitools mcp add --names system.ai.github --profile logfood --agents claude-code",
		mcp.AdoptCommand(ctx, adoptEntry(nil, flags...), "claude-code"))
	assert.Equal(t, "DATABRICKS_CONFIG_FILE="+shellquote.BashArg(filepath.ToSlash(pinned))+" databricks aitools mcp add --names system.ai.github --profile logfood --agents claude-code",
		mcp.AdoptCommand(ctx, adoptEntry(map[string]any{"DATABRICKS_CONFIG_FILE": cfgPath}, flags...), "claude-code"))
	assert.Empty(t, mcp.AdoptCommand(ctx, adoptEntry(nil, "--profile", "logfood"), "claude-code"))
	assert.Empty(t, mcp.AdoptCommand(ctx, adoptEntry(nil, "--service", "system.ai.github"), "claude-code"))
}
