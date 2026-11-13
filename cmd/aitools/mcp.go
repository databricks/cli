package aitools

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/aitools/mcp/agentconfig"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
)

// Error categories that the telemetry enum does not define yet. The JSON output reports them by name. The
// binding categories come from mcp.BindCategory, which has the same names.
const (
	categoryAgentConfigInvalid protos.AitoolsErrorCategory = "AGENT_CONFIG_INVALID"
	categoryNameConflict       protos.AitoolsErrorCategory = "MCP_NAME_CONFLICT"
	categoryServiceNotFound    protos.AitoolsErrorCategory = "MCP_SERVICE_NOT_FOUND"
	categoryNoProfile          protos.AitoolsErrorCategory = "NO_PROFILE"
	categoryProfileRequired    protos.AitoolsErrorCategory = "PROFILE_REQUIRED"
	categoryPermissionDenied   protos.AitoolsErrorCategory = "MCP_PERMISSION_DENIED"
	categoryLocationNotFound   protos.AitoolsErrorCategory = "MCP_LOCATION_NOT_FOUND"
	categorySignInTimeout      protos.AitoolsErrorCategory = "SIGN_IN_TIMEOUT"
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
	cmd.AddCommand(NewMcpListCmd())
	cmd.AddCommand(NewMcpRemoveCmd())
	cmd.AddCommand(NewMcpLoginCmd())
	cmd.AddCommand(NewMcpLogoutCmd())
	cmd.AddCommand(NewMcpServeCmd())
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

// errorCategory classifies an error by its typed errors only, never by its text. Failures without a
// category of their own fall through to the install classification, as in the other aitools commands.
func errorCategory(err error) protos.AitoolsErrorCategory {
	if ce, ok := errors.AsType[*cliError](err); ok {
		return ce.category
	}
	// An uncategorized binding failure falls through, so that its cause (a filesystem or network error) is classified.
	if be, ok := errors.AsType[*mcp.BindError](err); ok && be.Category != mcp.BindUncategorized {
		return protos.AitoolsErrorCategory(be.Category)
	}
	switch {
	case errors.Is(err, agentconfig.ErrConfigInvalid):
		return categoryAgentConfigInvalid
	case errors.Is(err, agentconfig.ErrNotOwned):
		return categoryNameConflict
	}
	return classifyInstallError(err)
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
		return "", failure(categoryNoProfile, nil, "no workspace profiles in %s. Run: databricks auth login --host <url> --profile <name>", filepath.ToSlash(path))
	case 1:
		return profiles[0].Name, nil
	}
	return "", failure(categoryProfileRequired, &profileChoice{profiles: profiles}, "several workspace profiles exist; pass --profile. Found: %s", strings.Join(profiles.Names(), ", "))
}

// profileFlag returns the root --profile value, or "" when it is not set.
func profileFlag(cmd *cobra.Command) string {
	return cmd.Flag("profile").Value.String()
}
