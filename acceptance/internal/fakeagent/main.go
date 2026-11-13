// Command fakeagent stands in for the claude, codex, gemini and copilot CLIs in the aitools mcp acceptance tests. The
// harness builds it once per run and copies it under each agent's name, and a test puts those on PATH, so no test runs
// a CLI from the host.
//
// Run as an agent, it appends its argv to the file named by $FAKE_AGENT_LOG, then makes the change through agentconfig's
// direct writer, which edits the agent's user-scope config file as the agent's CLI would. The stand-in clears its PATH
// first, so the writer finds no agent CLI there and writes the file itself, in the writer's layout. Only the add and
// remove commands that the agents' CLI wrappers use are accepted; any other command fails, so a test that runs an
// unexpected command fails too.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/mcp/agentconfig"
	"github.com/databricks/cli/libs/env"
)

// agentOf maps the name a stand-in runs under to the agent whose config it edits.
var agentOf = map[string]string{
	"claude":  agents.NameClaudeCode,
	"codex":   agents.NameCodex,
	"gemini":  agents.NameGemini,
	"copilot": agents.NameCopilot,
}

func main() {
	ctx := context.Background()
	program := strings.TrimSuffix(strings.ToLower(filepath.Base(os.Args[0])), ".exe")
	args := os.Args[1:]
	if err := logArgv(ctx, append([]string{program}, args...)); err != nil {
		fail(err)
	}
	if err := run(ctx, program, args); err != nil {
		fail(err)
	}
}

// fail reports err on stderr and exits 1, as a CLI does when a command fails.
func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

// logArgv appends one invocation to the log: the program on a line of its own, then each argument on a line of its own,
// indented. The arguments are not quoted, so a path or JSON in an argument reads as it was written.
func logArgv(ctx context.Context, argv []string) error {
	f, err := os.OpenFile(env.Get(ctx, "FAKE_AGENT_LOG"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString(argv[0] + "\n")
	for _, arg := range argv[1:] {
		b.WriteString("\t" + arg + "\n")
	}
	_, err = f.WriteString(b.String())
	return errors.Join(err, f.Close())
}

// run dispatches one invocation to the handler of the agent that program stands for. It empties PATH first, so that the
// writer does not run a CLI.
func run(ctx context.Context, program string, args []string) error {
	name, ok := agentOf[program]
	if !ok {
		return fmt.Errorf("fakeagent: no agent CLI named %q", program)
	}
	// The writer runs an agent CLI that it finds on PATH. The stand-ins are on PATH, so it must find none.
	if err := os.Setenv("PATH", ""); err != nil {
		return err
	}
	agent := agents.ByName(name)
	switch program {
	case "claude":
		return claude(ctx, agent, args)
	case "gemini":
		return gemini(ctx, agent, args)
	default:
		return codex(ctx, agent, program, args)
	}
}

// claude handles `mcp add-json --scope user <name> <json>` and `mcp remove --scope user <name>`.
func claude(ctx context.Context, agent *agents.Agent, args []string) error {
	switch {
	case len(args) == 6 && args[0] == "mcp" && args[1] == "add-json" && args[2] == "--scope" && args[3] == "user":
		var entry struct {
			Command string            `json:"command"`
			Args    []string          `json:"args"`
			Env     map[string]string `json:"env"`
		}
		if err := json.Unmarshal([]byte(args[5]), &entry); err != nil {
			return err
		}
		return upsert(ctx, agent, args[4], entry.Command, entry.Args, entry.Env)
	case len(args) == 5 && args[0] == "mcp" && args[1] == "remove" && args[2] == "--scope" && args[3] == "user":
		return remove(ctx, agent, args[4])
	}
	return unexpected("claude", args)
}

// codex handles `mcp add <name> [--env K=V]... -- <command> <args>...` and `mcp remove <name>`. copilot takes the same
// forms, and `--tools *` after the name.
func codex(ctx context.Context, agent *agents.Agent, program string, args []string) error {
	switch {
	case len(args) >= 3 && args[0] == "mcp" && args[1] == "add":
		name, rest := args[2], args[3:]
		if program == "copilot" && len(rest) >= 2 && rest[0] == "--tools" && rest[1] == "*" {
			rest = rest[2:]
		}
		env := map[string]string{}
		for len(rest) >= 2 && rest[0] == "--env" {
			key, value, err := splitVar(rest[1])
			if err != nil {
				return err
			}
			env[key] = value
			rest = rest[2:]
		}
		if len(rest) < 2 || rest[0] != "--" {
			return unexpected(program, args)
		}
		return upsert(ctx, agent, name, rest[1], rest[2:], env)
	case len(args) == 3 && args[0] == "mcp" && args[1] == "remove":
		return remove(ctx, agent, args[2])
	}
	return unexpected(program, args)
}

// gemini handles `mcp add --scope user [-e K=V]... <name> <command> -- <args>...` and `mcp remove --scope user <name>`.
func gemini(ctx context.Context, agent *agents.Agent, args []string) error {
	switch {
	case len(args) >= 5 && args[0] == "mcp" && args[1] == "add" && args[2] == "--scope" && args[3] == "user":
		rest := args[4:]
		env := map[string]string{}
		for len(rest) >= 2 && rest[0] == "-e" {
			key, value, err := splitVar(rest[1])
			if err != nil {
				return err
			}
			env[key] = value
			rest = rest[2:]
		}
		if len(rest) < 3 || rest[2] != "--" {
			return unexpected("gemini", args)
		}
		return upsert(ctx, agent, rest[0], rest[1], rest[3:], env)
	case len(args) == 5 && args[0] == "mcp" && args[1] == "remove" && args[2] == "--scope" && args[3] == "user":
		return remove(ctx, agent, args[4])
	}
	return unexpected("gemini", args)
}

// upsert adds the server through the direct writer, which writes the entry as the agent's CLI would.
func upsert(ctx context.Context, agent *agents.Agent, name, cmd string, args []string, env map[string]string) error {
	_, _, err := agentconfig.Upsert(ctx, agent, name, cmd, args, env, false)
	return err
}

// remove deletes the server from the config file that the agent reads now.
func remove(ctx context.Context, agent *agents.Agent, name string) error {
	path, err := agentconfig.Path(ctx, agent)
	if err != nil {
		return err
	}
	_, err = agentconfig.RemoveAt(ctx, agent, path, name)
	return err
}

// splitVar splits a KEY=VALUE argument at its first "=".
func splitVar(s string) (string, string, error) {
	key, value, ok := strings.Cut(s, "=")
	if !ok {
		return "", "", fmt.Errorf("fakeagent: %q is not KEY=VALUE", s)
	}
	return key, value, nil
}

// unexpected is the error for a command that the stand-in does not accept, so that a test which runs one fails.
func unexpected(program string, args []string) error {
	return fmt.Errorf("fakeagent: %s does not take %q", program, args)
}
