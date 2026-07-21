package genieclicmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/env"
)

// defaultHarness is the coding agent genie-cli drives unless the user selects
// another with --harness. It matches an agent name ucode understands and (for
// codex) the agent registry name in libs/aitools/agents.
const defaultHarness = "codex"

// harnessCodex and harnessOpenCode are the harnesses genie-cli primes with the
// Databricks system prompt. Any other ucode agent name is still accepted and
// launched, just without prompt injection (see buildInjection).
const (
	harnessCodex    = "codex"
	harnessOpenCode = "opencode"
)

// injection describes how the Databricks system prompt is delivered to a
// harness for one session. Delivery differs per agent: Codex takes an inline
// `-c` override, OpenCode reads an instructions file referenced through an
// environment variable. Fields are empty when a harness gets no prompt.
type injection struct {
	// forwardArgs are appended after the agent's "--" separator (Codex).
	forwardArgs []string
	// env are extra KEY=value entries for the launched process (OpenCode).
	env []string
}

// buildInjection renders the system prompt for the harness and returns how to
// deliver it for this session, without writing to the user's agent config.
//
//   - codex: inline `-c developer_instructions=<toml>` (additive developer-role
//     message).
//   - opencode: the prompt is written to a genie-cli-managed file and referenced
//     via OPENCODE_CONFIG_CONTENT={"instructions":["<file>"]}, whose entries
//     OpenCode appends to the model context. Uses an env var (not a flag)
//     because OpenCode has no inline system-prompt flag.
//   - any other harness: no injection (returns a zero injection, nil error).
func buildInjection(ctx context.Context, harness, host, profile string) (injection, error) {
	prompt := buildSystemPrompt(host, profile)

	switch harness {
	case harnessCodex:
		return injection{forwardArgs: []string{"-c", "developer_instructions=" + tomlQuote(prompt)}}, nil
	case harnessOpenCode:
		path, err := writeOpenCodePromptFile(ctx, prompt)
		if err != nil {
			return injection{}, err
		}
		content, err := json.Marshal(map[string]any{"instructions": []string{path}})
		if err != nil {
			return injection{}, fmt.Errorf("failed to encode OpenCode config: %w", err)
		}
		return injection{env: []string{"OPENCODE_CONFIG_CONTENT=" + string(content)}}, nil
	default:
		return injection{}, nil
	}
}

// openCodePromptPath returns the genie-cli-managed file that holds the OpenCode
// instructions. It lives under the user's home in a stable, overwritten
// location so there is no temp file to clean up after the exec handoff.
func openCodePromptPath(ctx context.Context) (string, error) {
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".databricks", "genie-cli", "opencode-instructions.md"), nil
}

// writeOpenCodePromptFile writes the prompt to the managed path and returns it.
func writeOpenCodePromptFile(ctx context.Context, prompt string) (string, error) {
	path, err := openCodePromptPath(ctx)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("failed to create genie-cli directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(prompt), 0o644); err != nil {
		return "", fmt.Errorf("failed to write the OpenCode system prompt: %w", err)
	}
	return path, nil
}
