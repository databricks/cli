package genieclicmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/execv"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/process"
	"github.com/databricks/databricks-sdk-go/config"
)

const (
	// ucodeBin is the launcher that wires a coding agent to the Databricks AI
	// Gateway. It, the agent binary, and the underlying model routing are
	// implementation details hidden behind the genie-cli command.
	ucodeBin = "ucode"
	// uvBin installs ucode; it is the only supported installer for the launcher
	// (a Python tool published as a uv tool). See ucodeInstallSpec.
	uvBin = "uv"
	// ucodeInstallSpec is the source uv installs the launcher from.
	// https://github.com/databricks/ucode
	ucodeInstallSpec = "git+https://github.com/databricks/ucode"

	// mcpLocation is the Unity Catalog schema whose MCP services are registered
	// for the agent. system.ai hosts the built-in Databricks MCP services.
	mcpLocation = "system.ai"
)

// lookPath is a package-level seam so tests can resolve binaries without
// depending on the host PATH.
var lookPath = exec.LookPath

// launch replaces the current process with the given command; a package-level
// seam so tests can assert the launch argv without exec'ing.
var launch = execv.Execv

// ensureUcode makes sure the launcher is installed, installing it through uv if
// it is missing. It returns the absolute path to the launcher binary.
func ensureUcode(ctx context.Context) (string, error) {
	if path, err := lookPath(ucodeBin); err == nil {
		return path, nil
	}

	// The launcher is only distributed as a uv tool, so uv is a hard prerequisite
	// rather than something we can work around.
	if _, err := lookPath(uvBin); err != nil {
		return "", fmt.Errorf("genie-cli requires uv to install its runtime, but uv was not found on PATH\n"+
			"Install uv (https://docs.astral.sh/uv/getting-started/installation) and re-run, "+
			"or install the runtime manually with: uv tool install %s", ucodeInstallSpec)
	}

	log.Infof(ctx, "Installing the genie-cli runtime with uv")
	if err := process.Forwarded(ctx, []string{uvBin, "tool", "install", ucodeInstallSpec}, os.Stdin, os.Stdout, os.Stderr); err != nil {
		return "", fmt.Errorf("failed to install the genie-cli runtime: %w", err)
	}

	// uv installs into its tool bin dir (e.g. ~/.local/bin), which may not be on
	// the current process's PATH even after a successful install.
	path, err := lookPath(ucodeBin)
	if err != nil {
		return "", fmt.Errorf("the genie-cli runtime was installed but %q is not on PATH\n"+
			"Add uv's tool bin directory to PATH (run: uv tool update-shell) and re-run", ucodeBin)
	}
	return path, nil
}

// configureArgs builds the launcher's non-interactive configure argv for the
// chosen harness and resolved workspace config. The launcher needs an explicit
// workspace identity or it prompts interactively, which would break the
// non-interactive setup.
func configureArgs(ucodePath, harness string, cfg *config.Config) ([]string, error) {
	args := []string{ucodePath, "configure", "--agents", harness, "--skip-validate", "--skip-upgrade"}

	switch {
	case cfg.Profile != "":
		args = append(args, "--profiles", cfg.Profile)
		// --use-pat reuses the profile's stored token with no interactive login;
		// it is only valid for PAT profiles and requires --profiles.
		if cfg.AuthType == auth.AuthTypePat {
			args = append(args, "--use-pat")
		}
	case cfg.Host != "":
		args = append(args, "--workspaces", cfg.Host)
	default:
		return nil, errors.New("no Databricks profile or host resolved; run `databricks auth login` first")
	}
	return args, nil
}

// configureAgent points the harness at this workspace's AI Gateway. This also
// installs the agent binary and the Databricks CLI the launcher relies on.
func configureAgent(ctx context.Context, ucodePath, harness string, cfg *config.Config) error {
	args, err := configureArgs(ucodePath, harness, cfg)
	if err != nil {
		return err
	}
	if err := process.Forwarded(ctx, args, os.Stdin, os.Stdout, os.Stderr); err != nil {
		return fmt.Errorf("failed to configure the Databricks agent: %w", err)
	}
	return nil
}

// registerMCP registers the Databricks MCP services from the configured Unity
// Catalog location for the agent, so the session has Databricks MCP tools. It
// is best-effort: a workspace without MCP services in that schema (or an
// unreachable one) should not block the agent, so failures are logged and
// swallowed, matching the plugin-refresh policy.
func registerMCP(ctx context.Context, ucodePath string) {
	args := []string{ucodePath, "configure", "mcp", "--location", mcpLocation}
	if err := process.Forwarded(ctx, args, os.Stdin, os.Stdout, os.Stderr); err != nil {
		log.Warnf(ctx, "Skipping Databricks MCP setup: %v", err)
	}
}

// launchAgent replaces the current process with the harness session.
// --skip-preflight trusts the configure step we just ran and skips a redundant
// auth + gateway re-validation. The system prompt is delivered per the harness's
// injection (forwarded args and/or extra env); userArgs are forwarded to the
// underlying agent after ours.
func launchAgent(ctx context.Context, ucodePath, harness string, inj injection, userArgs []string) error {
	// Everything after "--" is forwarded verbatim to the underlying agent.
	forwarded := append([]string{}, inj.forwardArgs...)
	forwarded = append(forwarded, userArgs...)

	args := []string{ucodePath, harness, "--skip-preflight"}
	if len(forwarded) > 0 {
		args = append(args, "--")
		args = append(args, forwarded...)
	}
	return launch(execv.Options{
		Args: args,
		Env:  append(envSlice(ctx), inj.env...),
	})
}

// envSlice returns the process environment as a "KEY=value" slice, pulled
// through libs/env so tests can override it per context.
func envSlice(ctx context.Context) []string {
	all := env.All(ctx)
	out := make([]string, 0, len(all))
	for k, v := range all {
		out = append(out, k+"="+v)
	}
	return out
}
