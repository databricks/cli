// Package agentlauncher installs the on-PATH launchers that start a coding agent
// through "databricks ssh agent-shim <agent>" on the remote compute. It is shared
// by the server, which installs the launchers, and the client, whose shim they run.
package agentlauncher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/databricks/cli/libs/atomicfile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/shellquote"
	"github.com/databricks/cli/libs/telemetry/protos"
)

const (
	// RootDir is the home-relative directory holding files/dependencies for the shim.
	RootDir = ".agent-shim"

	// BinDir holds the per-agent launchers.
	BinDir = RootDir + "/bin"

	// WorkspaceHomeEnv passes the user's workspace home to the shim for the agent context.
	WorkspaceHomeEnv = "DATABRICKS_WORKSPACE_HOME"

	// ContextFile holds the Databricks session context, under RootDir.
	ContextFile = "agent-system-context.md"
)

type AgentSpec struct {
	Name          string
	TelemetryName protos.SshAgentShimAgent
	// ContextArg puts the system context where the agent reads it, returning the
	// argv to pass to the agent binary.
	ContextArg func(home, systemContext string) ([]string, error)
}

// Currently limited to the agents whose Unity Gateway CLI command supports --workspace
var SupportedAgents = []AgentSpec{
	{Name: "claude", TelemetryName: protos.SshAgentShimAgentClaudeCode, ContextArg: claudeContextArg},
	{Name: "codex", TelemetryName: protos.SshAgentShimAgentCodex, ContextArg: codexContextArg},
}

func claudeContextArg(home, systemContext string) ([]string, error) {
	// Keep the scratch context file under the shim's own directory so it never
	// clobbers an unrelated file the user happens to have in their home.
	dir := filepath.Join(home, RootDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create %s: %w", dir, err)
	}
	f := filepath.Join(dir, ContextFile)
	if err := os.WriteFile(f, []byte(systemContext), 0o644); err != nil {
		return nil, fmt.Errorf("failed to write agent context file: %w", err)
	}
	return []string{"--append-system-prompt-file", f}, nil
}

func codexContextArg(home, systemContext string) ([]string, error) {
	// The override value is parsed as TOML, so encode the multi-line context as
	// a TOML string rather than relying on codex's raw-string fallback.
	// https://github.com/openai/codex/blob/rust-v0.145.0/codex-rs/utils/cli/src/config_override.rs
	kv, err := toml.Marshal(map[string]string{"developer_instructions": systemContext})
	if err != nil {
		return nil, fmt.Errorf("failed to encode agent context: %w", err)
	}
	return []string{"-c", strings.TrimSpace(string(kv))}, nil
}

func AgentByName(name string) (AgentSpec, bool) {
	for _, a := range SupportedAgents {
		if a.Name == name {
			return a, true
		}
	}
	return AgentSpec{}, false
}

// Install writes a launcher per supported agent to $HOME/BinDir. Each launcher runs
// "<binary> ssh agent-shim <agent>" and forwards wsHome for the agent context.
// Each launcher is written atomically: bash reads a script incrementally, so
// truncating the live file in place could cut off a launcher that a running
// session is still executing. A failed launcher doesn't stop the others from
// being written; the returned error names every agent that failed.
func Install(ctx context.Context, binary, wsHome string) error {
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	var errs []error
	for _, agent := range SupportedAgents {
		path := filepath.Join(home, BinDir, agent.Name)
		err := atomicfile.Write(path, []byte(script(binary, wsHome, agent.Name)), 0o755, atomicfile.MkDir(0o755))
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to write the %s launcher: %w", agent.Name, err))
		}
	}
	return errors.Join(errs...)
}

// script returns one agent's launcher. binary and wsHome are shell-quoted so a
// path or username with shell metacharacters can't break (or inject into) it.
func script(binary, wsHome, agent string) string {
	return fmt.Sprintf(`#!/usr/bin/env bash
# "%[1]s" launcher from "databricks ssh connect"; delegates to "ssh agent-shim %[1]s".
export %[2]s=%[3]s
exec %[4]s ssh agent-shim %[1]s "$@"
`, agent, WorkspaceHomeEnv, shellquote.BashArg(wsHome), shellquote.BashArg(binary))
}
