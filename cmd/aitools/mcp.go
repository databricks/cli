package aitools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/aitools/mcp/agentconfig"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
)

// defaultMCPLocation is the schema that ships the Databricks managed MCP services.
const defaultMCPLocation = "system.ai"

// Error categories that the telemetry enum does not define yet.
// The JSON output reports them by name.
const (
	categoryAgentConfigInvalid protos.AitoolsErrorCategory = "AGENT_CONFIG_INVALID"
	categoryAuthFailed         protos.AitoolsErrorCategory = "AUTH_FAILED"
	categoryProfileNotFound    protos.AitoolsErrorCategory = "PROFILE_NOT_FOUND"
	categoryProfileHostChanged protos.AitoolsErrorCategory = "PROFILE_HOST_CHANGED"
	categoryServiceNotFound    protos.AitoolsErrorCategory = "MCP_SERVICE_NOT_FOUND"
)

// NewMcpCmd returns the `aitools mcp` group. It stays hidden until the whole MCP
// stack has landed, so a release never ships `add` without `serve`.
func NewMcpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "mcp",
		Short:  "Manage Databricks managed MCP servers for coding agents",
		Hidden: true,
	}
	cmd.AddCommand(NewMcpAddCmd())
	return cmd
}

// cliError is a failure with the message shown to the user, the category reported in JSON
// and telemetry, and the error it wraps when there is one.
type cliError struct {
	category protos.AitoolsErrorCategory
	message  string
	cause    error
}

func (e *cliError) Error() string { return e.message }

func (e *cliError) Unwrap() error { return e.cause }

// failure returns a cliError. cause is nil when the message is the whole failure.
func failure(category protos.AitoolsErrorCategory, cause error, format string, args ...any) error {
	return &cliError{category: category, message: fmt.Sprintf(format, args...), cause: cause}
}

// errorCategory classifies an error by its typed errors only, never by its text.
// Errors without a typed category are UNCATEGORIZED.
func errorCategory(err error) protos.AitoolsErrorCategory {
	if ce, ok := errors.AsType[*cliError](err); ok {
		return ce.category
	}
	if errors.Is(err, agentconfig.ErrConfigInvalid) {
		return categoryAgentConfigInvalid
	}
	return protos.AitoolsErrorCategoryUncategorized
}

// resolveProfile picks the profile a non-interactive command binds to: the
// --profile flag, then DATABRICKS_CONFIG_PROFILE, then the default profile, then the only
// workspace profile. It never prompts.
func resolveProfile(ctx context.Context, flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if name := env.Get(ctx, "DATABRICKS_CONFIG_PROFILE"); name != "" {
		return name, nil
	}
	if name := databrickscfg.ResolveDefaultProfile(ctx); name != "" {
		return name, nil
	}

	profiler := profile.GetProfiler(ctx)
	path, err := profiler.GetPath(ctx)
	if err != nil {
		return "", err
	}
	profiles, err := profiler.LoadProfiles(ctx, profile.MatchWorkspaceProfiles)
	if err != nil && !errors.Is(err, profile.ErrNoConfiguration) {
		return "", err
	}
	switch len(profiles) {
	case 0:
		return "", fmt.Errorf("no workspace profiles in %s. Run: databricks auth login --host <url> --profile <name>", path)
	case 1:
		return profiles[0].Name, nil
	}
	return "", fmt.Errorf("several workspace profiles exist; pass --profile. Found: %s", strings.Join(profiles.Names(), ", "))
}

// profileFlag returns the root --profile value, or "" when it is not set.
func profileFlag(cmd *cobra.Command) string {
	if f := cmd.Flag("profile"); f != nil {
		return f.Value.String()
	}
	return ""
}
